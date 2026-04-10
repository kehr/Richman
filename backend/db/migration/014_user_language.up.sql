-- Add language preference column to users.
-- Supported values: 'zh' (Simplified Chinese), 'en' (English).
-- Default 'en' as the base language.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS language VARCHAR(8) NOT NULL DEFAULT 'en';

ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_language;
ALTER TABLE users
    ADD CONSTRAINT chk_users_language
    CHECK (language IN ('zh', 'en'));
