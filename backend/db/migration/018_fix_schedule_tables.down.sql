-- Reverse 018_fix_schedule_tables

DROP INDEX IF EXISTS idx_holding_schedule_overrides_holding;
DROP INDEX IF EXISTS uq_holding_schedule_overrides_active;
DROP INDEX IF EXISTS idx_user_schedule_settings_user;
DROP INDEX IF EXISTS uq_user_schedule_settings_active_user;

ALTER TABLE holding_schedule_overrides
    DROP COLUMN IF EXISTS modifier,
    DROP COLUMN IF EXISTS creator,
    ALTER COLUMN is_deleted DROP DEFAULT,
    ALTER COLUMN is_deleted TYPE BOOLEAN USING (is_deleted <> 0),
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ADD CONSTRAINT holding_schedule_overrides_user_id_holding_id_key UNIQUE (user_id, holding_id);

ALTER TABLE user_schedule_settings
    DROP COLUMN IF EXISTS modifier,
    DROP COLUMN IF EXISTS creator,
    ALTER COLUMN is_deleted DROP DEFAULT,
    ALTER COLUMN is_deleted TYPE BOOLEAN USING (is_deleted <> 0),
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ADD CONSTRAINT user_schedule_settings_user_id_key UNIQUE (user_id);
