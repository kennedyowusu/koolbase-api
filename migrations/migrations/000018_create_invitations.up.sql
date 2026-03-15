CREATE TABLE invitations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    token_hash  TEXT NOT NULL UNIQUE,
    invited_by  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    accepted_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invitations_org_id ON invitations (org_id);
CREATE INDEX idx_invitations_token_hash ON invitations (token_hash);
