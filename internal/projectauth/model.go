package projectauth

import "time"

type User struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	Email        string     `json:"email"`
	FullName     *string    `json:"full_name,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Verified     bool       `json:"verified"`
	Disabled     bool       `json:"disabled"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	Metadata     *string    `json:"metadata,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         *User     `json:"user"`
}

type RegisterRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	FullName *string `json:"full_name,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	FullName  *string `json:"full_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Metadata  *string `json:"metadata,omitempty"`
}
