ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_currency VARCHAR(8) NOT NULL DEFAULT 'CNY';

ALTER TABLE users
    ADD CONSTRAINT chk_users_display_currency
    CHECK (display_currency IN ('CNY', 'USD', 'HKD'));
