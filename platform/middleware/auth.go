package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserKey contextKey = "user"
const OrgIDKey contextKey = "org_id"
const ActorIDKey contextKey = "actor_id"

type SessionValidator interface {
	ValidateSession(ctx context.Context, rawToken string) (interface{}, error)
}

func RequireAuth(svc interface {
	ValidateSession(ctx context.Context, rawToken string) (interface{}, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			user, err := svc.ValidateSession(r.Context(), token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, user)
		type userWithIDs interface {
			GetID() string
			GetOrgID() string
		}
		if u, ok := user.(userWithIDs); ok {
			ctx = context.WithValue(ctx, ActorIDKey, u.GetID())
			ctx = context.WithValue(ctx, OrgIDKey, u.GetOrgID())
		}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
