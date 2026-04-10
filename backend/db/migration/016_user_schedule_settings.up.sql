-- Store per-user schedule preferences for automated analysis triggers.
-- global_frequency controls the default cadence. Per-market overrides are
-- nullable (null = inherit from global). Window-level enabled/time/custom flags
-- allow fine-grained control over pre- vs. post-market analysis timing.
CREATE TABLE user_schedule_settings (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(user_id),

    -- global frequency
    global_frequency        TEXT NOT NULL DEFAULT 'daily',
    -- values: every_window | daily | every_2_days | every_3_days | weekly | custom
    global_frequency_days   INT,           -- only valid when global_frequency = 'custom', range 1-30

    -- a_share windows
    a_share_pre_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    a_share_pre_time        TIME    NOT NULL DEFAULT '08:30',
    a_share_pre_custom      BOOLEAN NOT NULL DEFAULT FALSE,
    a_share_post_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    a_share_post_time       TIME    NOT NULL DEFAULT '15:05',
    a_share_post_custom     BOOLEAN NOT NULL DEFAULT FALSE,
    a_share_frequency       TEXT,          -- null = follow global
    a_share_frequency_days  INT,

    -- us_stock / gold windows (times in Asia/Shanghai)
    us_pre_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    us_pre_time             TIME    NOT NULL DEFAULT '20:30',
    us_pre_custom           BOOLEAN NOT NULL DEFAULT FALSE,
    us_post_enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    us_post_time            TIME    NOT NULL DEFAULT '04:05',
    us_post_custom          BOOLEAN NOT NULL DEFAULT FALSE,
    us_frequency            TEXT,
    us_frequency_days       INT,

    -- audit
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted              BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (user_id)
);
