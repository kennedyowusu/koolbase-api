package flags

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/kennedyowusu/hatchway-api/platform/events"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	repo *Repository
	bus  *events.Bus
}

func NewHandler(db *pgxpool.Pool, bus *events.Bus) *Handler {
	return &Handler{repo: NewRepository(db), bus: bus}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")
	var body struct {
		Key               string `json:"key"`
		Enabled           bool   `json:"enabled"`
		RolloutPercentage int    `json:"rollout_percentage"`
		Description       string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		respond.Error(w, http.StatusBadRequest, "key is required")
		return
	}
	flag, err := h.repo.Create(r.Context(), envID, body.Key, body.Enabled, body.RolloutPercentage, body.Description)
	if err != nil {
		log.Error().Err(err).Msg("create flag failed")
		respond.Error(w, http.StatusInternalServerError, "failed to create flag")
		return
	}
	h.bus.Publish(events.Event{Type: events.FlagCreated, Payload: events.SnapshotPayload{EnvironmentID: envID}})
	respond.Created(w, flag)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flag_id")
	var body struct {
		Enabled           bool   `json:"enabled"`
		RolloutPercentage int    `json:"rollout_percentage"`
		KillSwitch        bool   `json:"kill_switch"`
		Description       string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	flag, err := h.repo.Update(r.Context(), flagID, body.Enabled, body.RolloutPercentage, body.KillSwitch, body.Description)
	if err != nil {
		log.Error().Err(err).Msg("update flag failed")
		respond.Error(w, http.StatusInternalServerError, "failed to update flag")
		return
	}
	h.bus.Publish(events.Event{Type: events.FlagUpdated, Payload: events.SnapshotPayload{EnvironmentID: flag.EnvironmentID}})
	respond.OK(w, flag)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flag_id")
	envID, err := h.repo.Delete(r.Context(), flagID)
	if err != nil {
		log.Error().Err(err).Msg("delete flag failed")
		respond.Error(w, http.StatusInternalServerError, "failed to delete flag")
		return
	}
	h.bus.Publish(events.Event{Type: events.FlagDeleted, Payload: events.SnapshotPayload{EnvironmentID: envID}})
	respond.NoContent(w)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	envID := chi.URLParam(r, "env_id")
	list, err := h.repo.ListByEnvironment(r.Context(), envID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list flags")
		return
	}
	if list == nil {
		list = []Flag{}
	}
	respond.OK(w, list)
}
