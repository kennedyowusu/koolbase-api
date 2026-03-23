package billing

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetOrgPlan(ctx context.Context, orgID string) (string, error) {
	var plan string
	err := r.db.QueryRow(ctx,
		`SELECT plan FROM organizations WHERE id = $1`, orgID,
	).Scan(&plan)
	return plan, err
}

func (r *PGRepository) CountEnvironments(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM environments e
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func (r *PGRepository) CountFlags(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM feature_flags ff
		 JOIN environments e ON e.id = ff.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func (r *PGRepository) CountConfigs(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM remote_configs rc
		 JOIN environments e ON e.id = rc.environment_id
		 JOIN projects p ON p.id = e.project_id
		 WHERE p.org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func (r *PGRepository) CountMembers(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func (r *PGRepository) CountFunctions(ctx context.Context, projectID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT name) FROM project_functions WHERE project_id = $1`, projectID,
	).Scan(&count)
	return count, err
}

func (r *PGRepository) CountSecrets(ctx context.Context, projectID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM project_secrets WHERE project_id = $1`, projectID,
	).Scan(&count)
	return count, err
}
