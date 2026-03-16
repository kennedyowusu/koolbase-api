package auditlog

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

func (w *Writer) HandleList(rw http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

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

	logs, total, err := w.List(r.Context(), orgID, limit, offset)
	if err != nil {
		respond.Error(rw, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	respond.OK(rw, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
