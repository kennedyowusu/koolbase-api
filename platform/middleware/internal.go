package middleware

import (
	"net/http"
	"os"
)

func InternalOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := os.Getenv("INTERNAL_API_KEY")
		if key == "" || r.Header.Get("X-Internal-Key") != key {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
