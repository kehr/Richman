DROP INDEX IF EXISTS idx_dc_prev;
DROP INDEX IF EXISTS idx_dc_badge_state;

ALTER TABLE decision_cards
    DROP COLUMN IF EXISTS execution_fingerprint,
    DROP COLUMN IF EXISTS prev_card_id,
    DROP COLUMN IF EXISTS confidence_delta,
    DROP COLUMN IF EXISTS badge_state,
    DROP COLUMN IF EXISTS target_position_ratio,
    DROP COLUMN IF EXISTS action_level,
    DROP COLUMN IF EXISTS recommendation_json;
