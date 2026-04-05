-- Asset catalog (pre-populated seed data)
CREATE TABLE IF NOT EXISTS asset_catalog (
    asset_catalog_id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) NOT NULL,
    name VARCHAR(128) NOT NULL,
    name_en VARCHAR(128) NOT NULL DEFAULT '',
    asset_type VARCHAR(32) NOT NULL,
    exchange VARCHAR(32) NOT NULL DEFAULT '',
    data_source VARCHAR(32) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_asset_catalog_code ON asset_catalog (code) WHERE is_deleted = 0;
CREATE INDEX IF NOT EXISTS idx_ac_type ON asset_catalog (is_deleted, asset_type);

-- Holdings
CREATE TABLE IF NOT EXISTS holdings (
    holding_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    asset_name VARCHAR(128) NOT NULL,
    asset_type VARCHAR(32) NOT NULL,
    cost_price DECIMAL(20,6) NOT NULL DEFAULT 0,
    position_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    quantity DECIMAL(20,6) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_hld_user ON holdings (is_deleted, user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_hld_user_asset ON holdings (user_id, asset_code) WHERE is_deleted = 0;

-- Trades
CREATE TABLE IF NOT EXISTS trades (
    trade_id BIGSERIAL PRIMARY KEY,
    holding_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    direction VARCHAR(8) NOT NULL,
    price DECIMAL(20,6) NOT NULL,
    quantity DECIMAL(20,6) NOT NULL,
    traded_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_trade_holding ON trades (is_deleted, holding_id);
