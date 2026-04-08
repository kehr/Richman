-- Add structured recommendation fields to decision_cards
-- Keeps existing recommendation VARCHAR column for read-only backward compatibility.
ALTER TABLE decision_cards
    ADD COLUMN IF NOT EXISTS recommendation_json JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS action_level SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS target_position_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS badge_state VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS confidence_delta DECIMAL(6,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS prev_card_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS execution_fingerprint VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_dc_badge_state ON decision_cards (is_deleted, badge_state);
CREATE INDEX IF NOT EXISTS idx_dc_prev ON decision_cards (prev_card_id) WHERE is_deleted = 0;
