ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;

ALTER TABLE users
    DROP COLUMN IF EXISTS categories,
    DROP COLUMN IF EXISTS risk_preference,
    DROP COLUMN IF EXISTS onboarding_completed_at,
    DROP COLUMN IF EXISTS total_capital_cny;
