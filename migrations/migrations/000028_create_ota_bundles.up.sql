CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS ota_bundles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  channel TEXT NOT NULL DEFAULT 'production',
  version INTEGER NOT NULL,
  checksum TEXT NOT NULL,
  storage_path TEXT NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  mandatory BOOLEAN NOT NULL DEFAULT FALSE,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  release_notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, channel, version)
);
CREATE INDEX IF NOT EXISTS idx_ota_bundles_project_channel ON ota_bundles(project_id, channel);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ota_active_unique ON ota_bundles(project_id, channel) WHERE active = TRUE;
