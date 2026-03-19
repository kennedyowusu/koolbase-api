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

func (r *Repository) RotatePublicKey(ctx context.Context, envID, newKey string) (*Environment, error) {
	var env Environment
	err := r.db.QueryRow(ctx,
		`UPDATE environments SET public_key = $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING id, project_id, name, slug, public_key, secret_key, created_at, updated_at`,
		newKey, envID,
	).Scan(&env.ID, &env.ProjectID, &env.Name, &env.Slug, &env.PublicKey, &env.SecretKey, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func (r *Repository) RotateSecretKey(ctx context.Context, envID, newKey string) (*Environment, error) {
	var env Environment
	err := r.db.QueryRow(ctx,
		`UPDATE environments SET secret_key = $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING id, project_id, name, slug, public_key, secret_key, created_at, updated_at`,
		newKey, envID,
	).Scan(&env.ID, &env.ProjectID, &env.Name, &env.Slug, &env.PublicKey, &env.SecretKey, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func (h *Handler) RotateKey(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")
	keyType := chi.URLParam(r, "key_type") // "public" or "secret"

	if keyType != "public" && keyType != "secret" {
		writeError(w, http.StatusBadRequest, "key_type must be public or secret")
		return
	}

	// Get environment to determine name for prefix
	envs, err := h.repo.db.Query(r.Context(),
		`SELECT name FROM environments WHERE id = $1`, envID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch environment")
		return
	}
	defer envs.Close()

	var envName string
	if envs.Next() {
		envs.Scan(&envName)
	} else {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	envs.Close()

	newKey, err := generateKey(keyType, envName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}

	var env *Environment
	if keyType == "public" {
		env, err = h.repo.RotatePublicKey(r.Context(), envID, newKey)
	} else {
		env, err = h.repo.RotateSecretKey(r.Context(), envID, newKey)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to rotate key")
		return
	}

	writeJSON(w, http.StatusOK, env)
}

func (h *Handler) Duplicate(w http.ResponseWriter, r *http.Request) {
	sourceEnvID := chi.URLParam(r, "env_id")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Fetch source environment to get project_id
	var projectID string
	err := h.repo.db.QueryRow(r.Context(),
		`SELECT project_id FROM environments WHERE id = $1`, sourceEnvID,
	).Scan(&projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "source environment not found")
		return
	}

	// Generate keys for new environment
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

	// Create the new environment
	newEnv, err := h.repo.Create(r.Context(), projectID, body.Name, slug, publicKey, secretKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create environment")
		return
	}

	// Copy feature flags
	rows, err := h.repo.db.Query(r.Context(),
		`SELECT key, enabled, rollout_percentage, kill_switch, description
		 FROM feature_flags WHERE environment_id = $1`, sourceEnvID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key, description string
			var enabled, killSwitch bool
			var rollout int
			if err := rows.Scan(&key, &enabled, &rollout, &killSwitch, &description); err != nil {
				continue
			}
			var newFlagID string
			h.repo.db.QueryRow(r.Context(),
				`INSERT INTO feature_flags (environment_id, key, enabled, rollout_percentage, kill_switch, description)
				 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
				newEnv.ID, key, enabled, rollout, killSwitch, description,
			).Scan(&newFlagID)

			// Copy flag rules
			if newFlagID != "" {
				ruleRows, err := h.repo.db.Query(r.Context(),
					`SELECT type, config, priority FROM flag_rules WHERE flag_id IN (
						SELECT id FROM feature_flags WHERE environment_id = $1 AND key = $2
					)`, sourceEnvID, key,
				)
				if err == nil {
					defer ruleRows.Close()
					for ruleRows.Next() {
						var ruleType string
						var config []byte
						var priority int
						if err := ruleRows.Scan(&ruleType, &config, &priority); err != nil {
							continue
						}
						h.repo.db.Exec(r.Context(),
							`INSERT INTO flag_rules (flag_id, type, config, priority) VALUES ($1, $2, $3, $4)`,
							newFlagID, ruleType, config, priority,
						)
					}
				}
			}
		}
	}

	// Copy remote configs
	configRows, err := h.repo.db.Query(r.Context(),
		`SELECT key, value, description FROM remote_configs WHERE environment_id = $1`, sourceEnvID,
	)
	if err == nil {
		defer configRows.Close()
		for configRows.Next() {
			var key, description string
			var value []byte
			if err := configRows.Scan(&key, &value, &description); err != nil {
				continue
			}
			h.repo.db.Exec(r.Context(),
				`INSERT INTO remote_configs (environment_id, key, value, description) VALUES ($1, $2, $3, $4)`,
				newEnv.ID, key, value, description,
			)
		}
	}

	// Copy version policies
	policyRows, err := h.repo.db.Query(r.Context(),
		`SELECT platform, min_version, latest_version, force_update, update_message, store_url
		 FROM version_policies WHERE environment_id = $1`, sourceEnvID,
	)
	if err == nil {
		defer policyRows.Close()
		for policyRows.Next() {
			var platform, minVersion, updateMessage string
			var latestVersion, storeURL *string
			var forceUpdate bool
			if err := policyRows.Scan(&platform, &minVersion, &latestVersion, &forceUpdate, &updateMessage, &storeURL); err != nil {
				continue
			}
			h.repo.db.Exec(r.Context(),
				`INSERT INTO version_policies (environment_id, platform, min_version, latest_version, force_update, update_message, store_url)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				newEnv.ID, platform, minVersion, latestVersion, forceUpdate, updateMessage, storeURL,
			)
		}
	}

	// Trigger snapshot rebuild for new environment
	go func() {
		if err := h.builder.Rebuild(context.Background(), newEnv.ID); err != nil {
			log.Error().Err(err).Str("env_id", newEnv.ID).Msg("snapshot build failed after duplication")
		}
	}()

	writeJSON(w, http.StatusCreated, newEnv)
}
