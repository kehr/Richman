-- Per-user LLM provider configuration. Holds the encrypted API key, the
-- provider type and model selection, and the fallback consent flags. Exactly
-- one active (non-deleted) configuration per user is enforced by the partial
-- unique index below.
CREATE TABLE IF NOT EXISTS llm_configs (
    config_id                              BIGSERIAL PRIMARY KEY,
    user_id                                BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    provider_type                          VARCHAR(32)  NOT NULL
        CHECK (provider_type IN ('claude', 'openai', 'openai_compatible')),
    base_url                               VARCHAR(512),
    api_key_cipher                         BYTEA        NOT NULL,
    api_key_nonce                          BYTEA        NOT NULL,
    api_key_hint                           VARCHAR(16)  NOT NULL,
    model                                  VARCHAR(128) NOT NULL,
    use_system_default_when_unconfigured   BOOLEAN      NOT NULL DEFAULT FALSE,
    fallback_to_system_default_on_failure  BOOLEAN      NOT NULL DEFAULT FALSE,
    health_status                          VARCHAR(16)  NOT NULL DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'failing', 'unknown')),
    last_probe_at                          TIMESTAMPTZ,
    last_probe_error                       TEXT,
    created_at                             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    creator                                VARCHAR(64)  NOT NULL DEFAULT 'system',
    modifier                               VARCHAR(64)  NOT NULL DEFAULT 'system',
    is_deleted                             SMALLINT     NOT NULL DEFAULT 0
);

-- Partial unique index: at most one active (is_deleted = 0) config per user.
-- Soft-deleted rows remain to preserve audit history without blocking new inserts.
CREATE UNIQUE INDEX IF NOT EXISTS uq_llm_configs_active_user
    ON llm_configs (user_id)
    WHERE is_deleted = 0;

-- Health status lookup for background probe scheduling.
CREATE INDEX IF NOT EXISTS idx_llm_configs_health_status
    ON llm_configs (health_status)
    WHERE is_deleted = 0;

COMMENT ON TABLE llm_configs IS
    'Per-user LLM provider configuration. Exactly one active config per user enforced by partial unique index.';
COMMENT ON COLUMN llm_configs.api_key_cipher IS
    'AES-256-GCM ciphertext of the plaintext api key. Master key from env LLM_CONFIG_MASTER_KEY.';
COMMENT ON COLUMN llm_configs.api_key_nonce IS
    'GCM nonce, 12 bytes, randomly generated on every save. Must be stored together with cipher.';
COMMENT ON COLUMN llm_configs.api_key_hint IS
    'Last 4 characters of the plaintext api key, prefixed with "..". Safe to log and display.';
