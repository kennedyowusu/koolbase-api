package bootstrap

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service *BootstrapService
	db      *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client) *Handler {
	envRepo := NewEnvironmentRepository(db)
	snapshotRepo := NewSnapshotRepository(rdb)
	builder := NewSnapshotBuilder(db, snapshotRepo)
	service := NewService(envRepo, snapshotRepo, builder)

	return &Handler{service: service, db: db}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	publicKey := q.Get("public_key")
	deviceID := q.Get("device_id")
	platform := q.Get("platform")
	appVersion := q.Get("app_version")

	if publicKey == "" {
		writeError(w, http.StatusBadRequest, "public_key is required")
		return
	}
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if platform == "" {
		writeError(w, http.StatusBadRequest, "platform is required")
		return
	}
	if appVersion == "" {
		writeError(w, http.StatusBadRequest, "app_version is required")
		return
	}

	snapshot, envID, err := h.service.GetSnapshot(r.Context(), publicKey)
	if err != nil {
		if errors.Is(err, ErrEnvironmentNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid public_key")
			return
		}
		log.Error().Err(err).Str("public_key", publicKey).Msg("bootstrap failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	go recordStat(context.Background(), h.db, envID, platform)
	go registerDevice(context.Background(), h.db, &Request{
		PublicKey:  publicKey,
		DeviceID:   deviceID,
		Platform:   platform,
		AppVersion: appVersion,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=30")
	w.Write(snapshot)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
