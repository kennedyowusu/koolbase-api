package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCollectionNotFound = errors.New("collection not found")
	ErrRecordNotFound     = errors.New("record not found")
	ErrCollectionExists   = errors.New("collection already exists")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetProjectByAPIKey(ctx context.Context, apiKey string) (projectID, environmentID string, err error) {
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

func (r *Repository) AuthorizeProject(ctx context.Context, projectID, orgID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects WHERE id = $1 AND org_id = $2`,
		projectID, orgID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) CreateCollection(ctx context.Context, projectID, name, readRule, writeRule, deleteRule string) (*Collection, error) {
	var c Collection
	err := r.db.QueryRow(ctx,
		`INSERT INTO db_collections (project_id, name, read_rule, write_rule, delete_rule)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, project_id, name, read_rule, write_rule, delete_rule, created_at`,
		projectID, name, readRule, writeRule, deleteRule,
	).Scan(&c.ID, &c.ProjectID, &c.Name, &c.ReadRule, &c.WriteRule, &c.DeleteRule, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrCollectionExists
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) GetCollection(ctx context.Context, projectID, name string) (*Collection, error) {
	var c Collection
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, read_rule, write_rule, delete_rule, created_at
		 FROM db_collections WHERE project_id = $1 AND name = $2`,
		projectID, name,
	).Scan(&c.ID, &c.ProjectID, &c.Name, &c.ReadRule, &c.WriteRule, &c.DeleteRule, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCollections(ctx context.Context, projectID string) ([]Collection, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, read_rule, write_rule, delete_rule, created_at
		 FROM db_collections WHERE project_id = $1 ORDER BY created_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collections := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.ReadRule, &c.WriteRule, &c.DeleteRule, &c.CreatedAt); err != nil {
			return nil, err
		}
		collections = append(collections, c)
	}
	return collections, rows.Err()
}

