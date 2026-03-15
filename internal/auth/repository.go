package auth

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateUser(ctx context.Context, orgID, email, passwordHash, role string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	MarkEmailVerified(ctx context.Context, userID string) error
	CreateEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, id string) error
	CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, id string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time, ip, userAgent string) (*Session, error)
	GetSession(ctx context.Context, tokenHash string) (*Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
	UpdateUser(ctx context.Context, userID, email string) (*User, error)
	SetPendingEmail(ctx context.Context, userID, pendingEmail string) error
	ConfirmEmailChange(ctx context.Context, userID string) error
	DeleteAccount(ctx context.Context, userID string) error
	GetUserByEmailIncludeDeleted(ctx context.Context, email string) (*User, error)
	ReactivateAccount(ctx context.Context, userID, email string) error
	PurgeDeletedAccounts(ctx context.Context) error
	GetInviteOrgAndRole(ctx context.Context, tokenHash string, orgID *string, role *string) error
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, orgID, email, passwordHash, role string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (org_id, email, password_hash, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, email, role, verified, created_at`,
		orgID, email, passwordHash, role,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.Role, &u.Verified, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, email, password_hash, role, verified, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Verified, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, email, password_hash, role, verified, created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Verified, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET verified = true WHERE id = $1`, userID)
	return err
}

func (r *PostgresRepository) CreateEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *PostgresRepository) GetEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationToken, error) {
	var t EmailVerificationToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at FROM email_verification_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt)
	if err != nil {
		return nil, fmt.Errorf("get verification token: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) MarkEmailVerificationTokenUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verification_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, _ = r.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID)
	_, err := r.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

func (r *PostgresRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt)
	if err != nil {
		return nil, fmt.Errorf("get reset token: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) MarkPasswordResetTokenUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash)
	return err
}

func (r *PostgresRepository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time, ip, userAgent string) (*Session, error) {
	var s Session
	err := r.db.QueryRow(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, token_hash, expires_at, ip, user_agent, created_at`,
		userID, tokenHash, expiresAt, ip, userAgent,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.IP, &s.UserAgent, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) GetSession(ctx context.Context, tokenHash string) (*Session, error) {
	var s Session
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, ip, user_agent, created_at
		 FROM sessions WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash,
	).Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.IP, &s.UserAgent, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *PostgresRepository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, userID, email string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`UPDATE users SET email = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, org_id, email, role, verified, created_at`,
		userID, email,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.Role, &u.Verified, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return &u, nil
}

func (r *PostgresRepository) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	var hash string
	err := r.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password")
	}
	_, err = r.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, string(newHash))
	return err
}

func (r *PostgresRepository) SetPendingEmail(ctx context.Context, userID, pendingEmail string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET pending_email = $2 WHERE id = $1`,
		userID, pendingEmail,
	)
	return err
}

func (r *PostgresRepository) ConfirmEmailChange(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET email = pending_email, pending_email = NULL WHERE id = $1`,
		userID,
	)
	return err
}

func (r *PostgresRepository) DeleteAccount(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *PostgresRepository) GetUserByEmailIncludeDeleted(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx,
		`SELECT id, org_id, email, password_hash, role, verified, deleted_at FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.OrgID, &u.Email, &u.PasswordHash, &u.Role, &u.Verified, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) ReactivateAccount(ctx context.Context, userID, email string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET deleted_at = NULL, verified = FALSE WHERE id = $1`,
		userID,
	)
	return err
}

func (r *PostgresRepository) PurgeDeletedAccounts(ctx context.Context) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM users WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '30 days'`,
	)
	return err
}

func (r *PostgresRepository) GetInviteOrgAndRole(ctx context.Context, tokenHash string, orgID *string, role *string) error {
	err := r.db.QueryRow(ctx,
		`SELECT org_id, role FROM invitations WHERE token_hash = $1 AND accepted_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	).Scan(orgID, role)
	if err != nil {
		return err
	}
	// Mark as accepted now that signup is completing
	_, err = r.db.Exec(ctx, `UPDATE invitations SET accepted_at = NOW() WHERE token_hash = $1`, tokenHash)
	return err
}
