package storage

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

func (h *Handler) authorizeProject(r *http.Request, projectID string) bool {
	user, ok := r.Context().Value(apimw.UserKey).(*auth.User)
	if !ok || user == nil {
		return false
	}
	ok, _ = h.repo.GetProjectIDByDashboardUser(r.Context(), projectID, user.OrgID)
	return ok
}

// Dashboard endpoints — require dashboard auth

func (h *Handler) CreateBucket(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	var body struct {
		Name   string `json:"name"`
		Public bool   `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		respond.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	bucket, err := h.svc.CreateBucket(r.Context(), projectID, body.Name, body.Public)
	if err != nil {
		if errors.Is(err, ErrBucketExists) {
			respond.Error(w, http.StatusConflict, "bucket already exists")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, bucket)
}

func (h *Handler) ListBuckets(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	buckets, err := h.svc.ListBuckets(r.Context(), projectID)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list buckets")
		return
	}
	respond.OK(w, buckets)
}

func (h *Handler) DeleteBucket(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	bucketName := chi.URLParam(r, "bucket_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.svc.DeleteBucket(r.Context(), projectID, bucketName); err != nil {
		if errors.Is(err, ErrBucketNotFound) {
			respond.Error(w, http.StatusNotFound, "bucket not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete bucket")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListObjects(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	bucketName := chi.URLParam(r, "bucket_name")
	if !h.authorizeProject(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	prefix := r.URL.Query().Get("prefix")
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	objects, total, err := h.svc.ListObjects(r.Context(), projectID, bucketName, prefix, limit, offset)
	if err != nil {
		if errors.Is(err, ErrBucketNotFound) {
			respond.Error(w, http.StatusNotFound, "bucket not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to list objects")
		return
	}
	respond.OK(w, map[string]interface{}{
		"objects": objects,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// SDK endpoints — use x-api-key

func (h *Handler) resolveProject(r *http.Request) (projectID string, err error) {
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		return "", errors.New("x-api-key header is required")
	}
	projectID, _, err = h.repo.GetProjectIDByAPIKey(r.Context(), apiKey)
	return projectID, err
}

func (h *Handler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req UploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Bucket == "" || req.Path == "" {
		respond.Error(w, http.StatusBadRequest, "bucket and path are required")
		return
	}
	if req.ContentType == "" {
		req.ContentType = "application/octet-stream"
	}

	// Get user ID from JWT if provided
	userID := ""
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Optional — user ID from Bearer token
		userID = r.Header.Get("x-user-id")
	}

	res, err := h.svc.GetUploadURL(r.Context(), projectID, userID, req.Bucket, req.Path, req.ContentType)
	if err != nil {
		if errors.Is(err, ErrBucketNotFound) {
			respond.Error(w, http.StatusNotFound, "bucket not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, res)
}

func (h *Handler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Bucket == "" || req.Path == "" {
		respond.Error(w, http.StatusBadRequest, "bucket and path are required")
		return
	}

	userID := r.Header.Get("x-user-id")
	obj, err := h.svc.ConfirmUpload(r.Context(), projectID, userID, req.Bucket, req.Path, req.ContentType, req.ETag, req.Size)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, obj)
}

func (h *Handler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	path := r.URL.Query().Get("path")
	if bucket == "" || path == "" {
		respond.Error(w, http.StatusBadRequest, "bucket and path are required")
		return
	}

	res, err := h.svc.GetDownloadURL(r.Context(), projectID, bucket, path)
	if err != nil {
		if errors.Is(err, ErrBucketNotFound) {
			respond.Error(w, http.StatusNotFound, "bucket not found")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, res)
}

func (h *Handler) DeleteObject(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Bucket == "" || req.Path == "" {
		respond.Error(w, http.StatusBadRequest, "bucket and path are required")
		return
	}

	if err := h.svc.DeleteObject(r.Context(), projectID, req.Bucket, req.Path); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
