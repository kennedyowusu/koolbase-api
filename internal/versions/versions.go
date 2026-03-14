package versions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type VersionPolicy struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environment_id"`
	Platform      string    `json:"platform"`
	MinVersion    string    `json:"min_version"`
	LatestVersion *string   `json:"latest_version"`
	ForceUpdate   bool      `json:"force_update"`
	UpdateMessage string    `json:"update_message"`
	StoreURL      *string   `json:"store_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, envID, platform, minVersion string, latestVersion *string, forceUpdate bool, updateMessage string, storeURL *string) (*VersionPolicy, error) {
	var vp VersionPolicy
	err := r.db.QueryRow(ctx,
		`INSERT INTO version_policies (environment_id, platform, min_version, latest_version, force_update, update_message, store_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (environment_id, platform)
		 DO UPDATE SET
		   min_version = $3,
		   latest_version = $4,
		   force_update = $5,
		   update_message = $6,
		   store_url = $7,
		   updated_at = NOW()
		 RETURNING id, environment_id, platform, min_version, latest_version, force_update, update_message, store_url, created_at, updated_at`,
		envID, platform, minVersion, latestVersion, forceUpdate, updateMessage, storeURL,
	).Scan(&vp.ID, &vp.EnvironmentID, &vp.Platform, &vp.MinVersion, &vp.LatestVersion, &vp.ForceUpdate, &vp.UpdateMessage, &vp.StoreURL, &vp.CreatedAt, &vp.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert version policy: %w", err)
	}
	return &vp, nil
}

func (r *Repository) ListByEnvironment(ctx context.Context, envID string) ([]VersionPolicy, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, environment_id, platform, min_version, latest_version, force_update, update_message, store_url, created_at, updated_at
		 FROM version_policies WHERE environment_id = $1 ORDER BY platform`,
		envID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []VersionPolicy
	for rows.Next() {
		var vp VersionPolicy
		if err := rows.Scan(&vp.ID, &vp.EnvironmentID, &vp.Platform, &vp.MinVersion, &vp.LatestVersion, &vp.ForceUpdate, &vp.UpdateMessage, &vp.StoreURL, &vp.CreatedAt, &vp.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, vp)
	}
	return list, rows.Err()
}

type Handler struct {
	repo *Repository
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{repo: NewRepository(db)}
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")

	var body struct {
		Platform      string  `json:"platform"`
		MinVersion    string  `json:"min_version"`
		LatestVersion *string `json:"latest_version"`
		ForceUpdate   bool    `json:"force_update"`
		UpdateMessage string  `json:"update_message"`
		StoreURL      *string `json:"store_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Platform == "" || body.MinVersion == "" {
		writeError(w, http.StatusBadRequest, "platform and min_version are required")
		return
	}

	vp, err := h.repo.Upsert(r.Context(), envID, body.Platform, body.MinVersion, body.LatestVersion, body.ForceUpdate, body.UpdateMessage, body.StoreURL)
	if err != nil {
		log.Error().Err(err).Msg("upsert version policy failed")
		writeError(w, http.StatusInternalServerError, "failed to upsert version policy")
		return
	}

	writeJSON(w, http.StatusOK, vp)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")

	list, err := h.repo.ListByEnvironment(r.Context(), envID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list version policies")
		return
	}
	if list == nil {
		list = []VersionPolicy{}
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
