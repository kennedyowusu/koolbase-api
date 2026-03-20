-- =========================
-- FUNCTIONS (VERSIONED)
-- =========================
CREATE TABLE project_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    runtime TEXT NOT NULL DEFAULT 'deno',
    entry_file TEXT NOT NULL DEFAULT 'index.ts',
    code TEXT NOT NULL,
    version INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    timeout_ms INT NOT NULL DEFAULT 10000,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_deployed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name, version)
);

CREATE UNIQUE INDEX idx_project_functions_active
ON project_functions(project_id, name)
WHERE is_active = TRUE;

CREATE INDEX idx_project_functions_project_id
ON project_functions(project_id);

-- =========================
-- TRIGGERS (DB EVENTS)
-- =========================
CREATE TABLE project_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    function_id UUID NOT NULL REFERENCES project_functions(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (
        event_type IN ('db.record.created', 'db.record.updated', 'db.record.deleted')
    ),
    collection TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, function_id, event_type, collection)
);

CREATE INDEX idx_project_triggers_project_id ON project_triggers(project_id);
CREATE INDEX idx_project_triggers_function_id ON project_triggers(function_id);

-- =========================
-- FUNCTION EXECUTION LOGS
-- =========================
CREATE TABLE function_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID NOT NULL REFERENCES project_functions(id) ON DELETE CASCADE,
    project_id UUID NOT NULL,
    function_version INT NOT NULL,
    trigger_type TEXT NOT NULL CHECK (trigger_type IN ('http', 'db')),
    event_type TEXT,
    collection TEXT,
    status TEXT NOT NULL CHECK (status IN ('success', 'error', 'timeout')),
    duration_ms INT NOT NULL DEFAULT 0,
    output TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_function_logs_function_id ON function_logs(function_id);
CREATE INDEX idx_function_logs_project_id ON function_logs(project_id);
CREATE INDEX idx_function_logs_created_at ON function_logs(created_at DESC);
CREATE INDEX idx_function_logs_project_created ON function_logs(project_id, created_at DESC);

-- =========================
-- AUTO UPDATE TIMESTAMP
-- =========================
CREATE TRIGGER project_functions_updated_at
    BEFORE UPDATE ON project_functions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
