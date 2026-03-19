CREATE TABLE db_collections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    read_rule   TEXT NOT NULL DEFAULT 'authenticated' CHECK (read_rule IN ('public', 'authenticated', 'owner')),
    write_rule  TEXT NOT NULL DEFAULT 'authenticated' CHECK (write_rule IN ('authenticated', 'owner')),
    delete_rule TEXT NOT NULL DEFAULT 'owner' CHECK (delete_rule IN ('authenticated', 'owner')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name)
);
CREATE INDEX idx_db_collections_project_id ON db_collections(project_id);

CREATE TABLE db_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    collection_id UUID NOT NULL REFERENCES db_collections(id) ON DELETE CASCADE,
    created_by    UUID REFERENCES project_users(id) ON DELETE SET NULL,
    data          JSONB NOT NULL DEFAULT '{}',
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_db_records_project_id ON db_records(project_id);
CREATE INDEX idx_db_records_collection_id ON db_records(collection_id);
CREATE INDEX idx_db_records_created_by ON db_records(created_by);
CREATE INDEX idx_db_records_deleted_at ON db_records(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_db_records_data_gin ON db_records USING GIN(data);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER db_records_updated_at
    BEFORE UPDATE ON db_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
