-- Fix schema issues in user_schedule_settings and holding_schedule_overrides.
-- Changes: (1) is_deleted BOOLEAN -> SMALLINT per project standard
--          (2) drop inline UNIQUE constraints, add partial unique indexes WHERE is_deleted = 0
--          (3) add missing audit columns creator and modifier
--          (4) add query indexes for common lookup patterns
-- NOTE: ON DELETE CASCADE cannot be added to existing FK constraints without
--       a full table rebuild. Existing FKs without cascade are acceptable for MVP
--       and will be addressed in a future migration if needed.

-- user_schedule_settings --

ALTER TABLE user_schedule_settings ALTER COLUMN is_deleted DROP DEFAULT;
ALTER TABLE user_schedule_settings ALTER COLUMN is_deleted TYPE SMALLINT USING (is_deleted::int)::smallint;
ALTER TABLE user_schedule_settings ALTER COLUMN is_deleted SET DEFAULT 0;
ALTER TABLE user_schedule_settings DROP CONSTRAINT IF EXISTS user_schedule_settings_user_id_key;
ALTER TABLE user_schedule_settings ADD COLUMN IF NOT EXISTS creator  VARCHAR(64) NOT NULL DEFAULT 'system';
ALTER TABLE user_schedule_settings ADD COLUMN IF NOT EXISTS modifier VARCHAR(64) NOT NULL DEFAULT 'system';

-- Partial unique index: at most one active (is_deleted = 0) schedule setting per user.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_schedule_settings_active_user
    ON user_schedule_settings (user_id)
    WHERE is_deleted = 0;

-- Lookup index for active record queries.
CREATE INDEX IF NOT EXISTS idx_user_schedule_settings_user
    ON user_schedule_settings (is_deleted, user_id);

-- holding_schedule_overrides --

ALTER TABLE holding_schedule_overrides ALTER COLUMN is_deleted DROP DEFAULT;
ALTER TABLE holding_schedule_overrides ALTER COLUMN is_deleted TYPE SMALLINT USING (is_deleted::int)::smallint;
ALTER TABLE holding_schedule_overrides ALTER COLUMN is_deleted SET DEFAULT 0;
ALTER TABLE holding_schedule_overrides DROP CONSTRAINT IF EXISTS holding_schedule_overrides_user_id_holding_id_key;
ALTER TABLE holding_schedule_overrides ADD COLUMN IF NOT EXISTS creator  VARCHAR(64) NOT NULL DEFAULT 'system';
ALTER TABLE holding_schedule_overrides ADD COLUMN IF NOT EXISTS modifier VARCHAR(64) NOT NULL DEFAULT 'system';

-- Partial unique index: one active override per (user, holding) pair.
CREATE UNIQUE INDEX IF NOT EXISTS uq_holding_schedule_overrides_active
    ON holding_schedule_overrides (user_id, holding_id)
    WHERE is_deleted = 0;

-- Lookup index for fetching overrides by holding.
CREATE INDEX IF NOT EXISTS idx_holding_schedule_overrides_holding
    ON holding_schedule_overrides (is_deleted, holding_id);
