-- Optional asset category for holdings, paired with onboarding category preselect.
ALTER TABLE holdings
    ADD COLUMN IF NOT EXISTS category VARCHAR(32) NULL;
