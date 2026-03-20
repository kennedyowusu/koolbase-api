package functions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kennedyowusu/hatchway-api/internal/auth"
	apimw "github.com/kennedyowusu/hatchway-api/platform/middleware"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	svc  *Service
	repo *Repository
}

func NewHandler(svc *Service, repo *Repository) *Handler {
	return &Handler{svc: svc, repo: repo}
}

func (h *Handler) authorizeProject(r *http.Request, projectID string) bool {
	user, ok := r.Context().Value(apimw.UserKey).(*auth.User)
	if !ok || user == nil {
		return false
	}
	ok, _ = h.repo.AuthorizeProject(r.Context(), projectID, user.OrgID)
	return ok
}

func (h *Handler) resolveAPIKey(r *http.Request) (projectID string, apiKey string, err error) {
	apiKey = r.Header.Get("x-api-key")
	if apiKey == "" {
		return "", "", errors.New("x-api-key header is required")
	}
	projectID, err = h.repo.GetProjectByAPIKey(r.Context(), apiKey)
	return projectID, apiKey, err
}

// Dashboard endpoints

func (h *Handler) ListFunctions(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	fns, err := h.svc.ListFunctions(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list functions")
		return
	}
	respond.OK(w, fns)
}

func (h *Handler) DeployFunction(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fn, err := h.svc.Deploy(r.Context(), projectID, req.Name, req.Code, req.TimeoutMs)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, fn)
}

func (h *Handler) DeleteFunction(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	name := chi.URLParam(r, "function_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.svc.DeleteFunction(r.Context(), projectID, name); err != nil {
		if errors.Is(err, ErrFunctionNotFound) {
			respond.Error(w, http.StatusNotFound, "function not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete function")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	functionID := r.URL.Query().Get("function_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	logs, err := h.svc.ListLogs(r.Context(), projectID, functionID, limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list logs")
		return
	}
	respond.OK(w, logs)
}

func (h *Handler) CreateTrigger(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var req CreateTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trigger, err := h.svc.CreateTrigger(r.Context(), projectID, req)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, trigger)
}

func (h *Handler) ListTriggers(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	triggers, err := h.svc.ListTriggers(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list triggers")
		return
	}
	respond.OK(w, triggers)
}

func (h *Handler) DeleteTrigger(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	triggerID := chi.URLParam(r, "trigger_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.svc.DeleteTrigger(r.Context(), projectID, triggerID); err != nil {
		if errors.Is(err, ErrTriggerNotFound) {
			respond.Error(w, http.StatusNotFound, "trigger not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete trigger")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SDK endpoint — invoke a function via API key

func (h *Handler) SDKInvoke(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	projectID, apiKey, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	name := chi.URLParam(r, "function_name")

	var req InvokeRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}

	// Fix 4 — full normalized header map
	req.Headers = map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			req.Headers[strings.ToLower(k)] = v[0]
		}
	}

	res, err := h.svc.Invoke(r.Context(), projectID, name, apiKey, req)
	if err != nil {
		if errors.Is(err, ErrFunctionNotFound) {
			respond.Error(w, http.StatusNotFound, "function not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.Status)
	json.NewEncoder(w).Encode(res.Body)
}
