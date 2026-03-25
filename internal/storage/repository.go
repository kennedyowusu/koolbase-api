package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBucketNotFound = errors.New("bucket not found")
var ErrObjectNotFound = errors.New("object not found")
var ErrBucketExists = errors.New("bucket already exists")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetProjectIDByAPIKey(ctx context.Context, apiKey string) (projectID, environmentID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT project_id, id FROM environments WHERE public_key = $1`,
		apiKey,
	).Scan(&projectID, &environmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errors.New("invalid api key")
		}
		return "", "", err
	}
	return projectID, environmentID, nil
}

func (r *Repository) GetProjectIDByDashboardUser(ctx context.Context, projectID, userOrgID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects WHERE id = $1 AND org_id = $2`,
		projectID, userOrgID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) CreateBucket(ctx context.Context, projectID, name string, public bool) (*Bucket, error) {
	var b Bucket
	err := r.db.QueryRow(ctx,
		`INSERT INTO storage_buckets (project_id, name, public)
		 VALUES ($1, $2, $3)
		 RETURNING id, project_id, name, public, created_at`,
		projectID, name, public,
	).Scan(&b.ID, &b.ProjectID, &b.Name, &b.Public, &b.CreatedAt)
	if err != nil {
		if err.Error() == `ERROR: duplicate key value violates unique constraint "storage_buckets_project_id_name_key" (SQLSTATE 23505)` {
			return nil, ErrBucketExists
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) GetBucket(ctx context.Context, projectID, name string) (*Bucket, error) {
	var b Bucket
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, public, created_at
		 FROM storage_buckets WHERE project_id = $1 AND name = $2`,
		projectID, name,
	).Scan(&b.ID, &b.ProjectID, &b.Name, &b.Public, &b.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBucketNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) ListBuckets(ctx context.Context, projectID string) ([]Bucket, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, public, created_at
		 FROM storage_buckets WHERE project_id = $1 ORDER BY created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := []Bucket{}
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.Name, &b.Public, &b.CreatedAt); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

func (r *Repository) DeleteBucket(ctx context.Context, projectID, name string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM storage_buckets WHERE project_id = $1 AND name = $2`,
		projectID, name,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrBucketNotFound
	}
	return nil
}

func (r *Repository) InsertObject(ctx context.Context, projectID, bucketID string, userID *string, path string, size int64, contentType, etag string) (*Object, error) {
	var obj Object
	err := r.db.QueryRow(ctx,
		`INSERT INTO storage_objects (project_id, bucket_id, user_id, path, size, content_type, etag)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (bucket_id, path)
		 DO UPDATE SET size = $5, content_type = $6, etag = $7, updated_at = NOW()
		 RETURNING id, project_id, bucket_id, user_id, path, size, content_type, etag, created_at, updated_at`,
		projectID, bucketID, userID, path, size, contentType, etag,
	).Scan(&obj.ID, &obj.ProjectID, &obj.BucketID, &obj.UserID, &obj.Path, &obj.Size, &obj.ContentType, &obj.ETag, &obj.CreatedAt, &obj.UpdatedAt)
	return &obj, err
}

func (r *Repository) ListObjects(ctx context.Context, bucketID, prefix string, limit, offset int) ([]Object, int, error) {
	var total int
	query := `SELECT COUNT(*) FROM storage_objects WHERE bucket_id = $1`
	args := []interface{}{bucketID}
	if prefix != "" {
		query += ` AND path LIKE $2`
		args = append(args, prefix+"%")
	}
	r.db.QueryRow(ctx, query, args...).Scan(&total)

	listQuery := `SELECT id, project_id, bucket_id, user_id, path, size, content_type, etag, created_at, updated_at
				  FROM storage_objects WHERE bucket_id = $1`
	listArgs := []interface{}{bucketID}
	argIdx := 2
	if prefix != "" {
		listQuery += fmt.Sprintf(` AND path LIKE $%d`, argIdx)
		listArgs = append(listArgs, prefix+"%")
		argIdx++
	}
	listQuery += fmt.Sprintf(` ORDER BY path ASC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	objects := []Object{}
	for rows.Next() {
		var obj Object
		if err := rows.Scan(&obj.ID, &obj.ProjectID, &obj.BucketID, &obj.UserID, &obj.Path, &obj.Size, &obj.ContentType, &obj.ETag, &obj.CreatedAt, &obj.UpdatedAt); err != nil {
			return nil, 0, err
		}
		objects = append(objects, obj)
	}
	return objects, total, rows.Err()
}

func (r *Repository) DeleteObject(ctx context.Context, bucketID, path string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM storage_objects WHERE bucket_id = $1 AND path = $2`,
		bucketID, path,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrObjectNotFound
	}
	return nil
}
