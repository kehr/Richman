-- Per-holding schedule overrides: allow users to set different analysis
-- frequency and window for individual holdings, overriding the market-level
-- defaults from user_schedule_settings. Null fields mean "follow market default".
CREATE TABLE holding_schedule_overrides (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(user_id),
    holding_id      BIGINT NOT NULL REFERENCES holdings(holding_id),
    frequency       TEXT,    -- null = follow market, same values as global_frequency
    frequency_days  INT,
    "window"        TEXT,    -- null = follow market, values: pre | post | both
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (user_id, holding_id)
);
