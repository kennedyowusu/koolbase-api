ALTER TABLE project_sessions ADD COLUMN environment_id UUID REFERENCES environments(id) ON DELETE CASCADE;
CREATE INDEX idx_project_sessions_environment_id ON project_sessions(environment_id);
