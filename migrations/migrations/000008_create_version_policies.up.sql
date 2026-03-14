CREATE TABLE version_policies (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    platform       TEXT NOT NULL CHECK (platform IN ('ios', 'android', 'flutter')),
    min_version    TEXT NOT NULL,
    latest_version TEXT,
    force_update   BOOLEAN NOT NULL DEFAULT FALSE,
    update_message TEXT NOT NULL DEFAULT 'A new version is available. Please update to continue.',
    store_url      TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, platform)
);
CREATE INDEX idx_version_policies_environment_id ON version_policies (environment_id);
