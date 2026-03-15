package environments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/kennedyowusu/hatchway-api/internal/bootstrap"
)

type Environment struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	PublicKey string    `json:"public_key"`
	SecretKey string    `json:"secret_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, projectID, name, slug, publicKey, secretKey string) (*Environment, error) {
	var env Environment
	err := r.db.QueryRow(ctx,
		`INSERT INTO environments (project_id, name, slug, public_key, secret_key)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, project_id, name, slug, public_key, secret_key, created_at, updated_at`,
		projectID, name, slug, publicKey, secretKey,
	).Scan(&env.ID, &env.ProjectID, &env.Name, &env.Slug, &env.PublicKey, &env.SecretKey, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create environment: %w", err)
	}
	return &env, nil
}

func (r *Repository) ListByProject(ctx context.Context, projectID string) ([]Environment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, project_id, name, slug, public_key, secret_key, created_at, updated_at
		 FROM environments WHERE project_id = $1 ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envs []Environment
	for rows.Next() {
		var env Environment
		if err := rows.Scan(&env.ID, &env.ProjectID, &env.Name, &env.Slug, &env.PublicKey, &env.SecretKey, &env.CreatedAt, &env.UpdatedAt); err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, rows.Err()
}

type Handler struct {
	repo    *Repository
	builder *bootstrap.SnapshotBuilder
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	snapshotRepo := bootstrap.NewSnapshotRepository(rdb)
	builder := bootstrap.NewSnapshotBuilder(db, snapshotRepo)
	return &Handler{repo: NewRepository(db), builder: builder}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := toSlug(body.Name)
	publicKey, err := generateKey("public", body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate public key")
		return
	}
	secretKey, err := generateKey("secret", body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate secret key")
		return
	}

	env, err := h.repo.Create(r.Context(), projectID, body.Name, slug, publicKey, secretKey)
	if err != nil {
		log.Error().Err(err).Msg("create environment failed")
		writeError(w, http.StatusInternalServerError, "failed to create environment")
		return
	}

	go func() {
		if err := h.builder.Rebuild(context.Background(), env.ID); err != nil {
			log.Error().Err(err).Str("env_id", env.ID).Msg("initial snapshot build failed")
		}
	}()

	writeJSON(w, http.StatusCreated, env)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	envs, err := h.repo.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	if envs == nil {
		envs = []Environment{}
	}
	writeJSON(w, http.StatusOK, envs)
}

func toSlug(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func generateKey(keyType, envName string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	var prefix string
	switch keyType {
	case "secret":
		if envName == "production" {
			prefix = "sk_live"
		} else {
			prefix = "sk_test"
		}
	default:
		if envName == "production" {
			prefix = "pk_live"
		} else {
			prefix = "pk_test"
		}
	}

	return fmt.Sprintf("%s_%s", prefix, token), nil
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

func (r *Repository) Delete(ctx context.Context, envID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM environments WHERE id = $1`, envID)
	return err
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")
	if envID == "" {
		writeError(w, http.StatusBadRequest, "env_id is required")
		return
	}

	if err := h.repo.Delete(r.Context(), envID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete environment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
