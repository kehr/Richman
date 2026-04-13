-- 023_invite_system (down): remove invite system tables and login streak columns.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

ALTER TABLE rm_users DROP COLUMN IF EXISTS last_login_date;
ALTER TABLE rm_users DROP COLUMN IF EXISTS login_streak;

DROP TABLE IF EXISTS rm_invite_rewards;
DROP TABLE IF EXISTS rm_user_invite_codes;
