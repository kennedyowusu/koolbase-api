package cron

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	var req CreateCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	schedule, err := h.svc.Create(r.Context(), projectID, req)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	respond.Created(w, schedule)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	schedules, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list cron schedules")
		return
	}

	respond.OK(w, schedules)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	cronID := chi.URLParam(r, "cron_id")

	if err := h.svc.Delete(r.Context(), projectID, cronID); err != nil {
		if errors.Is(err, ErrCronNotFound) {
			respond.Error(w, http.StatusNotFound, "cron schedule not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete cron schedule")
		return
	}

	respond.NoContent(w)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	cronID := chi.URLParam(r, "cron_id")

	var req UpdateCronRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	schedule, err := h.svc.Update(r.Context(), projectID, cronID, req)
	if err != nil {
		if errors.Is(err, ErrCronNotFound) {
			respond.Error(w, http.StatusNotFound, "cron schedule not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	respond.OK(w, schedule)
}
