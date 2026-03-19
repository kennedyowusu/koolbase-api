package realtime

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBAuthorizer struct {
	db *pgxpool.Pool
}

func NewDBAuthorizer(db *pgxpool.Pool) *DBAuthorizer {
	return &DBAuthorizer{db: db}
}

func (a *DBAuthorizer) AuthorizeProject(projectID, orgID string) (bool, error) {
	var count int
	err := a.db.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM projects WHERE id = $1 AND org_id = $2`,
		projectID, orgID,
	).Scan(&count)
	return count > 0, err
}
