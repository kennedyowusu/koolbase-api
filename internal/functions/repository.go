package functions

import (
	"context"
	"errors"
	"time"

	"encoding/json"
	"math"

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

	var maxVersion int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM project_functions WHERE project_id = $1 AND name = $2`,
		projectID, name,
	).Scan(&maxVersion); err != nil {
		return nil, err
	}
	nextVersion := maxVersion + 1

	if _, err := tx.Exec(ctx,
		`UPDATE project_functions SET is_active = FALSE WHERE project_id = $1 AND name = $2 AND is_active = TRUE`,
		projectID, name,
	); err != nil {
		return nil, err
	}

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

func (r *Repository) EnqueueRetry(ctx context.Context, projectID, functionName, eventType, collection, apiKey, lastError string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO function_retry_queue
		 (project_id, function_name, event_type, collection, payload, api_key, attempt, next_retry_at, last_error)
		 VALUES ($1, $2, $3, $4, $5, $6, 0, NOW(), $7)`,
		projectID, functionName, eventType, collection, payloadJSON, apiKey, lastError,
	)
	return err
}

type RetryJob struct {
	ID           string
	ProjectID    string
	FunctionName string
	EventType    string
	Collection   string
	Payload      map[string]interface{}
	APIKey       string
	Attempt      int
	MaxAttempts  int
	LastError    string
}

func (r *Repository) GetDueRetries(ctx context.Context) ([]RetryJob, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, event_type, collection, payload, api_key, attempt, max_attempts, last_error
		 FROM function_retry_queue
		 WHERE next_retry_at <= NOW() AND attempt < max_attempts
		 ORDER BY next_retry_at ASC
		 LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []RetryJob{}
	for rows.Next() {
		var j RetryJob
		var payloadJSON []byte
		var lastError *string
		if err := rows.Scan(&j.ID, &j.ProjectID, &j.FunctionName, &j.EventType, &j.Collection,
			&payloadJSON, &j.APIKey, &j.Attempt, &j.MaxAttempts, &lastError); err != nil {
			return nil, err
		}
		if lastError != nil {
			j.LastError = *lastError
		}
		json.Unmarshal(payloadJSON, &j.Payload)
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *Repository) MarkRetrySuccess(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM function_retry_queue WHERE id = $1`, id)
	return err
}

func (r *Repository) MarkRetryFailed(ctx context.Context, id string, attempt int, lastError string, nextRetryAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE function_retry_queue
		 SET attempt = $1, last_error = $2, next_retry_at = $3
		 WHERE id = $4`,
		attempt, lastError, nextRetryAt, id,
	)
	return err
}

func (r *Repository) MoveToDeadLetter(ctx context.Context, job RetryJob) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	payloadJSON, _ := json.Marshal(job.Payload)
	_, err = tx.Exec(ctx,
		`INSERT INTO function_dead_letters
		 (project_id, function_name, event_type, collection, payload, attempts, last_error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		job.ProjectID, job.FunctionName, job.EventType, job.Collection,
		payloadJSON, job.Attempt, job.LastError,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM function_retry_queue WHERE id = $1`, job.ID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

type DeadLetter struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	FunctionName string                 `json:"function_name"`
	EventType    string                 `json:"event_type"`
	Collection   string                 `json:"collection"`
	Payload      map[string]interface{} `json:"payload"`
	Attempts     int                    `json:"attempts"`
	LastError    string                 `json:"last_error"`
	FailedAt     time.Time              `json:"failed_at"`
}

