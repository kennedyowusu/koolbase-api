package flags

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, envID, key string, enabled bool, rollout int, description string) (*Flag, error) {
	var f Flag
	err := r.db.QueryRow(ctx,
		`INSERT INTO feature_flags (environment_id, key, enabled, rollout_percentage, description)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, environment_id, key, enabled, rollout_percentage, kill_switch, description, created_at, updated_at`,
		envID, key, enabled, rollout, description,
	).Scan(&f.ID, &f.EnvironmentID, &f.Key, &f.Enabled, &f.RolloutPercentage, &f.KillSwitch, &f.Description, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create flag: %w", err)
	}
	return &f, nil
}

func (r *Repository) Update(ctx context.Context, id string, enabled bool, rollout int, killSwitch bool, description string) (*Flag, error) {
	var f Flag
	err := r.db.QueryRow(ctx,
		`UPDATE feature_flags
		 SET enabled = $2, rollout_percentage = $3, kill_switch = $4, description = $5, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, environment_id, key, enabled, rollout_percentage, kill_switch, description, created_at, updated_at`,
		id, enabled, rollout, killSwitch, description,
	).Scan(&f.ID, &f.EnvironmentID, &f.Key, &f.Enabled, &f.RolloutPercentage, &f.KillSwitch, &f.Description, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update flag: %w", err)
	}
	return &f, nil
}

func (r *Repository) Delete(ctx context.Context, id string) (string, error) {
	var envID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM feature_flags WHERE id = $1 RETURNING environment_id`, id,
	).Scan(&envID)
	if err != nil {
		return "", fmt.Errorf("delete flag: %w", err)
	}
	return envID, nil
}

func (r *Repository) ListByEnvironment(ctx context.Context, envID string) ([]Flag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, environment_id, key, enabled, rollout_percentage, kill_switch, description, created_at, updated_at
		 FROM feature_flags WHERE environment_id = $1 ORDER BY created_at DESC`,
		envID,
	)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	defer rows.Close()

	var list []Flag
	for rows.Next() {
		var f Flag
		if err := rows.Scan(&f.ID, &f.EnvironmentID, &f.Key, &f.Enabled, &f.RolloutPercentage, &f.KillSwitch, &f.Description, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}
