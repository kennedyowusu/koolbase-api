package projectauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	verifyTTL       = 24 * time.Hour
	resetTTL        = 1 * time.Hour
	bcryptCost      = 12
)

type Service struct {
	repo      *Repository
	mailer    Mailer
	jwtSecret string
	appURL    string
}

type Mailer interface {
	SendEmail(ctx context.Context, to, subject, html string) error
}

type Claims struct {
	UserID        string `json:"sub"`
	ProjectID     string `json:"project_id"`
	EnvironmentID string `json:"environment_id"`
	Email         string `json:"email"`
	jwt.RegisteredClaims
}

func NewService(repo *Repository, mailer Mailer, jwtSecret, appURL string) *Service {
	return &Service{repo: repo, mailer: mailer, jwtSecret: jwtSecret, appURL: appURL}
}

func (s *Service) Register(ctx context.Context, projectID, environmentID string, req RegisterRequest) (*Session, error) {
	if len(req.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.CreateUser(ctx, projectID, req.Email, string(hash), req.FullName)
	if err != nil {
		return nil, err
	}

	go func() {
		plainToken, tokenHash, err := generateToken()
		if err != nil {
			return
		}
		s.repo.CreateEmailVerification(context.Background(), projectID, user.ID, tokenHash, time.Now().Add(verifyTTL))
		verifyURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, plainToken)
		s.mailer.SendEmail(context.Background(), user.Email, "Verify your email", verifyEmailHTML(verifyURL))
	}()

	session, err := s.createSession(ctx, projectID, environmentID, user)
	if err != nil {
		return nil, err
	}

	s.repo.LogEvent(ctx, projectID, user.ID, "user.signup", "", "")
	return session, nil
}

func (s *Service) Login(ctx context.Context, projectID, environmentID string, req LoginRequest, ip, userAgent string) (*Session, error) {
	user, passwordHash, err := s.repo.GetUserByEmail(ctx, projectID, req.Email)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	if user.Disabled {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	session, err := s.createSession(ctx, projectID, environmentID, user)
	if err != nil {
		return nil, err
	}

	go s.repo.UpdateLastLogin(context.Background(), user.ID)
	s.repo.LogEvent(ctx, projectID, user.ID, "user.login", ip, userAgent)
	return session, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	tokenHash := hashToken(refreshToken)
	userID, projectID, environmentID, err := s.repo.GetSessionByRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	s.repo.DeleteSessionByRefreshToken(ctx, tokenHash)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	session, err := s.createSession(ctx, projectID, environmentID, user)
	if err != nil {
		return nil, err
	}

	s.repo.LogEvent(ctx, projectID, userID, "session.refresh", "", "")
	return session, nil
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	tokenHash := hashToken(accessToken)
	err := s.repo.DeleteSessionByAccessToken(ctx, tokenHash)
	return err
}

func (s *Service) GetUser(ctx context.Context, accessToken string) (*User, error) {
	claims, err := s.validateJWT(accessToken)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return s.repo.GetUserByID(ctx, claims.UserID)
}

func (s *Service) UpdateUser(ctx context.Context, accessToken string, req UpdateUserRequest) (*User, error) {
	claims, err := s.validateJWT(accessToken)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return s.repo.UpdateUser(ctx, claims.UserID, req)
}

func (s *Service) VerifyEmail(ctx context.Context, projectID, token string) error {
	tokenHash := hashToken(token)
	_, err := s.repo.VerifyEmail(ctx, projectID, tokenHash)
	if err != nil {
		return err
	}
	s.repo.LogEvent(ctx, projectID, "", "email.verified", "", "")
	return nil
}

func (s *Service) ForgotPassword(ctx context.Context, projectID, email string) error {
	user, _, err := s.repo.GetUserByEmail(ctx, projectID, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return err
	}

	s.repo.CreatePasswordReset(ctx, projectID, user.ID, tokenHash, time.Now().Add(resetTTL))
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, plainToken)
	go s.mailer.SendEmail(context.Background(), email, "Reset your password", resetEmailHTML(resetURL))
	s.repo.LogEvent(ctx, projectID, user.ID, "password.reset_requested", "", "")
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, projectID, token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return err
	}

	tokenHash := hashToken(token)
	return s.repo.ResetPassword(ctx, projectID, tokenHash, string(hash))
}

func (s *Service) ValidateToken(ctx context.Context, accessToken string) (*Claims, error) {
	return s.validateJWT(accessToken)
}

func (s *Service) createSession(ctx context.Context, projectID, environmentID string, user *User) (*Session, error) {
	accessToken, err := s.generateJWT(user, environmentID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRawToken()
	if err != nil {
		return nil, err
	}

	accessExp := time.Now().Add(accessTokenTTL)
	refreshExp := time.Now().Add(refreshTokenTTL)

	err = s.repo.CreateSession(ctx,
		projectID, environmentID, user.ID,
		hashToken(accessToken), hashToken(refreshToken),
		accessExp, refreshExp,
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
		User:         user,
	}, nil
}

func (s *Service) generateJWT(user *User, environmentID string) (string, error) {
	claims := Claims{
		UserID:        user.ID,
		ProjectID:     user.ProjectID,
		EnvironmentID: environmentID,
		Email:         user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "koolbase",
			Audience:  []string{"koolbase-client"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) validateJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	}, jwt.WithAudience("koolbase-client"), jwt.WithIssuer("koolbase"))
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func generateToken() (plain, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hashed = hashToken(plain)
	return plain, hashed, nil
}

func generateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
