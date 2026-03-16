package projectauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailTaken      = errors.New("email already in use")
	ErrInvalidToken    = errors.New("invalid or expired token")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserDisabled    = errors.New("user account is disabled")
	ErrUserNotVerified = errors.New("email not verified")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GetProjectByAPIKey resolves environment public_key → environmentID + projectID
func (r *Repository) GetProjectByAPIKey(ctx context.Context, apiKey string) (projectID, environmentID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT project_id, id FROM environments WHERE public_key = $1`,
		apiKey,
	).Scan(&projectID, &environmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errors.New("invalid api key")
		}
		return "", "", err
	}
	return projectID, environmentID, nil
}

func (r *Repository) CreateUser(ctx context.Context, projectID, email, passwordHash string, fullName *string) (*User, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM project_users WHERE project_id = $1 AND LOWER(email) = LOWER($2)`,
		projectID, email,
	).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrEmailTaken
	}

	var u User
	err = r.db.QueryRow(ctx,
		`INSERT INTO project_users (project_id, email, password_hash, full_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, project_id, email, full_name, avatar_url, verified, disabled, last_login_at, created_at, updated_at`,
		projectID, email, passwordHash, fullName,
	).Scan(&u.ID, &u.ProjectID, &u.Email, &u.FullName, &u.AvatarURL, &u.Verified, &u.Disabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (r *Repository) GetUserByEmail(ctx context.Context, projectID, email string) (*User, string, error) {
	var u User
	var passwordHash string
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, email, full_name, avatar_url, verified, disabled, last_login_at, created_at, updated_at, COALESCE(password_hash, '')
		 FROM project_users WHERE project_id = $1 AND LOWER(email) = LOWER($2)`,
		projectID, email,
	).Scan(&u.ID, &u.ProjectID, &u.Email, &u.FullName, &u.AvatarURL, &u.Verified, &u.Disabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", err
	}
	return &u, passwordHash, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id, project_id, email, full_name, avatar_url, verified, disabled, last_login_at, created_at, updated_at
		 FROM project_users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.ProjectID, &u.Email, &u.FullName, &u.AvatarURL, &u.Verified, &u.Disabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

func (r *Repository) UpdateLastLogin(ctx context.Context, userID string) {
	r.db.Exec(ctx, `UPDATE project_users SET last_login_at = NOW() WHERE id = $1`, userID)
}

func (r *Repository) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`UPDATE project_users SET
		 full_name  = COALESCE($2, full_name),
		 avatar_url = COALESCE($3, avatar_url),
		 updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, project_id, email, full_name, avatar_url, verified, disabled, last_login_at, created_at, updated_at`,
		userID, req.FullName, req.AvatarURL,
	).Scan(&u.ID, &u.ProjectID, &u.Email, &u.FullName, &u.AvatarURL, &u.Verified, &u.Disabled, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (r *Repository) CreateSession(ctx context.Context, projectID, environmentID, userID, accessHash, refreshHash string, accessExp, refreshExp interface{}) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO project_sessions (project_id, environment_id, user_id, access_token_hash, refresh_token_hash, access_expires_at, refresh_expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		projectID, environmentID, userID, accessHash, refreshHash, accessExp, refreshExp,
	)
	return err
}

func (r *Repository) GetSessionByAccessToken(ctx context.Context, tokenHash string) (userID, projectID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT user_id, project_id FROM project_sessions
		 WHERE access_token_hash = $1 AND access_expires_at > NOW()`,
		tokenHash,
	).Scan(&userID, &projectID)
	if err != nil {
		return "", "", ErrInvalidToken
	}
	return userID, projectID, nil
}

func (r *Repository) GetSessionByRefreshToken(ctx context.Context, tokenHash string) (userID, projectID, environmentID string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT user_id, project_id, environment_id FROM project_sessions
		 WHERE refresh_token_hash = $1 AND refresh_expires_at > NOW()`,
		tokenHash,
	).Scan(&userID, &projectID, &environmentID)
	if err != nil {
		return "", "", "", ErrInvalidToken
	}
	return userID, projectID, environmentID, nil
}

func (r *Repository) DeleteSessionByAccessToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM project_sessions WHERE access_token_hash = $1`, tokenHash)
	return err
}

func (r *Repository) CreateEmailVerification(ctx context.Context, projectID, userID, tokenHash string, expiresAt interface{}) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO project_email_verifications (project_id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (token_hash) DO NOTHING`,
		projectID, userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) VerifyEmail(ctx context.Context, projectID, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx,
		`UPDATE project_email_verifications SET used_at = NOW()
		 WHERE token_hash = $1 AND project_id = $2 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING user_id`,
		tokenHash, projectID,
	).Scan(&userID)
	if err != nil {
		return "", ErrInvalidToken
	}
	r.db.Exec(ctx, `UPDATE project_users SET verified = TRUE, updated_at = NOW() WHERE id = $1`, userID)
	return userID, nil
}

func (r *Repository) CreatePasswordReset(ctx context.Context, projectID, userID, tokenHash string, expiresAt interface{}) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO project_password_resets (project_id, user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		projectID, userID, tokenHash, expiresAt,
	)
	return err
}

func (r *Repository) ResetPassword(ctx context.Context, projectID, tokenHash, newPasswordHash string) error {
	var userID string
	err := r.db.QueryRow(ctx,
		`UPDATE project_password_resets SET used_at = NOW()
		 WHERE token_hash = $1 AND project_id = $2 AND used_at IS NULL AND expires_at > NOW()
		 RETURNING user_id`,
		tokenHash, projectID,
	).Scan(&userID)
	if err != nil {
		return ErrInvalidToken
	}
	_, err = r.db.Exec(ctx,
		`UPDATE project_users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newPasswordHash, userID,
	)
	return err
}

func (r *Repository) LogEvent(ctx context.Context, projectID, userID, eventType, ip, userAgent string) {
	go r.db.Exec(context.Background(),
		`INSERT INTO project_auth_events (project_id, user_id, event_type, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5)`,
		projectID, nullStr(userID), eventType, nullStr(ip), nullStr(userAgent),
	)
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