func (r *Repository) DeleteCollection(ctx context.Context, projectID, name string) error {
	res, err := r.db.Exec(ctx,
		`DELETE FROM db_collections WHERE project_id = $1 AND name = $2`,
		projectID, name,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

func (r *Repository) InsertRecord(ctx context.Context, projectID, collectionID string, createdBy *string, data map[string]interface{}) (*Record, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var rec Record
	var rawData []byte
	err = r.db.QueryRow(ctx,
		`INSERT INTO db_records (project_id, collection_id, created_by, data)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, project_id, collection_id, created_by, data, created_at, updated_at`,
		projectID, collectionID, createdBy, dataJSON,
	).Scan(&rec.ID, &rec.ProjectID, &rec.CollectionID, &rec.CreatedBy, &rawData, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawData, &rec.Data); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) GetRecord(ctx context.Context, projectID, recordID string) (*Record, error) {
	var rec Record
	var rawData []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, collection_id, created_by, data, created_at, updated_at
		 FROM db_records WHERE id = $1 AND project_id = $2 AND deleted_at IS NULL`,
		recordID, projectID,
	).Scan(&rec.ID, &rec.ProjectID, &rec.CollectionID, &rec.CreatedBy, &rawData, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(rawData, &rec.Data); err != nil {
		return nil, err
	}
	return &rec, nil
}

// isSafeFieldName validates that a field name only contains safe characters
// to prevent SQL injection via filter keys.
func isSafeFieldName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_') {
			return false
		}
	}
	return true
}

func (r *Repository) QueryRecords(ctx context.Context, projectID, collectionID string, filters map[string]interface{}, limit, offset int, orderBy string, orderDesc bool) ([]Record, int, error) {
	args := []interface{}{projectID, collectionID}
	argIdx := 3
	filterSQL := ""

	for key, val := range filters {
		// Sanitize filter keys to prevent SQL injection
		if !isSafeFieldName(key) {
			continue
		}
		filterSQL += fmt.Sprintf(` AND data->>'%s' = $%d`, key, argIdx)
		args = append(args, fmt.Sprintf("%v", val))
		argIdx++
	}

	var total int
	r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM db_records WHERE project_id = $1 AND collection_id = $2 AND deleted_at IS NULL%s`, filterSQL),
		args...,
	).Scan(&total)

	order := "created_at ASC"
	if orderBy != "" {
		dir := "ASC"
		if orderDesc {
			dir = "DESC"
		}
		order = fmt.Sprintf("data->>'%s' %s", orderBy, dir)
	} else if orderDesc {
		order = "created_at DESC"
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx,
		fmt.Sprintf(`SELECT id, project_id, collection_id, created_by, data, created_at, updated_at
		 FROM db_records WHERE project_id = $1 AND collection_id = $2 AND deleted_at IS NULL%s
		 ORDER BY %s LIMIT $%d OFFSET $%d`, filterSQL, order, argIdx, argIdx+1),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records := []Record{}
	for rows.Next() {
		var rec Record
		var rawData []byte
		if err := rows.Scan(&rec.ID, &rec.ProjectID, &rec.CollectionID, &rec.CreatedBy, &rawData, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(rawData, &rec.Data); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	return records, total, rows.Err()
}

func (r *Repository) UpdateRecord(ctx context.Context, projectID, recordID string, data map[string]interface{}) (*Record, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var rec Record
	var rawData []byte
	err = r.db.QueryRow(ctx,
		`UPDATE db_records SET data = data || $1::jsonb
		 WHERE id = $2 AND project_id = $3 AND deleted_at IS NULL
		 RETURNING id, project_id, collection_id, created_by, data, created_at, updated_at`,
		dataJSON, recordID, projectID,
	).Scan(&rec.ID, &rec.ProjectID, &rec.CollectionID, &rec.CreatedBy, &rawData, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal(rawData, &rec.Data); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) DeleteRecord(ctx context.Context, projectID, recordID string) error {
	res, err := r.db.Exec(ctx,
		`UPDATE db_records SET deleted_at = NOW() WHERE id = $1 AND project_id = $2 AND deleted_at IS NULL`,
		recordID, projectID,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (r *Repository) GetCollectionByID(ctx context.Context, collectionID string) (*Collection, error) {
	var c Collection
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, name, read_rule, write_rule, delete_rule, created_at
		 FROM db_collections WHERE id = $1`,
		collectionID,
	).Scan(&c.ID, &c.ProjectID, &c.Name, &c.ReadRule, &c.WriteRule, &c.DeleteRule, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCollectionNotFound
		}
		return nil, err
	}
	return &c, nil
}

// PopulateRecords fetches related records for a list of records.
// populateFields format: "field_name:collection_name"
// e.g. "author_id:users" fetches the record from "users" where id = record.data["author_id"]
// and injects it as "author" into the record data.
func (r *Repository) PopulateRecords(ctx context.Context, projectID string, records []Record, populateFields []string) error {
	if len(populateFields) == 0 || len(records) == 0 {
		return nil
	}

	// Cache collection lookups to avoid N+1
	collectionCache := map[string]*Collection{}

	for _, pop := range populateFields {
		// Validate format contains ":"
		if !strings.Contains(pop, ":") {
			continue
		}

		parts := splitPopulate(pop)
		if parts == nil {
			continue
		}
		fieldName := parts[0]
		collectionName := parts[1]
		if fieldName == "" || collectionName == "" {
			continue
		}

		// Validate field name is safe
		if !isSafeFieldName(fieldName) {
			continue
		}

		// Get collection from cache or DB
		col, ok := collectionCache[collectionName]
		if !ok {
			var err error
			col, err = r.GetCollection(ctx, projectID, collectionName)
			if err != nil {
				// Collection doesn't exist — skip this populate field
				continue
			}
			collectionCache[collectionName] = col
		}

		// Collect all referenced IDs from records
		idSet := map[string]bool{}
		for _, rec := range records {
			val, ok := rec.Data[fieldName]
			if !ok {
				continue
			}
			id, ok := val.(string)
			if !ok || id == "" {
				continue
			}
			idSet[id] = true
		}
		if len(idSet) == 0 {
			continue
		}

		// Cap at 100 IDs to prevent unbounded queries
		ids := make([]string, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
			if len(ids) >= 100 {
				break
			}
		}

		// Use ANY with string array for safe queries
		rows, err := r.db.Query(ctx,
			`SELECT id, project_id, collection_id, created_by, data, created_at, updated_at
			 FROM db_records
			 WHERE project_id = $1
			 AND collection_id = $2
			 AND id = ANY($3)
			 AND deleted_at IS NULL`,
			projectID, col.ID, ids,
		)
		if err != nil {
			return fmt.Errorf("populate query failed for %s: %w", pop, err)
		}

		refMap := map[string]map[string]interface{}{}
		for rows.Next() {
			var ref Record
			var rawData []byte
			if err := rows.Scan(
				&ref.ID,
				&ref.ProjectID,
				&ref.CollectionID,
				&ref.CreatedBy,
				&rawData,
				&ref.CreatedAt,
				&ref.UpdatedAt,
			); err != nil {
				rows.Close()
				return fmt.Errorf("populate scan failed for %s: %w", pop, err)
			}
			if err := json.Unmarshal(rawData, &ref.Data); err != nil {
				rows.Close()
				return fmt.Errorf("populate unmarshal failed for %s: %w", pop, err)
			}
			refMap[ref.ID] = map[string]interface{}{
				"id":         ref.ID,
				"data":       ref.Data,
				"created_at": ref.CreatedAt,
				"updated_at": ref.UpdatedAt,
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("populate rows error for %s: %w", pop, err)
		}

		// Inject populated data — strip _id suffix for key name (author_id → author)
		populatedKey := strings.TrimSuffix(fieldName, "_id")
		for i, rec := range records {
			val, ok := rec.Data[fieldName]
			if !ok {
				continue
			}
			id, ok := val.(string)
			if !ok || id == "" {
				continue
			}
			ref, found := refMap[id]
			if !found {
				continue
			}
			// Guard against key collision — don't overwrite existing non-ID field
			if _, exists := rec.Data[populatedKey]; exists && populatedKey != fieldName {
				continue
			}
			// Copy data map to avoid shared mutation
			newData := make(map[string]interface{}, len(rec.Data)+1)
			for k, v := range rec.Data {
				newData[k] = v
			}
			newData[populatedKey] = ref
			records[i].Data = newData
		}
	}
	return nil
}

func splitPopulate(s string) []string {
	for i, c := range s {
		if c == ':' {
			left := strings.TrimSpace(s[:i])
			right := strings.TrimSpace(s[i+1:])
			if left == "" || right == "" {
				return nil
			}
			return []string{left, right}
		}
	}
	return nil
}
