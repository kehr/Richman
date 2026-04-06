CREATE TABLE IF NOT EXISTS analysis_results (
    analysis_result_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    holding_id BIGINT NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    raw_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_ar_user ON analysis_results (is_deleted, user_id);

CREATE TABLE IF NOT EXISTS decision_cards (
    decision_card_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    holding_id BIGINT NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    asset_name VARCHAR(128) NOT NULL,
    asset_type VARCHAR(32) NOT NULL,
    cost_price DECIMAL(20,6) NOT NULL DEFAULT 0,
    position_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    trend_direction VARCHAR(16) NOT NULL,
    trend_summary TEXT NOT NULL DEFAULT '',
    position_direction VARCHAR(16) NOT NULL,
    position_summary TEXT NOT NULL DEFAULT '',
    catalyst_direction VARCHAR(16) NOT NULL,
    catalyst_summary TEXT NOT NULL DEFAULT '',
    confidence DECIMAL(5,2) NOT NULL DEFAULT 0,
    recommendation VARCHAR(32) NOT NULL,
    action_advice TEXT NOT NULL DEFAULT '',
    detailed_advice TEXT NOT NULL DEFAULT '',
    risk_warnings JSONB NOT NULL DEFAULT '[]',
    today_highlights TEXT NOT NULL DEFAULT '',
    weight_trend DECIMAL(5,4) NOT NULL DEFAULT 0,
    weight_position DECIMAL(5,4) NOT NULL DEFAULT 0,
    weight_catalyst DECIMAL(5,4) NOT NULL DEFAULT 0,
    analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_dc_user ON decision_cards (is_deleted, user_id);
CREATE INDEX IF NOT EXISTS idx_dc_holding ON decision_cards (is_deleted, holding_id);
