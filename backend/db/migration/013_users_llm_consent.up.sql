-- Consent flag that lets the Resolver fall through to the system default LLM
-- when a user has no personal llm_configs row. Stored on users because the
-- consent must exist independently of whether the user ever configures their
-- own provider. Default FALSE keeps unconfigured users in the template path
-- until they explicitly opt in via onboarding.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS use_system_default_llm_consent BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.use_system_default_llm_consent IS
    'When TRUE, Resolver may fall through to the system default LLM if the user has no personal llm_configs row. Set via onboarding consent checkbox.';
