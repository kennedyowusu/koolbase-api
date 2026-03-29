package cron

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCronNotFound = errors.New("cron schedule not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, projectID, functionName, cronExpression string, nextRunAt time.Time) (*CronSchedule, error) {
	var c CronSchedule
	err := r.db.QueryRow(ctx,
		`INSERT INTO cron_schedules (project_id, function_name, cron_expression, next_run_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, project_id, function_name, cron_expression, enabled, last_run_at, next_run_at, created_at`,
		projectID, functionName, cronExpression, nextRunAt,
	).Scan(&c.ID, &c.ProjectID, &c.FunctionName, &c.CronExpression, &c.Enabled, &c.LastRunAt, &c.NextRunAt, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) List(ctx context.Context, projectID string) ([]CronSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, cron_expression, enabled, last_run_at, next_run_at, created_at
		 FROM cron_schedules WHERE project_id = $1 ORDER BY created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := []CronSchedule{}
	for rows.Next() {
		var c CronSchedule
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.FunctionName, &c.CronExpression, &c.Enabled, &c.LastRunAt, &c.NextRunAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, c)
	}
	return schedules, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, projectID, cronID string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM cron_schedules WHERE id = $1 AND project_id = $2`,
		cronID, projectID,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrCronNotFound
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, projectID, cronID string, enabled bool) (*CronSchedule, error) {
	var c CronSchedule
	err := r.db.QueryRow(ctx,
		`UPDATE cron_schedules SET enabled = $1
		 WHERE id = $2 AND project_id = $3
		 RETURNING id, project_id, function_name, cron_expression, enabled, last_run_at, next_run_at, created_at`,
		enabled, cronID, projectID,
	).Scan(&c.ID, &c.ProjectID, &c.FunctionName, &c.CronExpression, &c.Enabled, &c.LastRunAt, &c.NextRunAt, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCronNotFound
		}
		return nil, err
	}
	return &c, nil
}

// GetDue returns all enabled schedules where next_run_at <= now
func (r *Repository) GetDue(ctx context.Context) ([]CronSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, function_name, cron_expression, enabled, last_run_at, next_run_at, created_at
		 FROM cron_schedules
		 WHERE enabled = TRUE AND next_run_at <= NOW()
		 ORDER BY next_run_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedules := []CronSchedule{}
	for rows.Next() {
		var c CronSchedule
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.FunctionName, &c.CronExpression, &c.Enabled, &c.LastRunAt, &c.NextRunAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, c)
	}
	return schedules, rows.Err()
}

// MarkRan updates last_run_at and sets next_run_at
func (r *Repository) MarkRan(ctx context.Context, cronID string, nextRunAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cron_schedules SET last_run_at = NOW(), next_run_at = $1 WHERE id = $2`,
		nextRunAt, cronID,
	)
	return err
}
