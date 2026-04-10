ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_display_currency;
ALTER TABLE users DROP COLUMN IF EXISTS display_currency;
