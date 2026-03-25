package ota

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kennedyowusu/hatchway-api/internal/storage"
)

const maxBundleSize = 50 * 1024 * 1024 // 50MB

type Handler struct {
	repo *Repository
	r2   *storage.R2Client
}

func NewHandler(repo *Repository, r2 *storage.R2Client) *Handler {
	return &Handler{repo: repo, r2: r2}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	if err := r.ParseMultipartForm(maxBundleSize); err != nil {
		http.Error(w, `{"error":"bundle too large (max 50MB)"}`, http.StatusRequestEntityTooLarge)
		return
	}

	file, _, err := r.FormFile("bundle")
	if err != nil {
		http.Error(w, `{"error":"bundle file required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBundleSize))
	if err != nil {
		http.Error(w, `{"error":"failed to read bundle"}`, http.StatusInternalServerError)
		return
	}

	// Compute SHA-256 checksum
	sum := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(sum[:])

	channel := r.FormValue("channel")
	if channel == "" {
		channel = "production"
	}

	mandatory := r.FormValue("mandatory") == "true"
	releaseNotes := r.FormValue("release_notes")

	ctx := r.Context()

	version, err := h.repo.NextVersion(ctx, projectID, channel)
	if err != nil {
		http.Error(w, `{"error":"failed to get next version"}`, http.StatusInternalServerError)
		return
	}

	storagePath := fmt.Sprintf("ota/%s/%s/v%d.zip", projectID, channel, version)

	if err := h.r2.PutObject(ctx, storagePath, "application/zip", bytes.NewReader(data), int64(len(data))); err != nil {
		http.Error(w, `{"error":"failed to upload bundle"}`, http.StatusInternalServerError)
		return
	}

	var rn *string
	if releaseNotes != "" {
		rn = &releaseNotes
	}

	bundle, err := h.repo.Create(ctx, Bundle{
		ProjectID:    projectID,
		Channel:      channel,
		Version:      version,
		Checksum:     checksum,
		StoragePath:  storagePath,
		FileSize:     int64(len(data)),
		Mandatory:    mandatory,
		Active:       false,
		ReleaseNotes: rn,
	})
	if err != nil {
		http.Error(w, `{"error":"failed to save bundle"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bundle)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	bundles, err := h.repo.List(r.Context(), projectID)
	if err != nil {
		http.Error(w, `{"error":"failed to list bundles"}`, http.StatusInternalServerError)
		return
	}
	if bundles == nil {
		bundles = []Bundle{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundles)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	bundleID := chi.URLParam(r, "bundle_id")

	bundle, err := h.repo.GetByID(r.Context(), projectID, bundleID)
	if err != nil {
		http.Error(w, `{"error":"bundle not found"}`, http.StatusNotFound)
		return
	}

	if err := h.repo.SetActive(r.Context(), projectID, bundle.Channel, bundleID); err != nil {
		http.Error(w, `{"error":"failed to activate bundle"}`, http.StatusInternalServerError)
		return
	}

	bundle.Active = true
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundle)
}

func (h *Handler) UpdateMandatory(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	bundleID := chi.URLParam(r, "bundle_id")

	var body struct {
		Mandatory bool `json:"mandatory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.repo.GetByID(r.Context(), projectID, bundleID); err != nil {
		http.Error(w, `{"error":"bundle not found"}`, http.StatusNotFound)
		return
	}

	if err := h.repo.UpdateMandatory(r.Context(), bundleID, body.Mandatory); err != nil {
		http.Error(w, `{"error":"failed to update bundle"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	bundleID := chi.URLParam(r, "bundle_id")

	bundle, err := h.repo.GetByID(r.Context(), projectID, bundleID)
	if err != nil {
		http.Error(w, `{"error":"bundle not found"}`, http.StatusNotFound)
		return
	}

	h.r2.DeleteObject(r.Context(), bundle.StoragePath)

	if err := h.repo.Delete(r.Context(), projectID, bundleID); err != nil {
		http.Error(w, `{"error":"failed to delete bundle"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SDKCheck(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		http.Error(w, `{"error":"x-api-key required"}`, http.StatusUnauthorized)
		return
	}

	projectID, _, err := h.repo.GetProjectIDByAPIKey(r.Context(), apiKey)
	if err != nil {
		http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
		return
	}

	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "production"
	}

	currentVersion := 0
	fmt.Sscanf(r.URL.Query().Get("version"), "%d", &currentVersion)

	bundle, err := h.repo.GetActive(r.Context(), projectID, channel)
	if err != nil || bundle == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"has_update": false})
		return
	}

	if bundle.Version <= currentVersion {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"has_update": false})
		return
	}

	downloadURL, err := h.r2.GenerateDownloadURL(r.Context(), bundle.StoragePath)
	if err != nil {
		http.Error(w, `{"error":"failed to generate download URL"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_update":    true,
		"version":       bundle.Version,
		"checksum":      bundle.Checksum,
		"mandatory":     bundle.Mandatory,
		"download_url":  downloadURL,
		"release_notes": bundle.ReleaseNotes,
		"file_size":     bundle.FileSize,
	})
}
