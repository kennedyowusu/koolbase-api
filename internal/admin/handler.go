package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kennedyowusu/hatchway-api/internal/bootstrap"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	builder      *bootstrap.SnapshotBuilder
	snapshotRepo *bootstrap.SnapshotRepository
	repo         *Repository
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	snapshotRepo := bootstrap.NewSnapshotRepository(rdb)
	builder := bootstrap.NewSnapshotBuilder(db, snapshotRepo)
	return &Handler{
		builder:      builder,
		snapshotRepo: snapshotRepo,
		repo:         NewRepository(db),
	}
}

// ─── Existing ──────────────────────────────────────────────────────────────

func (h *Handler) RebuildSnapshot(w http.ResponseWriter, r *http.Request) {
	environmentID := chi.URLParam(r, "environment_id")
	if environmentID == "" {
		http.Error(w, `{"error":"environment_id is required"}`, http.StatusBadRequest)
		return
	}
	if err := h.builder.Rebuild(r.Context(), environmentID); err != nil {
		log.Error().Err(err).Str("environment_id", environmentID).Msg("snapshot rebuild failed")
		http.Error(w, `{"error":"rebuild failed"}`, http.StatusInternalServerError)
		return
	}
	log.Info().Str("environment_id", environmentID).Msg("snapshot rebuilt via internal API")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","message":"snapshot rebuilt successfully"}`))
}

// ─── Admin Endpoints ───────────────────────────────────────────────────────

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats(r.Context())
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}
	respond.JSON(w, http.StatusOK, stats)
}

func (h *Handler) GetOrgs(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	result, err := h.repo.GetOrgs(r.Context(), page, limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch orgs")
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	emailSearch := r.URL.Query().Get("email")
	result, err := h.repo.GetUsers(r.Context(), page, limit, emailSearch)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetProjects(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	result, err := h.repo.GetProjects(r.Context(), page, limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch projects")
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetFunctionLogs(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	status := r.URL.Query().Get("status")
	result, err := h.repo.GetFunctionLogs(r.Context(), page, limit, status)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch function logs")
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

func (h *Handler) GetDeadLetters(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	result, err := h.repo.GetDeadLetters(r.Context(), page, limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch dead letters")
		return
	}
	respond.JSON(w, http.StatusOK, result)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return page, limit
}

// Ensure fmt is used
var _ = fmt.Sprintf
