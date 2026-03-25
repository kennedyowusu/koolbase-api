CREATE TABLE project_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    password_hash TEXT,
    full_name     TEXT,
    avatar_url    TEXT,
    verified      BOOLEAN NOT NULL DEFAULT FALSE,
    disabled      BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, email)
);
CREATE INDEX idx_project_users_project_id ON project_users(project_id);
CREATE INDEX idx_project_users_email_lower ON project_users(project_id, lower(email));

CREATE TABLE project_sessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES project_users(id) ON DELETE CASCADE,
    access_token_hash  TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    access_expires_at  TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,
    ip                 TEXT,
    user_agent         TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_project_sessions_project_id ON project_sessions(project_id);
CREATE INDEX idx_project_sessions_user_id ON project_sessions(user_id);
CREATE INDEX idx_project_sessions_access_token ON project_sessions(access_token_hash);
CREATE INDEX idx_project_sessions_refresh_token ON project_sessions(refresh_token_hash);

CREATE TABLE project_user_identities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES project_users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    identity_data    JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, provider, provider_user_id)
);
CREATE INDEX idx_project_user_identities_user_id ON project_user_identities(user_id);

CREATE TABLE project_password_resets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES project_users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_project_password_resets_token ON project_password_resets(token_hash);

CREATE TABLE project_email_verifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES project_users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_project_email_verifications_token ON project_email_verifications(token_hash);

CREATE TABLE project_auth_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID REFERENCES project_users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    ip         TEXT,
    user_agent TEXT,
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_project_auth_events_project_id ON project_auth_events(project_id);
CREATE INDEX idx_project_auth_events_user_id ON project_auth_events(user_id);
CREATE INDEX idx_project_sessions_access_expires ON project_sessions(access_token_hash, access_expires_at);
CREATE INDEX idx_project_sessions_refresh_expires ON project_sessions(refresh_token_hash, refresh_expires_at);
CREATE INDEX idx_project_email_verifications_used ON project_email_verifications(token_hash, used_at);
CREATE INDEX idx_project_password_resets_used ON project_password_resets(token_hash, used_at);
CREATE INDEX idx_project_users_created_at ON project_users(project_id, created_at DESC);
