package organizations

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

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Repository ---

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, name, slug string) (*Organization, error) {
	var org Organization
	err := r.db.QueryRow(ctx,
		`INSERT INTO organizations (name, slug)
		 VALUES ($1, $2)
		 RETURNING id, name, slug, created_at, updated_at`,
		name, slug,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return &org, nil
}

func (r *Repository) List(ctx context.Context) ([]Organization, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, slug, created_at, updated_at
		 FROM organizations
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

// --- Handler ---

type Handler struct {
	repo *Repository
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{repo: NewRepository(db)}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	slug := toSlug(body.Name)

	org, err := h.repo.Create(r.Context(), body.Name, slug)
	if err != nil {
		log.Error().Err(err).Msg("create organization failed")
		writeError(w, http.StatusInternalServerError, "failed to create organization")
		return
	}

	writeJSON(w, http.StatusCreated, org)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list organizations")
		return
	}
	if orgs == nil {
		orgs = []Organization{}
	}
	writeJSON(w, http.StatusOK, orgs)
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

// CreateOrg satisfies the auth.OrgCreator interface.
func (h *Handler) CreateOrg(ctx context.Context, name string) (string, error) {
	org, err := h.repo.Create(ctx, name, toSlug(name))
	if err != nil {
		return "", err
	}
	return org.ID, nil
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var org struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Slug      string    `json:"slug"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	err := h.repo.db.QueryRow(r.Context(),
		`UPDATE organizations SET name = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, slug, created_at, updated_at`,
		orgID, body.Name,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update organization")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")
	rows, err := h.repo.db.Query(r.Context(),
		`SELECT id, name, slug, created_at, updated_at FROM organizations WHERE id = $1`, orgID)
	if err != nil || !rows.Next() {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	var org struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Slug      string    `json:"slug"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	rows.Close()
	writeJSON(w, http.StatusOK, org)
}
