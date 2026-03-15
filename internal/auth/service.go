package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/kennedyowusu/hatchway-api/platform/email"
	"github.com/kennedyowusu/hatchway-api/platform/events"
)

var (
	ErrEmailTaken         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenUsed          = errors.New("token has already been used")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrSessionNotFound    = errors.New("session not found")
)

const (
	bcryptCost     = 12
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 1 * time.Hour
	sessionTTL     = 30 * 24 * time.Hour
)

type OrgCreator interface {
	CreateOrg(ctx context.Context, name string) (id string, err error)
}

type Service struct {
	repo   Repository
	orgSvc OrgCreator
	mailer email.Provider
	bus    *events.Bus
	appURL string
}

func NewService(repo Repository, orgSvc OrgCreator, mailer email.Provider, bus *events.Bus, appURL string) *Service {
	return &Service{repo: repo, orgSvc: orgSvc, mailer: mailer, bus: bus, appURL: appURL}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*User, error) {
	existing, _ := s.repo.GetUserByEmailIncludeDeleted(ctx, req.Email)
	if existing != nil && existing.DeletedAt == nil {
		return nil, ErrEmailTaken
	}
	if existing != nil && existing.DeletedAt != nil {
		// Reactivate soft-deleted account with new password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		if err := s.repo.ReactivateAccount(ctx, existing.ID, req.Email); err != nil {
			return nil, fmt.Errorf("reactivate account: %w", err)
		}
		if err := s.repo.UpdatePassword(ctx, existing.ID, string(hash)); err != nil {
			return nil, fmt.Errorf("update password: %w", err)
		}
		// Send verification email
		plainToken, tokenHash, err := generateToken()
		if err != nil {
			return nil, fmt.Errorf("generate verification token: %w", err)
		}
		if err := s.repo.CreateEmailVerificationToken(ctx, existing.ID, tokenHash, time.Now().Add(verifyTokenTTL)); err != nil {
			return nil, fmt.Errorf("store verification token: %w", err)
		}
		verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, plainToken)
		s.mailer.Send(ctx, email.Message{
			To:      existing.Email,
			Subject: "Verify your Koolbase account",
			HTML:    verificationEmailHTML(verifyURL),
		})
		return existing, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	orgID, err := s.orgSvc.CreateOrg(ctx, req.OrgName)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, orgID, req.Email, string(hash), "owner")
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}

	if err := s.repo.CreateEmailVerificationToken(ctx, user.ID, tokenHash, time.Now().Add(verifyTokenTTL)); err != nil {
		return nil, fmt.Errorf("store verification token: %w", err)
	}

	// Send verification email directly
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, plainToken)
	if err := s.mailer.Send(ctx, email.Message{
		To:      user.Email,
		Subject: "Verify your Koolbase account",
		HTML: verificationEmailHTML(verifyURL),
	}); err != nil {
		log.Error().Err(err).Str("email", user.Email).Msg("send verification email failed")
	}

	s.bus.Publish(events.Event{
		Type: events.UserSignedUp,
		Payload: events.UserSignedUpPayload{
			UserID: user.ID,
			Email:  user.Email,
			OrgID:  orgID,
			Token:  plainToken,
		},
	})

	return user, nil
}

func (s *Service) VerifyEmail(ctx context.Context, req VerifyEmailRequest) error {
	tokenHash := hashToken(req.Token)

	record, err := s.repo.GetEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return ErrTokenInvalid
	}
	if record.UsedAt != nil {
		return ErrTokenUsed
	}
	if time.Now().After(record.ExpiresAt) {
		return ErrTokenExpired
	}

	if err := s.repo.MarkEmailVerified(ctx, record.UserID); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}
	if err := s.repo.MarkEmailVerificationTokenUsed(ctx, record.ID); err != nil {
		log.Warn().Err(err).Msg("mark verification token used failed")
	}

	s.bus.Publish(events.Event{Type: events.UserVerifiedEmail, Payload: record.UserID})

	if user, err := s.repo.GetUserByID(ctx, record.UserID); err == nil {
    go s.mailer.Send(context.Background(), email.Message{
        To:      user.Email,
        Subject: "Welcome to Koolbase",
        HTML:    welcomeEmailHTML(user.Email, s.appURL, "https://koolbase.com/docs"),
    })
}

	return nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, ip, userAgent string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.Verified {
		return nil, ErrEmailNotVerified
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	_, err = s.repo.CreateSession(ctx, user.ID, tokenHash, time.Now().Add(sessionTTL), ip, userAgent)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	loginTime := time.Now().UTC().Format("Jan 2, 2006 at 15:04 (UTC)")
	go s.mailer.Send(context.Background(), email.Message{
		To:      user.Email,
		Subject: "New login detected on your Koolbase account",
		HTML: newLoginEmailHTML(user.Email, "Secured", "Unknown", "Web Browser", loginTime),
	})

	return &AuthResponse{AccessToken: plainToken, User: user}, nil

}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.repo.DeleteSession(ctx, hashToken(rawToken))
}

