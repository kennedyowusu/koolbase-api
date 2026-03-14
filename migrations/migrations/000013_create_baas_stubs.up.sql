CREATE TABLE kb_databases (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'provisioning' CHECK (status IN ('provisioning', 'active', 'paused', 'error')),
    region     TEXT NOT NULL DEFAULT 'us-east-1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE kb_storage_buckets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    public     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);

CREATE TABLE kb_functions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    runtime     TEXT NOT NULL DEFAULT 'deno',
    source_hash TEXT,
    status      TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'deployed', 'error')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);

CREATE TABLE kb_auth_providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    config      JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, provider)
);