func (r *Repository) ListDeadLetters(ctx context.Context, projectID string, limit int) ([]DeadLetter, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, event_type, collection, payload, attempts, last_error, failed_at
		 FROM function_dead_letters
		 WHERE project_id = $1
		 ORDER BY failed_at DESC LIMIT $2`,
		projectID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	letters := []DeadLetter{}
	for rows.Next() {
		var d DeadLetter
		var payloadJSON []byte
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.FunctionName, &d.EventType,
			&d.Collection, &payloadJSON, &d.Attempts, &d.LastError, &d.FailedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(payloadJSON, &d.Payload)
		letters = append(letters, d)
	}
	return letters, rows.Err()
}

func (r *Repository) DeleteDeadLetter(ctx context.Context, projectID, id string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM function_dead_letters WHERE id = $1 AND project_id = $2`,
		id, projectID,
	)
	return err
}

func (r *Repository) ReplayDeadLetter(ctx context.Context, projectID, id string) (*DeadLetter, error) {
	var d DeadLetter
	var payloadJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, function_name, event_type, collection, payload, attempts, last_error, failed_at
		 FROM function_dead_letters WHERE id = $1 AND project_id = $2`,
		id, projectID,
	).Scan(&d.ID, &d.ProjectID, &d.FunctionName, &d.EventType,
		&d.Collection, &payloadJSON, &d.Attempts, &d.LastError, &d.FailedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(payloadJSON, &d.Payload)
	return &d, nil
}

func (r *Repository) UpsertSecret(ctx context.Context, projectID, name, encryptedValue string) (*Secret, error) {
	var s Secret
	err := r.db.QueryRow(ctx,
		`INSERT INTO project_secrets (project_id, name, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (project_id, name) DO UPDATE SET value = $3, updated_at = NOW()
		 RETURNING id, project_id, name, created_at, updated_at`,
		projectID, name, encryptedValue,
	).Scan(&s.ID, &s.ProjectID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListSecrets(ctx context.Context, projectID string) ([]Secret, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, created_at, updated_at
		 FROM project_secrets WHERE project_id = $1 ORDER BY name ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secrets := []Secret{}
	for rows.Next() {
		var s Secret
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}
	return secrets, rows.Err()
}

func (r *Repository) GetSecretValues(ctx context.Context, projectID string) (map[string]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT name, value FROM project_secrets WHERE project_id = $1`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	secrets := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		secrets[name] = value
	}
	return secrets, rows.Err()
}

func (r *Repository) DeleteSecret(ctx context.Context, projectID, name string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM project_secrets WHERE project_id = $1 AND name = $2`,
		projectID, name,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("secret not found")
	}
	return nil
}


type TriggerStat struct {
	FunctionName string  `json:"function_name"`
	EventType    string  `json:"event_type"`
	Collection   string  `json:"collection"`
	Total        int     `json:"total"`
	Successes    int     `json:"successes"`
	Failures     int     `json:"failures"`
	Timeouts     int     `json:"timeouts"`
	SuccessRate  float64 `json:"success_rate"`
	FailureRate  float64 `json:"failure_rate"`
}

func (r *Repository) GetTriggerStats(ctx context.Context, projectID string) ([]TriggerStat, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
			function_name,
			event_type,
			collection,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'success') as successes,
			COUNT(*) FILTER (WHERE status = 'error') as failures,
			COUNT(*) FILTER (WHERE status = 'timeout') as timeouts
		FROM function_logs
		WHERE project_id = $1
			AND trigger_type = 'db'
			AND created_at >= NOW() - INTERVAL '24 hours'
			AND event_type IS NOT NULL
			AND collection IS NOT NULL
			AND function_name IS NOT NULL
		GROUP BY function_name, event_type, collection
		ORDER BY total DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []TriggerStat{}
	for rows.Next() {
		var s TriggerStat
		if err := rows.Scan(
			&s.FunctionName, &s.EventType, &s.Collection,
			&s.Total, &s.Successes, &s.Failures, &s.Timeouts,
		); err != nil {
			return nil, err
		}
		if s.Total > 0 {
			s.SuccessRate = math.Round((float64(s.Successes)/float64(s.Total))*100*100) / 100
			s.FailureRate = math.Round((float64(s.Failures)/float64(s.Total))*100*100) / 100
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
