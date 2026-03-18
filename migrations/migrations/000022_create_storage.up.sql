CREATE TABLE storage_buckets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    public     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);
CREATE INDEX idx_storage_buckets_project_id ON storage_buckets(project_id);

CREATE TABLE storage_objects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    bucket_id    UUID NOT NULL REFERENCES storage_buckets(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES project_users(id) ON DELETE SET NULL,
    path         TEXT NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    content_type TEXT,
    etag         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (bucket_id, path)
);
CREATE INDEX idx_storage_objects_project_id ON storage_objects(project_id);
CREATE INDEX idx_storage_objects_bucket_id ON storage_objects(bucket_id);
CREATE INDEX idx_storage_objects_user_id ON storage_objects(user_id);
CREATE INDEX idx_storage_objects_path ON storage_objects(bucket_id, path);
