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
		if strings.Contains(err.Error(), "http: request body too large") {
			respond.Error(w, http.StatusRequestEntityTooLarge, "function code too large: maximum 1MB")
			return
		}
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	runtime := req.Runtime
	if runtime == "" {
		runtime = "deno"
	}
	fn, err := h.svc.Deploy(r.Context(), projectID, req.Name, runtime, req.Code, req.TimeoutMs)
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
			if strings.Contains(err.Error(), "http: request body too large") {
				respond.Error(w, http.StatusRequestEntityTooLarge, "payload too large: maximum 1MB")
				return
			}
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}

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

func (h *Handler) ListDeadLetters(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	letters, err := h.repo.ListDeadLetters(r.Context(), projectID, limit)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list dead letters")
		return
	}
	respond.OK(w, letters)
}

func (h *Handler) DeleteDeadLetter(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	id := chi.URLParam(r, "id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.repo.DeleteDeadLetter(r.Context(), projectID, id); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete dead letter")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ReplayDeadLetter(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	id := chi.URLParam(r, "id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	dl, err := h.repo.ReplayDeadLetter(r.Context(), projectID, id)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "dead letter not found")
		return
	}

	// Re-enqueue as a fresh retry
	if err := h.repo.EnqueueRetry(r.Context(), projectID, dl.FunctionName,
		dl.EventType, dl.Collection, "", "replayed from DLQ", dl.Payload); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to enqueue replay")
		return
	}

	h.repo.DeleteDeadLetter(r.Context(), projectID, id)

	respond.OK(w, map[string]string{"status": "replayed"})
}

func (h *Handler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}
	secrets, err := h.repo.ListSecrets(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list secrets")
		return
	}
	respond.OK(w, secrets)
}

func (h *Handler) UpsertSecret(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var req UpsertSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Value == "" {
		respond.Error(w, http.StatusBadRequest, "name and value are required")
		return
	}
	if len(req.Name) > 64 {
		respond.Error(w, http.StatusBadRequest, "secret name too long")
		return
	}

	encrypted, err := Encrypt(req.Value)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to encrypt secret")
		return
	}

	secret, err := h.repo.UpsertSecret(r.Context(), projectID, req.Name, encrypted)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to save secret")
		return
	}
	respond.OK(w, secret)
}

func (h *Handler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	name := chi.URLParam(r, "secret_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}
	if err := h.repo.DeleteSecret(r.Context(), projectID, name); err != nil {
		respond.Error(w, http.StatusNotFound, "secret not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetTriggerStats(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	stats, err := h.repo.GetTriggerStats(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to get trigger stats")
		return
	}
	respond.OK(w, stats)
}

func (h *Handler) DashboardInvoke(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	projectID := chi.URLParam(r, "project_id")
	name := chi.URLParam(r, "function_name")

	var req InvokeRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				respond.Error(w, http.StatusRequestEntityTooLarge, "payload too large: maximum 1MB")
				return
			}
			respond.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}
	req.Headers = map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			req.Headers[strings.ToLower(k)] = v[0]
		}
	}

	res, err := h.svc.Invoke(r.Context(), projectID, name, "", req)
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
