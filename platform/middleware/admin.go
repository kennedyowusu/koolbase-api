package middleware

import (
	"net/http"
	"os"
	"strings"
)

type adminUser interface {
	GetEmail() string
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminEmail := os.Getenv("ADMIN_EMAIL")
		if adminEmail == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"admin not configured"}`))
			return
		}

		userVal := r.Context().Value(UserKey)
		if userVal == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		// Try interface with GetEmail()
		type emailer interface{ GetEmail() string }
		if u, ok := userVal.(emailer); ok {
			allowed := false
			for _, e := range strings.Split(adminEmail, ",") {
							if strings.EqualFold(u.GetEmail(), strings.TrimSpace(e)) {
											allowed = true
											break
							}
			}
			if !allowed {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusForbidden)
							w.Write([]byte(`{"error":"forbidden"}`))
							return
			}
			next.ServeHTTP(w, r)
			return
		}

		type emailStruct struct{ Email string }
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	})
}

func AdminIPWhitelist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedIPs := os.Getenv("ADMIN_ALLOWED_IPS")
		if allowedIPs == "" {
			comingSoon(w)
			return
		}

		clientIP := extractClientIP(r)
		for _, ip := range strings.Split(allowedIPs, ",") {
			if strings.TrimSpace(ip) == clientIP {
				next.ServeHTTP(w, r)
				return
			}
		}

		comingSoon(w)
	})
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return strings.TrimPrefix(strings.TrimSuffix(ip, "]"), "[")
}

func comingSoon(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Koolbase</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      background: #0a0a0a;
      color: #ffffff;
      font-family: -apple-system, BlinkMacSystemFont, 'Inter', sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
    }
    .container { text-align: center; }
    .logo { font-size: 1.5rem; font-weight: 700; color: #3b82f6; margin-bottom: 1.5rem; }
    h1 { font-size: 2rem; font-weight: 800; margin-bottom: 0.75rem; }
    p { color: #94a3b8; font-size: 1rem; }
  </style>
</head>
<body>
  <div class="container">
    <div class="logo">Koolbase</div>
    <h1>Coming Soon</h1>
    <p>Something great is on the way.</p>
  </div>
</body>
</html>`))
}
