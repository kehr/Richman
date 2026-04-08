-- Drop the onboarding_skipped_at column
ALTER TABLE users DROP COLUMN IF EXISTS onboarding_skipped_at;
