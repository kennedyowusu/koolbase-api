ALTER TABLE project_triggers DROP COLUMN IF EXISTS function_name;
ALTER TABLE project_triggers ADD COLUMN function_id UUID REFERENCES project_functions(id) ON DELETE CASCADE;
