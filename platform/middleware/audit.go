package middleware

import (
	"net/http"
	"strings"

	"github.com/kennedyowusu/hatchway-api/internal/auditlog"
	"github.com/rs/zerolog/log"
)

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

			actorID, _ := r.Context().Value(ActorIDKey).(string)
			orgID, _ := r.Context().Value(OrgIDKey).(string)

			log.Debug().Str("actor_id", actorID).Str("org_id", orgID).Str("path", r.URL.Path).Msg("audit middleware fired")

			if orgID == "" {
				log.Debug().Msg("audit middleware: org_id empty, skipping")
				return
			}

			action, resourceType, resourceID := inferAction(r)
			if action == "" {
				return
			}

			writer.Write(r.Context(), auditlog.Entry{
				OrgID:        orgID,
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
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
		case "configs":
			resourceType = "config"
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
		case "version":
			resourceType = "version"
		case "environments":
			resourceType = "environment"
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
		case "projects":
			resourceType = "project"
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
		case "invites":
			resourceType = "invite"
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
		case "members":
			resourceType = "member"
			if i+1 < len(parts) {
				resourceID = parts[i+1]
			}
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
