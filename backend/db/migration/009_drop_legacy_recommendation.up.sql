-- Drop the legacy VARCHAR recommendation column. The structured
-- recommendation_json column (added in migration 006) is now the single
-- source of truth for recommendation data on decision_cards.
ALTER TABLE decision_cards DROP COLUMN IF EXISTS recommendation;
