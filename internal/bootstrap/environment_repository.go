package bootstrap

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Environment holds the resolved environment data from a public key lookup
type Environment struct {
	ID        string
	ProjectID string
	Name      string
	PublicKey string
}

// EnvironmentRepository handles DB lookups for environments
type EnvironmentRepository struct {
	db *pgxpool.Pool
}

func NewEnvironmentRepository(db *pgxpool.Pool) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

// FindByPublicKey resolves an environment from an SDK public key.
// This is the entry point for every bootstrap request.
func (r *EnvironmentRepository) FindByPublicKey(ctx context.Context, publicKey string) (*Environment, error) {
	var env Environment
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, public_key
		 FROM environments
		 WHERE public_key = $1
		 LIMIT 1`,
		publicKey,
	).Scan(&env.ID, &env.ProjectID, &env.Name, &env.PublicKey)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrEnvironmentNotFound, publicKey)
	}

	return &env, nil
}
