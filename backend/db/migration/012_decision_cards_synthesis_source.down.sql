DROP INDEX IF EXISTS idx_decision_cards_synthesis_source;
ALTER TABLE decision_cards
    DROP COLUMN IF EXISTS synthesis_source,
    DROP COLUMN IF EXISTS provider_used;
