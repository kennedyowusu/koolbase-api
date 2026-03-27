-- Drop existing check constraints on rules
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_read_rule_check;
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_write_rule_check;
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_delete_rule_check;

-- Add new columns
ALTER TABLE db_collections ADD COLUMN IF NOT EXISTS owner_field TEXT;
ALTER TABLE db_collections ADD COLUMN IF NOT EXISTS rule_mode TEXT NOT NULL DEFAULT 'all';
ALTER TABLE db_collections ADD COLUMN IF NOT EXISTS rule_conditions JSONB NOT NULL DEFAULT '[]';

-- Add updated check constraints that include new rule types
ALTER TABLE db_collections ADD CONSTRAINT db_collections_read_rule_check
  CHECK (read_rule = ANY (ARRAY['public', 'authenticated', 'owner', 'scoped', 'conditional']));

ALTER TABLE db_collections ADD CONSTRAINT db_collections_write_rule_check
  CHECK (write_rule = ANY (ARRAY['authenticated', 'owner', 'scoped', 'conditional']));

ALTER TABLE db_collections ADD CONSTRAINT db_collections_delete_rule_check
  CHECK (delete_rule = ANY (ARRAY['authenticated', 'owner', 'scoped', 'conditional']));

ALTER TABLE db_collections ADD CONSTRAINT db_collections_rule_mode_check
  CHECK (rule_mode = ANY (ARRAY['all', 'any']));
