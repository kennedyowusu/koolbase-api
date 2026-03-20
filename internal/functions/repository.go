package functions

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrFunctionNotFound = errors.New("function not found")
	ErrTriggerNotFound  = errors.New("trigger not found")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetProjectByAPIKey(ctx context.Context, apiKey string) (projectID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT project_id FROM environments WHERE public_key = $1`, apiKey,
	).Scan(&projectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("invalid api key")
		}
		return "", err
	}
	return projectID, nil
}

func (r *Repository) AuthorizeProject(ctx context.Context, projectID, orgID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects WHERE id = $1 AND org_id = $2`,
		projectID, orgID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) DeployFunction(ctx context.Context, projectID, name, code string, timeoutMs int) (*Function, error) {
	now := time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Get next version — check error
	var maxVersion int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM project_functions WHERE project_id = $1 AND name = $2`,
		projectID, name,
	).Scan(&maxVersion); err != nil {
		return nil, err
	}
	nextVersion := maxVersion + 1

	// Deactivate previous active version
	if _, err := tx.Exec(ctx,
		`UPDATE project_functions SET is_active = FALSE WHERE project_id = $1 AND name = $2 AND is_active = TRUE`,
		projectID, name,
	); err != nil {
		return nil, err
	}

	// Insert new active version
	var fn Function
	if err := tx.QueryRow(ctx,
		`INSERT INTO project_functions
		 (project_id, name, code, version, is_active, timeout_ms, last_deployed_at)
		 VALUES ($1, $2, $3, $4, TRUE, $5, $6)
		 RETURNING id, project_id, name, runtime, entry_file, code, version, is_active, timeout_ms, enabled, last_deployed_at, created_at, updated_at`,
		projectID, name, code, nextVersion, timeoutMs, now,
	).Scan(
		&fn.ID, &fn.ProjectID, &fn.Name, &fn.Runtime, &fn.EntryFile,
		&fn.Code, &fn.Version, &fn.IsActive, &fn.TimeoutMs, &fn.Enabled,
		&fn.LastDeployedAt, &fn.CreatedAt, &fn.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &fn, nil
}

func (r *Repository) GetActiveFunction(ctx context.Context, projectID, name string) (*Function, error) {
	var fn Function
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, runtime, entry_file, code, version, is_active, timeout_ms, enabled, last_deployed_at, created_at, updated_at
		 FROM project_functions
		 WHERE project_id = $1 AND name = $2 AND is_active = TRUE AND enabled = TRUE`,
		projectID, name,
	).Scan(
		&fn.ID, &fn.ProjectID, &fn.Name, &fn.Runtime, &fn.EntryFile,
		&fn.Code, &fn.Version, &fn.IsActive, &fn.TimeoutMs, &fn.Enabled,
		&fn.LastDeployedAt, &fn.CreatedAt, &fn.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFunctionNotFound
		}
		return nil, err
	}
	return &fn, nil
}

func (r *Repository) ListFunctions(ctx context.Context, projectID string) ([]Function, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, runtime, entry_file, code, version, is_active, timeout_ms, enabled, last_deployed_at, created_at, updated_at
		 FROM project_functions
		 WHERE project_id = $1 AND is_active = TRUE
		 ORDER BY name ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fns := []Function{}
	for rows.Next() {
		var fn Function
		if err := rows.Scan(
			&fn.ID, &fn.ProjectID, &fn.Name, &fn.Runtime, &fn.EntryFile,
			&fn.Code, &fn.Version, &fn.IsActive, &fn.TimeoutMs, &fn.Enabled,
			&fn.LastDeployedAt, &fn.CreatedAt, &fn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		fns = append(fns, fn)
	}
	return fns, rows.Err()
}

func (r *Repository) DeleteFunction(ctx context.Context, projectID, name string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM project_functions WHERE project_id = $1 AND name = $2`,
		projectID, name,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrFunctionNotFound
	}
	return nil
}

func (r *Repository) InsertLog(ctx context.Context, log Log) (*Log, error) {
	err := r.db.QueryRow(ctx,
		`INSERT INTO function_logs
		 (function_id, project_id, function_version, trigger_type, event_type, collection, status, duration_ms, output, error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at`,
		log.FunctionID, log.ProjectID, log.FunctionVersion,
		log.TriggerType, log.EventType, log.Collection,
		log.Status, log.DurationMs, log.Output, log.Error,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *Repository) ListLogs(ctx context.Context, projectID, functionID string, limit int) ([]Log, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, function_id, project_id, function_version, trigger_type, event_type, collection, status, duration_ms, output, error, created_at
		 FROM function_logs
		 WHERE project_id = $1 AND ($2::text = '' OR function_id::text = $2)
		 ORDER BY created_at DESC LIMIT $3`,
		projectID, functionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []Log{}
	for rows.Next() {
		var l Log
		if err := rows.Scan(
			&l.ID, &l.FunctionID, &l.ProjectID, &l.FunctionVersion,
			&l.TriggerType, &l.EventType, &l.Collection,
			&l.Status, &l.DurationMs, &l.Output, &l.Error, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// Triggers — use function_name for version-agnostic binding
func (r *Repository) CreateTrigger(ctx context.Context, projectID, functionName, eventType, collection string) (*Trigger, error) {
	var t Trigger
	err := r.db.QueryRow(ctx,
		`INSERT INTO project_triggers (project_id, function_name, event_type, collection)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, project_id, function_name, event_type, collection, enabled, created_at`,
		projectID, functionName, eventType, collection,
	).Scan(&t.ID, &t.ProjectID, &t.FunctionName, &t.EventType, &t.Collection, &t.Enabled, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListTriggers(ctx context.Context, projectID string) ([]Trigger, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, event_type, collection, enabled, created_at
		 FROM project_triggers WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggers := []Trigger{}
	for rows.Next() {
		var t Trigger
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.FunctionName, &t.EventType, &t.Collection, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}
	return triggers, rows.Err()
}

func (r *Repository) GetTriggersForEvent(ctx context.Context, projectID, eventType, collection string) ([]Trigger, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, event_type, collection, enabled, created_at
		 FROM project_triggers
		 WHERE project_id = $1 AND event_type = $2 AND collection = $3 AND enabled = TRUE`,
		projectID, eventType, collection,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggers := []Trigger{}
	for rows.Next() {
		var t Trigger
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.FunctionName, &t.EventType, &t.Collection, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}
	return triggers, rows.Err()
}

func (r *Repository) DeleteTrigger(ctx context.Context, projectID, triggerID string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM project_triggers WHERE id = $1 AND project_id = $2`,
		triggerID, projectID,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrTriggerNotFound
	}
	return nil
}