func (s *Service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	if err := s.repo.CreatePasswordResetToken(ctx, user.ID, tokenHash, time.Now().Add(resetTokenTTL)); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	// Send reset email directly
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, plainToken)
	if err := s.mailer.Send(ctx, email.Message{
		To:      user.Email,
		Subject: "Reset your Koolbase password",
		HTML: passwordResetEmailHTML(resetURL),
	}); err != nil {
		log.Error().Err(err).Str("email", user.Email).Msg("send reset email failed")
	}

	s.bus.Publish(events.Event{
		Type: events.UserRequestedReset,
		Payload: events.UserRequestedResetPayload{
			UserID: user.ID,
			Email:  user.Email,
			Token:  plainToken,
		},
	})

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)

	record, err := s.repo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return ErrTokenInvalid
	}
	if record.UsedAt != nil {
		return ErrTokenUsed
	}
	if time.Now().After(record.ExpiresAt) {
		return ErrTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, record.UserID, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.repo.MarkPasswordResetTokenUsed(ctx, record.ID); err != nil {
		log.Warn().Err(err).Msg("mark reset token used failed")
	}

	if err := s.repo.DeleteAllUserSessions(ctx, record.UserID); err != nil {
		log.Warn().Err(err).Msg("delete sessions after password reset failed")
	}

	s.bus.Publish(events.Event{Type: events.UserResetPassword, Payload: record.UserID})
	return nil
}

func (s *Service) ValidateSession(ctx context.Context, rawToken string) (interface{}, error) {
	session, err := s.repo.GetSession(ctx, hashToken(rawToken))
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return s.repo.GetUserByID(ctx, session.UserID)
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

func (s *Service) UpdateUser(ctx context.Context, userID, email string) (*User, error) {
	return s.repo.UpdateUser(ctx, userID, email)
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return s.repo.ChangePassword(ctx, userID, currentPassword, newPassword)
}

func (s *Service) RequestEmailChange(ctx context.Context, userID, newEmail string) error {
	// Check email not already taken
	existing, _ := s.repo.GetUserByEmail(ctx, newEmail)
	if existing != nil {
		return ErrEmailTaken
	}

	// Store pending email
	if err := s.repo.SetPendingEmail(ctx, userID, newEmail); err != nil {
		return fmt.Errorf("set pending email: %w", err)
	}

	// Generate verification token
	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	if err := s.repo.CreateEmailVerificationToken(ctx, userID, tokenHash, time.Now().Add(verifyTokenTTL)); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	// Send verification to new email
	verifyURL := fmt.Sprintf("%s/verify-email-change?token=%s", s.appURL, plainToken)
	if err := s.mailer.Send(context.Background(), email.Message{
		To:      newEmail,
		Subject: "Confirm your new email address",
		HTML:    verificationEmailHTML(verifyURL),
	}); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func (s *Service) ConfirmEmailChange(ctx context.Context, token string) error {
	tokenHash := hashToken(token)

	record, err := s.repo.GetEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return ErrTokenInvalid
	}
	if record.UsedAt != nil {
		return ErrTokenUsed
	}
	if time.Now().After(record.ExpiresAt) {
		return ErrTokenExpired
	}

	if err := s.repo.ConfirmEmailChange(ctx, record.UserID); err != nil {
		return fmt.Errorf("confirm email change: %w", err)
	}

	if err := s.repo.MarkEmailVerificationTokenUsed(ctx, record.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	return s.repo.DeleteAccount(ctx, userID)
}
