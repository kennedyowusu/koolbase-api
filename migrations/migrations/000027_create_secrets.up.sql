CREATE TABLE project_secrets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);

CREATE INDEX idx_project_secrets_project_id ON project_secrets(project_id);

CREATE TRIGGER project_secrets_updated_at
    BEFORE UPDATE ON project_secrets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
