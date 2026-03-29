CREATE TABLE IF NOT EXISTS cron_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    function_name   TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at     TIMESTAMPTZ,
    next_run_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cron_schedules_project_id ON cron_schedules(project_id);
CREATE INDEX IF NOT EXISTS idx_cron_schedules_next_run_at ON cron_schedules(next_run_at) WHERE enabled = TRUE;
