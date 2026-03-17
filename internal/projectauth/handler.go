package projectauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	svc  *Service
	repo *Repository
}

func NewHandler(svc *Service, repo *Repository) *Handler {
	return &Handler{svc: svc, repo: repo}
}

func (h *Handler) resolveProject(r *http.Request) (projectID, environmentID string, err error) {
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		return "", "", errors.New("x-api-key header is required")
	}
	return h.repo.GetProjectByAPIKey(r.Context(), apiKey)
}

func (h *Handler) extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	projectID, environmentID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	session, err := h.svc.Register(r.Context(), projectID, environmentID, req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			respond.Error(w, http.StatusConflict, "email already in use")
			return
		}
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.Created(w, session)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	projectID, environmentID, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ip = strings.Split(ip, ",")[0]
		ip = strings.TrimSpace(ip)
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	} else {
		ip = r.RemoteAddr
	}
	userAgent := r.Header.Get("User-Agent")

	session, err := h.svc.Login(r.Context(), projectID, environmentID, req, ip, userAgent)
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			respond.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		if errors.Is(err, ErrUserDisabled) {
			respond.Error(w, http.StatusForbidden, "account is disabled")
			return
		}
		respond.Error(w, http.StatusUnauthorized, "login failed")
		return
	}
	respond.OK(w, session)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		respond.Error(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	session, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	respond.OK(w, session)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.extractBearerToken(r)
	if token == "" {
		respond.Error(w, http.StatusUnauthorized, "missing token")
		return
	}
	if err := h.svc.Logout(r.Context(), token); err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	token := h.extractBearerToken(r)
	if token == "" {
		respond.Error(w, http.StatusUnauthorized, "missing token")
		return
	}
	user, err := h.svc.GetUser(r.Context(), token)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	respond.OK(w, user)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	token := h.extractBearerToken(r)
	if token == "" {
		respond.Error(w, http.StatusUnauthorized, "missing token")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.UpdateUser(r.Context(), token, req)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	respond.OK(w, user)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	projectID, _, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		respond.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), projectID, body.Token); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	respond.OK(w, map[string]string{"message": "Email verified successfully"})
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	projectID, _, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	h.svc.ForgotPassword(r.Context(), projectID, req.Email)
	respond.OK(w, map[string]string{"message": "If that email exists, a reset link has been sent"})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	projectID, _, err := h.resolveProject(r)
	if err != nil {
		respond.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "token and password are required")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), projectID, req.Token, req.Password); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	respond.OK(w, map[string]string{"message": "Password reset successfully"})
}
