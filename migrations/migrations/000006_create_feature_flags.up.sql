CREATE TABLE feature_flags (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id     UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    key                TEXT NOT NULL,
    enabled            BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_percentage INT NOT NULL DEFAULT 100 CHECK (rollout_percentage BETWEEN 0 AND 100),
    kill_switch        BOOLEAN NOT NULL DEFAULT FALSE,
    description        TEXT NOT NULL DEFAULT '',
    created_by         UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, key)
);
CREATE INDEX idx_feature_flags_environment_id ON feature_flags (environment_id);

CREATE TABLE flag_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    flag_id    UUID NOT NULL REFERENCES feature_flags (id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('rollout', 'targeting', 'schedule')),
    config     JSONB NOT NULL DEFAULT '{}',
    priority   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_flag_rules_flag_id ON flag_rules (flag_id);
