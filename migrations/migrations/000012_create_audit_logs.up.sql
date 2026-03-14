CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    actor_id      UUID REFERENCES users (id) ON DELETE SET NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    action        TEXT NOT NULL,
    diff          JSONB,
    ip            TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_logs_org_id     ON audit_logs (org_id);
CREATE INDEX idx_audit_logs_actor_id   ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_resource   ON audit_logs (resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
