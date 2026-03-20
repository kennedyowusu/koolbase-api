ALTER TABLE project_triggers DROP COLUMN function_id;
ALTER TABLE project_triggers ADD COLUMN function_name TEXT NOT NULL;
DROP INDEX IF EXISTS idx_project_triggers_function_id;
CREATE INDEX idx_project_triggers_function_name ON project_triggers(project_id, function_name);
ALTER TABLE project_triggers DROP CONSTRAINT IF EXISTS project_triggers_project_id_function_id_event_type_collection_key;
ALTER TABLE project_triggers ADD CONSTRAINT project_triggers_unique UNIQUE (project_id, function_name, event_type, collection);
