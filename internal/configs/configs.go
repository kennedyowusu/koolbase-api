package configs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Config struct {
	ID            string          `json:"id"`
	EnvironmentID string          `json:"environment_id"`
	Key           string          `json:"key"`
	Value         json.RawMessage `json:"value"`
	Description   string          `json:"description"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, envID, key string, value json.RawMessage, description string) (*Config, error) {
	var c Config
	err := r.db.QueryRow(ctx,
		`INSERT INTO remote_configs (environment_id, key, value, description)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, environment_id, key, value, description, created_at, updated_at`,
		envID, key, []byte(value), description,
	).Scan(&c.ID, &c.EnvironmentID, &c.Key, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create config: %w", err)
	}
	return &c, nil
}

func (r *Repository) Update(ctx context.Context, id string, value json.RawMessage, description string) (*Config, error) {
	var c Config
	err := r.db.QueryRow(ctx,
		`UPDATE remote_configs
		 SET value = $2, description = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, environment_id, key, value, description, created_at, updated_at`,
		id, []byte(value), description,
	).Scan(&c.ID, &c.EnvironmentID, &c.Key, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update config: %w", err)
	}
	return &c, nil
}

func (r *Repository) Delete(ctx context.Context, id string) (string, error) {
	var envID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM remote_configs WHERE id = $1 RETURNING environment_id`, id,
	).Scan(&envID)
	if err != nil {
		return "", fmt.Errorf("delete config: %w", err)
	}
	return envID, nil
}

func (r *Repository) ListByEnvironment(ctx context.Context, envID string) ([]Config, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, environment_id, key, value, description, created_at, updated_at
		 FROM remote_configs WHERE environment_id = $1 ORDER BY created_at DESC`,
		envID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Config
	for rows.Next() {
		var c Config
		if err := rows.Scan(&c.ID, &c.EnvironmentID, &c.Key, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

type Handler struct {
	repo *Repository
	rdb  *redis.Client
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{repo: NewRepository(db), rdb: rdb}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")

	var body struct {
		Key         string          `json:"key"`
		Value       json.RawMessage `json:"value"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		writeError(w, http.StatusBadRequest, "key and value are required")
		return
	}
	if len(body.Value) == 0 {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	config, err := h.repo.Create(r.Context(), envID, body.Key, body.Value, body.Description)
	if err != nil {
		log.Error().Err(err).Msg("create config failed")
		writeError(w, http.StatusInternalServerError, "failed to create config")
		return
	}

	writeJSON(w, http.StatusCreated, config)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "config_id")

	var body struct {
		Value       json.RawMessage `json:"value"`
		Description string          `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	config, err := h.repo.Update(r.Context(), configID, body.Value, body.Description)
	if err != nil {
		log.Error().Err(err).Msg("update config failed")
		writeError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	configID := chi.URLParam(r, "config_id")

	_, err := h.repo.Delete(r.Context(), configID)
	if err != nil {
		log.Error().Err(err).Msg("delete config failed")
		writeError(w, http.StatusInternalServerError, "failed to delete config")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")

	list, err := h.repo.ListByEnvironment(r.Context(), envID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list configs")
		return
	}
	if list == nil {
		list = []Config{}
	}
	writeJSON(w, http.StatusOK, list)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
