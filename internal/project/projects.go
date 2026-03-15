package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// --- Model ---

type Project struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"org_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// --- Repository ---

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, orgID, name, slug string) (*Project, error) {
	var p Project
	err := r.db.QueryRow(ctx,
		`INSERT INTO projects (org_id, name, slug)
		 VALUES ($1, $2, $3)
		 RETURNING id, org_id, name, slug, created_at, updated_at`,
		orgID, name, slug,
	).Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return &p, nil
}

func (r *Repository) ListByOrg(ctx context.Context, orgID string) ([]Project, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, org_id, name, slug, created_at, updated_at
		 FROM projects
		 WHERE org_id = $1
		 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// --- Handler ---

type Handler struct {
	repo *Repository
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{repo: NewRepository(db)}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := toSlug(body.Name)

	project, err := h.repo.Create(r.Context(), orgID, body.Name, slug)
	if err != nil {
		log.Error().Err(err).Msg("create project failed")
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	projects, err := h.repo.ListByOrg(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if projects == nil {
		projects = []Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

// --- Helpers ---

func toSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
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

func (r *Repository) Delete(ctx context.Context, projectID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, projectID)
	return err
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}

	if err := h.repo.Delete(r.Context(), projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
