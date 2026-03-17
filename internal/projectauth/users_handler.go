package projectauth

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kennedyowusu/hatchway-api/internal/auth"
	apimw "github.com/kennedyowusu/hatchway-api/platform/middleware"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

func (h *Handler) authorizeProjectAccess(r *http.Request, projectID string) bool {
	user, ok := r.Context().Value(apimw.UserKey).(*auth.User)
	if !ok || user == nil {
		return false
	}

	var count int
	err := h.repo.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM projects
		 WHERE id = $1
		 AND org_id = (SELECT org_id FROM users WHERE id = $2)`,
		projectID, user.ID,
	).Scan(&count)

	if err != nil {
		return false
	}
	return count > 0
}

func (h *Handler) ListProjectUsers(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	if !h.authorizeProjectAccess(r, projectID) {
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
	if limit > 100 {
		limit = 100
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	users, total, err := h.repo.ListUsers(r.Context(), projectID, limit, offset)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	respond.OK(w, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) DisableProjectUser(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	userID := chi.URLParam(r, "user_id")

	if !h.authorizeProjectAccess(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.repo.SetUserDisabled(r.Context(), projectID, userID, true); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respond.Error(w, http.StatusNotFound, "user not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to disable user")
		return
	}
	respond.OK(w, map[string]string{"message": "User disabled"})
}

func (h *Handler) EnableProjectUser(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	userID := chi.URLParam(r, "user_id")

	if !h.authorizeProjectAccess(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.repo.SetUserDisabled(r.Context(), projectID, userID, false); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respond.Error(w, http.StatusNotFound, "user not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to enable user")
		return
	}
	respond.OK(w, map[string]string{"message": "User enabled"})
}

func (h *Handler) DeleteProjectUser(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	userID := chi.URLParam(r, "user_id")

	if !h.authorizeProjectAccess(r, projectID) {
		respond.Error(w, http.StatusForbidden, "access denied")
		return
	}

	if err := h.repo.DeleteUser(r.Context(), projectID, userID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respond.Error(w, http.StatusNotFound, "user not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// keep context import used
var _ = context.Background
