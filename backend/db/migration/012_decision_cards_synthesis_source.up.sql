-- Record the provenance of each decision card: whether it was produced by a
-- live LLM call, a deterministic template fallback, or a mix of both, and
-- which provider layer served the request. Both columns are nullable so the
-- optimistic backfill below stays idempotent if re-run.
ALTER TABLE decision_cards
    ADD COLUMN IF NOT EXISTS synthesis_source VARCHAR(16)
        CHECK (synthesis_source IN ('llm', 'template', 'mixed')),
    ADD COLUMN IF NOT EXISTS provider_used    VARCHAR(32)
        CHECK (provider_used IN ('user', 'system_default', 'none'));

-- Optimistic backfill: historical deployments were assumed to be LLM-driven
-- via the configured user provider. Rows already written by newer code paths
-- will have non-null values and are left untouched.
UPDATE decision_cards
SET synthesis_source = 'llm',
    provider_used    = 'user'
WHERE synthesis_source IS NULL;

CREATE INDEX IF NOT EXISTS idx_decision_cards_synthesis_source
    ON decision_cards (synthesis_source)
    WHERE is_deleted = 0;

COMMENT ON COLUMN decision_cards.synthesis_source IS
    'Source of the synthesized content: llm (full AI), template (fallback), mixed (LLM text + template recommendation).';
COMMENT ON COLUMN decision_cards.provider_used IS
    'Which provider layer served this analysis: user, system_default, or none.';
