package invitations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kennedyowusu/hatchway-api/internal/auth"
	"github.com/kennedyowusu/hatchway-api/platform/email"
	apimiddleware "github.com/kennedyowusu/hatchway-api/platform/middleware"
	"github.com/kennedyowusu/hatchway-api/platform/respond"
	"github.com/rs/zerolog/log"
)

var (
	ErrAlreadyMember = errors.New("user is already a member of this organization")
	ErrInviteExpired = errors.New("invitation has expired")
	ErrInviteUsed    = errors.New("invitation has already been used")
	ErrInviteInvalid = errors.New("invitation is invalid")
)

const inviteTTL = 48 * time.Hour

type Invitation struct {
	ID         string     `json:"id"`
	OrgID      string     `json:"org_id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	InvitedBy  string     `json:"invited_by"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Member struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"created_at"`
}

type Handler struct {
	db     *pgxpool.Pool
	mailer email.Provider
	appURL string
}

func NewHandler(db *pgxpool.Pool, mailer email.Provider, appURL string) *Handler {
	return &Handler{db: db, mailer: mailer, appURL: appURL}
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		respond.Error(w, http.StatusBadRequest, "email is required")
		return
	}
	if body.Role == "" {
		body.Role = "member"
	}
	if body.Role != "admin" && body.Role != "member" {
		respond.Error(w, http.StatusBadRequest, "role must be admin or member")
		return
	}

	var count int
	h.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM users WHERE email = $1 AND org_id = $2 AND deleted_at IS NULL`, body.Email, orgID).Scan(&count)
	if count > 0 {
		respond.Error(w, http.StatusConflict, "user is already a member of this organization")
		return
	}

	user := r.Context().Value(apimiddleware.UserKey).(*auth.User)
	var orgName string
	h.db.QueryRow(r.Context(), `SELECT name FROM organizations WHERE id = $1`, orgID).Scan(&orgName)

	plain, hash, err := generateToken()
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	var inv Invitation
	err = h.db.QueryRow(r.Context(),
		`INSERT INTO invitations (org_id, email, role, token_hash, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, org_id, email, role, invited_by, expires_at, created_at`,
		orgID, body.Email, body.Role, hash, user.ID, time.Now().Add(inviteTTL),
	).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create invitation")
		return
	}

	acceptURL := fmt.Sprintf("%s/invite/accept?token=%s", h.appURL, plain)
	go func() {
		if err := h.mailer.Send(context.Background(), email.Message{
			To:      body.Email,
			Subject: fmt.Sprintf("You've been invited to join %s on Koolbase", orgName),
			HTML:    inviteEmailHTML(orgName, body.Email, body.Role, acceptURL),
		}); err != nil {
			log.Error().Err(err).Str("email", body.Email).Msg("send invite email failed")
		}
	}()

	respond.OK(w, inv)
}

func (h *Handler) ValidateInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		respond.Error(w, http.StatusBadRequest, "token is required")
		return
	}

	tokenHash := hashToken(body.Token)

	var inv Invitation
	err := h.db.QueryRow(r.Context(),
		`SELECT id, org_id, email, role, accepted_at, expires_at FROM invitations WHERE token_hash = $1`,
		tokenHash,
	).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.AcceptedAt, &inv.ExpiresAt)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, ErrInviteInvalid.Error())
		return
	}
	if inv.AcceptedAt != nil {
		respond.Error(w, http.StatusBadRequest, ErrInviteUsed.Error())
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		respond.Error(w, http.StatusBadRequest, ErrInviteExpired.Error())
		return
	}

	h.db.Exec(r.Context(), `UPDATE invitations SET accepted_at = NOW() WHERE id = $1`, inv.ID)

	var existingUserID string
	h.db.QueryRow(r.Context(), `SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`, inv.Email).Scan(&existingUserID)
	if existingUserID != "" {
		h.db.Exec(r.Context(), `UPDATE users SET org_id = $1, role = $2 WHERE id = $3`, inv.OrgID, inv.Role, existingUserID)
	}

	respond.OK(w, map[string]string{
		"message": "Invitation accepted. You can now log in.",
		"email":   inv.Email,
		"org_id":  inv.OrgID,
	})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	rows, err := h.db.Query(r.Context(),
		`SELECT id, email, role, verified, created_at FROM users
		 WHERE org_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`,
		orgID,
	)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	defer rows.Close()

	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Email, &m.Role, &m.Verified, &m.CreatedAt); err != nil {
			continue
		}
		members = append(members, m)
	}
	respond.OK(w, members)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")
	userID := chi.URLParam(r, "user_id")

	var role string
	h.db.QueryRow(r.Context(), `SELECT role FROM users WHERE id = $1 AND org_id = $2`, userID, orgID).Scan(&role)
	if role == "owner" {
		respond.Error(w, http.StatusForbidden, "cannot remove the organization owner")
		return
	}

	h.db.Exec(r.Context(), `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND org_id = $2`, userID, orgID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org_id")

	rows, err := h.db.Query(r.Context(),
		`SELECT id, org_id, email, role, invited_by, accepted_at, expires_at, created_at
		 FROM invitations WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	defer rows.Close()

	invites := []Invitation{}
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			continue
		}
		invites = append(invites, inv)
	}
	respond.OK(w, invites)
}

func generateToken() (plain string, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hashed = hashToken(plain)
	return plain, hashed, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
