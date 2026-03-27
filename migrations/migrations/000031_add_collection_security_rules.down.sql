UPDATE db_collections SET read_rule = 'authenticated' WHERE read_rule IN ('scoped', 'conditional');
UPDATE db_collections SET write_rule = 'authenticated' WHERE write_rule IN ('scoped', 'conditional');
UPDATE db_collections SET delete_rule = 'owner' WHERE delete_rule IN ('scoped', 'conditional');

ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_read_rule_check;
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_write_rule_check;
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_delete_rule_check;
ALTER TABLE db_collections DROP CONSTRAINT IF EXISTS db_collections_rule_mode_check;
ALTER TABLE db_collections DROP COLUMN IF EXISTS owner_field;
ALTER TABLE db_collections DROP COLUMN IF EXISTS rule_mode;
ALTER TABLE db_collections DROP COLUMN IF EXISTS rule_conditions;

ALTER TABLE db_collections ADD CONSTRAINT db_collections_read_rule_check
  CHECK (read_rule = ANY (ARRAY['public'::text, 'authenticated'::text, 'owner'::text]));
ALTER TABLE db_collections ADD CONSTRAINT db_collections_write_rule_check
  CHECK (write_rule = ANY (ARRAY['authenticated'::text, 'owner'::text]));
ALTER TABLE db_collections ADD CONSTRAINT db_collections_delete_rule_check
  CHECK (delete_rule = ANY (ARRAY['authenticated'::text, 'owner'::text]));
