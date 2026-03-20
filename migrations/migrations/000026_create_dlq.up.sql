CREATE TABLE function_retry_queue (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL,
    function_name TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    collection    TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}',
    api_key       TEXT NOT NULL,
    attempt       INT NOT NULL DEFAULT 0,
    max_attempts  INT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_retry_queue_next ON function_retry_queue(next_retry_at)
    WHERE attempt < max_attempts;
CREATE INDEX idx_retry_queue_project ON function_retry_queue(project_id);

CREATE TABLE function_dead_letters (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL,
    function_name TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    collection    TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}',
    attempts      INT NOT NULL DEFAULT 5,
    last_error    TEXT,
    failed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dead_letters_project ON function_dead_letters(project_id);
CREATE INDEX idx_dead_letters_failed_at ON function_dead_letters(failed_at DESC);
