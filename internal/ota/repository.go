package ota

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Bundle struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Channel      string    `json:"channel"`
	Version      int       `json:"version"`
	Checksum     string    `json:"checksum"`
	StoragePath  string    `json:"storage_path"`
	FileSize     int64     `json:"file_size"`
	Mandatory    bool      `json:"mandatory"`
	Active       bool      `json:"active"`
	ReleaseNotes *string   `json:"release_notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) NextVersion(ctx context.Context, projectID, channel string) (int, error) {
	var max int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM ota_bundles WHERE project_id = $1 AND channel = $2`,
		projectID, channel,
	).Scan(&max)
	return max + 1, err
}

func (r *Repository) Create(ctx context.Context, b Bundle) (Bundle, error) {
	var out Bundle
	err := r.db.QueryRow(ctx,
		`INSERT INTO ota_bundles
		 (project_id, channel, version, checksum, storage_path, file_size, mandatory, active, release_notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, project_id, channel, version, checksum, storage_path, file_size, mandatory, active, release_notes, created_at, updated_at`,
		b.ProjectID, b.Channel, b.Version, b.Checksum, b.StoragePath,
		b.FileSize, b.Mandatory, b.Active, b.ReleaseNotes,
	).Scan(
		&out.ID, &out.ProjectID, &out.Channel, &out.Version,
		&out.Checksum, &out.StoragePath, &out.FileSize,
		&out.Mandatory, &out.Active, &out.ReleaseNotes,
		&out.CreatedAt, &out.UpdatedAt,
	)
	return out, err
}

func (r *Repository) List(ctx context.Context, projectID string) ([]Bundle, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, channel, version, checksum, storage_path, file_size, mandatory, active, release_notes, created_at, updated_at
		 FROM ota_bundles WHERE project_id = $1
		 ORDER BY channel, version DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bundles []Bundle
	for rows.Next() {
		var b Bundle
		if err := rows.Scan(
			&b.ID, &b.ProjectID, &b.Channel, &b.Version,
			&b.Checksum, &b.StoragePath, &b.FileSize,
			&b.Mandatory, &b.Active, &b.ReleaseNotes,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bundles = append(bundles, b)
	}
	return bundles, nil
}

func (r *Repository) GetActive(ctx context.Context, projectID, channel string) (*Bundle, error) {
	var b Bundle
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, channel, version, checksum, storage_path, file_size, mandatory, active, release_notes, created_at, updated_at
		 FROM ota_bundles WHERE project_id = $1 AND channel = $2 AND active = TRUE`,
		projectID, channel,
	).Scan(
		&b.ID, &b.ProjectID, &b.Channel, &b.Version,
		&b.Checksum, &b.StoragePath, &b.FileSize,
		&b.Mandatory, &b.Active, &b.ReleaseNotes,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) SetActive(ctx context.Context, projectID, channel, bundleID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE ota_bundles SET active = FALSE WHERE project_id = $1 AND channel = $2`,
		projectID, channel,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE ota_bundles SET active = TRUE, updated_at = now() WHERE id = $1`,
		bundleID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) UpdateMandatory(ctx context.Context, bundleID string, mandatory bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE ota_bundles SET mandatory = $1, updated_at = now() WHERE id = $2`,
		mandatory, bundleID,
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, projectID, bundleID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM ota_bundles WHERE id = $1 AND project_id = $2`,
		bundleID, projectID,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, projectID, bundleID string) (*Bundle, error) {
	var b Bundle
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, channel, version, checksum, storage_path, file_size, mandatory, active, release_notes, created_at, updated_at
		 FROM ota_bundles WHERE id = $1 AND project_id = $2`,
		bundleID, projectID,
	).Scan(
		&b.ID, &b.ProjectID, &b.Channel, &b.Version,
		&b.Checksum, &b.StoragePath, &b.FileSize,
		&b.Mandatory, &b.Active, &b.ReleaseNotes,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) GetProjectIDByAPIKey(ctx context.Context, apiKey string) (projectID, environmentID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT project_id, id FROM environments WHERE public_key = $1`,
		apiKey,
	).Scan(&projectID, &environmentID)
	if err != nil {
		return "", "", err
	}
	return projectID, environmentID, nil
}
