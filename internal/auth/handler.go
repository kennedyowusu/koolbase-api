package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kennedyowusu/hatchway-api/platform/middleware"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.OrgName == "" {
		respond.Error(w, http.StatusBadRequest, "email, password, and org_name are required")
		return
	}
	if len(req.Password) < 8 {
		respond.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	user, err := h.svc.Signup(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			respond.Error(w, http.StatusConflict, "email already in use")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "signup failed")
		return
	}

	respond.Created(w, map[string]any{
		"message": "Account created. Please check your email to verify your account.",
		"user_id": user.ID,
	})
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		respond.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.svc.VerifyEmail(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrTokenInvalid):
			respond.Error(w, http.StatusBadRequest, "invalid token")
		case errors.Is(err, ErrTokenUsed):
			respond.Error(w, http.StatusBadRequest, "token has already been used")
		case errors.Is(err, ErrTokenExpired):
			respond.Error(w, http.StatusBadRequest, "token has expired")
		default:
			respond.Error(w, http.StatusInternalServerError, "verification failed")
		}
		return
	}

	respond.OK(w, map[string]string{"message": "Email verified. You can now log in."})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	authResp, err := h.svc.Login(r.Context(), req, realIP(r), r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			respond.Error(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, ErrEmailNotVerified):
			respond.Error(w, http.StatusForbidden, "please verify your email before logging in")
		default:
			respond.Error(w, http.StatusInternalServerError, "login failed")
		}
		return
	}

	respond.OK(w, authResp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		respond.Error(w, http.StatusUnauthorized, "missing token")
		return
	}

	if err := h.svc.Logout(r.Context(), token); err != nil {
		respond.Error(w, http.StatusInternalServerError, "logout failed")
		return
	}

	respond.NoContent(w)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	_ = h.svc.ForgotPassword(r.Context(), req)
	respond.OK(w, map[string]string{"message": "If that email exists, a reset link has been sent."})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		respond.Error(w, http.StatusBadRequest, "token and password are required")
		return
	}
	if len(req.Password) < 8 {
		respond.Error(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrTokenInvalid):
			respond.Error(w, http.StatusBadRequest, "invalid token")
		case errors.Is(err, ErrTokenUsed):
			respond.Error(w, http.StatusBadRequest, "token has already been used")
		case errors.Is(err, ErrTokenExpired):
			respond.Error(w, http.StatusBadRequest, "reset link has expired — please request a new one")
		default:
			respond.Error(w, http.StatusInternalServerError, "password reset failed")
		}
		return
	}

	respond.OK(w, map[string]string{"message": "Password updated. You can now log in."})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}


func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserKey).(*User)
	if !ok || user == nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	respond.OK(w, user)
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserKey).(*User)
	if !ok || user == nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}
	updated, err := h.svc.UpdateUser(r.Context(), user.ID, body.Email)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	respond.OK(w, updated)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserKey).(*User)
	if !ok || user == nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.CurrentPassword == "" || body.NewPassword == "" {
		respond.Error(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}
	if err := h.svc.ChangePassword(r.Context(), user.ID, body.CurrentPassword, body.NewPassword); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	respond.OK(w, map[string]string{"message": "password updated"})
}

func realIP(r *http.Request) string {
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return strings.Split(ip, ",")[0]
    }
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    return r.RemoteAddr
}

func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserKey).(*User)
	if !ok || user == nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.svc.RequestEmailChange(r.Context(), user.ID, body.Email); err != nil {
		if err == ErrEmailTaken {
			respond.Error(w, http.StatusConflict, "email already in use")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "failed to request email change")
		return
	}

	respond.OK(w, map[string]string{"message": "verification email sent to new address"})
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		respond.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.svc.ConfirmEmailChange(r.Context(), body.Token); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	respond.OK(w, map[string]string{"message": "email updated successfully"})
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserKey).(*User)
	if !ok || user == nil {
		respond.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.DeleteAccount(r.Context(), user.ID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete account")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
