package database

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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

func (h *Handler) resolveAPIKey(r *http.Request) (projectID string, err error) {
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		return "", errors.New("x-api-key header is required")
	}
	projectID, _, err = h.repo.GetProjectByAPIKey(r.Context(), apiKey)
	return projectID, err
}

func (h *Handler) getUserID(r *http.Request) string {
	return r.Header.Get("x-user-id")
}

func (h *Handler) authorizeProject(r *http.Request, projectID string) bool {
	user, ok := r.Context().Value(apimw.UserKey).(*auth.User)
	if !ok || user == nil {
		return false
	}
	ok, _ = h.repo.AuthorizeProject(r.Context(), projectID, user.OrgID)
	return ok
}

// Dashboard endpoints

func (h *Handler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var req CreateCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		respond.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	col, err := h.svc.CreateCollection(r.Context(), projectID, req)
	if err != nil {
		if errors.Is(err, ErrCollectionExists) {
			respond.Error(w, http.StatusConflict, "collection already exists")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, col)
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	collections, err := h.svc.ListCollections(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list collections")
		return
	}
	respond.OK(w, collections)
}

func (h *Handler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	name := chi.URLParam(r, "collection_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.svc.DeleteCollection(r.Context(), projectID, name); err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			respond.Error(w, http.StatusNotFound, "collection not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete collection")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListRecordsDashboard(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	collectionName := chi.URLParam(r, "collection_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	records, total, err := h.svc.Query(r.Context(), projectID, "", QueryRequest{
		Collection: collectionName,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, map[string]interface{}{
		"records": records,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// SDK endpoints

func (h *Handler) SDKInsert(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req InsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rec, err := h.svc.Insert(r.Context(), projectID, h.getUserID(r), req)
	if err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			respond.Error(w, http.StatusNotFound, "collection not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, rec)
}

func (h *Handler) SDKQuery(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	records, total, err := h.svc.Query(r.Context(), projectID, h.getUserID(r), req)
	if err != nil {
		if errors.Is(err, ErrCollectionNotFound) {
			respond.Error(w, http.StatusNotFound, "collection not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, map[string]interface{}{
		"records": records,
		"total":   total,
	})
}

func (h *Handler) SDKGet(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	recordID := chi.URLParam(r, "record_id")
	rec, err := h.svc.Get(r.Context(), projectID, h.getUserID(r), recordID)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			respond.Error(w, http.StatusNotFound, "record not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, rec)
}

func (h *Handler) SDKUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	recordID := chi.URLParam(r, "record_id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rec, err := h.svc.Update(r.Context(), projectID, h.getUserID(r), recordID, req)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			respond.Error(w, http.StatusNotFound, "record not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, rec)
}

func (h *Handler) SDKDelete(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveAPIKey(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	recordID := chi.URLParam(r, "record_id")
	if err := h.svc.Delete(r.Context(), projectID, h.getUserID(r), recordID); err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			respond.Error(w, http.StatusNotFound, "record not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
