CREATE TABLE IF NOT EXISTS notification_channels (
    notification_channel_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_type VARCHAR(32) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_nc_user ON notification_channels (is_deleted, user_id);

CREATE TABLE IF NOT EXISTS notification_logs (
    notification_log_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_type VARCHAR(32) NOT NULL,
    message_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_nl_user ON notification_logs (is_deleted, user_id);
