-- Add onboarding_skipped_at field to track when users skip the onboarding flow
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS onboarding_skipped_at TIMESTAMPTZ NULL;
