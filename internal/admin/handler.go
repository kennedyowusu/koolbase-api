package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/kennedyowusu/hatchway-api/internal/bootstrap"
)

type Handler struct {
	builder      *bootstrap.SnapshotBuilder
	snapshotRepo *bootstrap.SnapshotRepository
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	snapshotRepo := bootstrap.NewSnapshotRepository(rdb)
	builder := bootstrap.NewSnapshotBuilder(db, snapshotRepo)

	return &Handler{
		builder:      builder,
		snapshotRepo: snapshotRepo,
	}
}

// RebuildSnapshot triggers a fresh snapshot build for an environment.
// POST /internal/environments/{environment_id}/snapshot/rebuild
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
