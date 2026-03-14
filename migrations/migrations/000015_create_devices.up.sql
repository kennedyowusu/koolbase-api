CREATE TABLE IF NOT EXISTS devices (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
  device_id       TEXT NOT NULL,
  platform        TEXT NOT NULL,
  app_version     TEXT NOT NULL,
  last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (environment_id, device_id)
);

CREATE INDEX idx_devices_environment_id ON devices(environment_id);
CREATE INDEX idx_devices_last_seen_at ON devices(last_seen_at);
