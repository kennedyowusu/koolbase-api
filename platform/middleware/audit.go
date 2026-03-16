package middleware

import (
	"net/http"
	"strings"

	"github.com/kennedyowusu/hatchway-api/internal/auditlog"
)

type userWithID interface {
	GetID() string
}

func AuditLog(writer *auditlog.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			if r.Method != http.MethodPost && r.Method != http.MethodPut &&
				r.Method != http.MethodPatch && r.Method != http.MethodDelete {
				return
			}

			if strings.Contains(r.URL.Path, "/auth/") {
				return
			}

			// Get actor ID from context without importing auth package
			actorID := ""
			if v := r.Context().Value(UserKey); v != nil {
				if u, ok := v.(userWithID); ok {
					actorID = u.GetID()
				}
			}

			action, resourceType, resourceID := inferAction(r)
			if action == "" {
				return
			}

			writer.Write(r.Context(), auditlog.Entry{
				ActorID:      actorID,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Action:       action,
				IP:           r.RemoteAddr,
			})
		})
	}
}

func inferAction(r *http.Request) (auditlog.Action, string, string) {
	path := r.URL.Path
	method := r.Method
	parts := strings.Split(strings.Trim(path, "/"), "/")

	var resourceType, resourceID string
	for i, part := range parts {
		switch part {
		case "flags":
			resourceType = "flag"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		case "configs":
			resourceType = "config"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		case "version":
			resourceType = "version"
		case "environments":
			resourceType = "environment"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		case "projects":
			resourceType = "project"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		case "invites":
			resourceType = "invite"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		case "members":
			resourceType = "member"
			if i+1 < len(parts) { resourceID = parts[i+1] }
		}
	}

	if resourceType == "" {
		return "", "", ""
	}

	var action auditlog.Action
	switch method {
	case http.MethodPost:
		action = auditlog.Action(resourceType + ".created")
	case http.MethodPut, http.MethodPatch:
		action = auditlog.Action(resourceType + ".updated")
	case http.MethodDelete:
		action = auditlog.Action(resourceType + ".deleted")
	}

	return action, resourceType, resourceID
}
