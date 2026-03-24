CREATE TABLE IF NOT EXISTS org_usage_alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  resource TEXT NOT NULL,
  alerted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(org_id, resource)
);
CREATE INDEX IF NOT EXISTS idx_org_usage_alerts_org ON org_usage_alerts(org_id);
