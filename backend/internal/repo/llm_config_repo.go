package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
)

// LLMConfigRepo handles persistence of per-user LLM provider configurations
// (llm_configs table). The one-active-row-per-user invariant is enforced by
// the partial unique index uq_llm_configs_active_user and by the Upsert
// transaction pattern below.
//
// Safety contract: LLMConfig.APIKeyCipher / APIKeyNonce are sensitive byte
// slices. This repo returns the model straight through without redaction
// because the live Resolver needs the cipher to decrypt. Any caller that
// logs or serializes the returned config MUST go through LLMConfig.Masked().
type LLMConfigRepo struct {
	pool *pgxpool.Pool
}

// NewLLMConfigRepo creates a new LLMConfigRepo backed by the given pgx pool.
func NewLLMConfigRepo(pool *pgxpool.Pool) *LLMConfigRepo {
	return &LLMConfigRepo{pool: pool}
}

// Compile-time assertions that the production repo types satisfy the narrow
// interfaces the resolver depends on. If a future refactor drops or reshapes
// one of the methods, these lines fail to compile and the misalignment is
// caught before the wire-up site. Both lines live in this file because it
// is already the only repo file that imports the llm package.
var (
	_ llm.LLMConfigRepo   = (*LLMConfigRepo)(nil)
	_ llm.UserConsentRepo = (*UserRepo)(nil)
)

// llmConfigColumns is the canonical column list for SELECT queries so every
// GetActive-style read pulls the same shape. Matches the 18 fields on
// model.LLMConfig in the same order as the struct definition.
const llmConfigColumns = `config_id, user_id, provider_type, base_url,
	api_key_cipher, api_key_nonce, api_key_hint, model,
	use_system_default_when_unconfigured, fallback_to_system_default_on_failure,
	health_status, last_probe_at, last_probe_error,
	created_at, updated_at, creator, modifier, is_deleted`

// scanLLMConfig reads the canonical llm_configs columns into a model.LLMConfig.
// Keep the Scan argument order in sync with llmConfigColumns.
func scanLLMConfig(row pgx.Row, c *model.LLMConfig) error {
	return row.Scan(
		&c.ConfigID, &c.UserID, &c.ProviderType, &c.BaseURL,
		&c.APIKeyCipher, &c.APIKeyNonce, &c.APIKeyHint, &c.Model,
		&c.UseSystemDefaultWhenUnconfigured, &c.FallbackToSystemDefaultOnFailure,
		&c.HealthStatus, &c.LastProbeAt, &c.LastProbeError,
		&c.CreatedAt, &c.UpdatedAt, &c.Creator, &c.Modifier, &c.IsDeleted,
	)
}

// GetActiveByUserID returns the single active (is_deleted = 0) config for a
// user. The partial unique index guarantees at most one row; this method
// returns llm.ErrConfigNotFound when no such row exists so callers can use
// errors.Is to distinguish "user never configured" from transport-level DB
// errors. pgx.ErrNoRows is normalized here to keep pgx details out of the
// Resolver and Service layers.
func (r *LLMConfigRepo) GetActiveByUserID(
	ctx context.Context, userID int64,
) (*model.LLMConfig, error) {
	var c model.LLMConfig
	row := r.pool.QueryRow(ctx,
		`SELECT `+llmConfigColumns+`
		 FROM llm_configs
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	)
	if err := scanLLMConfig(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, llm.ErrConfigNotFound
		}
		return nil, fmt.Errorf("query active llm config: %w", err)
	}
	return &c, nil
}

// Upsert creates or updates the single active config for a user inside a
// transaction. To keep the audit trail intact and avoid ON CONFLICT DO UPDATE
// complexity (many immutable columns, RETURNING interaction with partial
// index), we:
//
//  1. Soft-delete any existing active row (is_deleted = 1, modifier stamped).
//  2. INSERT the fresh row.
//
// Both steps run inside a single tx so a crash between them cannot leave the
// partial unique index doubly populated. The method mutates cfg in place with
// the returned config_id / created_at / updated_at so callers can surface the
// freshly-persisted state.
func (r *LLMConfigRepo) Upsert(ctx context.Context, cfg *model.LLMConfig) error {
	if cfg == nil {
		return errors.New("llm_config_repo: cfg is nil")
	}
	if cfg.UserID == 0 {
		return errors.New("llm_config_repo: user_id is required")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin llm config upsert tx: %w", err)
	}
	defer func() {
		// Rollback is a no-op after successful Commit; safe to defer unconditionally.
		_ = tx.Rollback(ctx)
	}()

	// Step 1: soft-delete any currently active row for this user so the
	// partial unique index is free before INSERT.
	if _, err = tx.Exec(ctx,
		`UPDATE llm_configs
		 SET is_deleted = 1,
		     modifier = $2,
		     updated_at = NOW()
		 WHERE user_id = $1 AND is_deleted = 0`,
		cfg.UserID, cfg.Modifier,
	); err != nil {
		return fmt.Errorf("soft-delete previous active llm config: %w", err)
	}

	// Step 2: insert the new active row. RETURNING pulls the db-assigned
	// config_id and the default timestamps back into the caller's struct.
	row := tx.QueryRow(ctx,
		`INSERT INTO llm_configs (
			user_id, provider_type, base_url,
			api_key_cipher, api_key_nonce, api_key_hint, model,
			use_system_default_when_unconfigured, fallback_to_system_default_on_failure,
			health_status, last_probe_at, last_probe_error,
			creator, modifier
		 ) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9,
			$10, $11, $12,
			$13, $14
		 )
		 RETURNING `+llmConfigColumns,
		cfg.UserID, cfg.ProviderType, cfg.BaseURL,
		cfg.APIKeyCipher, cfg.APIKeyNonce, cfg.APIKeyHint, cfg.Model,
		cfg.UseSystemDefaultWhenUnconfigured, cfg.FallbackToSystemDefaultOnFailure,
		cfg.HealthStatus, cfg.LastProbeAt, cfg.LastProbeError,
		cfg.Creator, cfg.Modifier,
	)
	if err = scanLLMConfig(row, cfg); err != nil {
		return fmt.Errorf("insert llm config: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit llm config upsert: %w", err)
	}
	return nil
}

// SoftDelete marks the user's currently active config as deleted. The next
// GetActiveByUserID call returns llm.ErrConfigNotFound. Idempotent — calling
// it on a user with no active row is a no-op (UPDATE affects 0 rows).
func (r *LLMConfigRepo) SoftDelete(ctx context.Context, userID int64, modifier string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE llm_configs
		 SET is_deleted = 1,
		     modifier = $2,
		     updated_at = NOW()
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID, modifier,
	)
	if err != nil {
		return fmt.Errorf("soft-delete llm config: %w", err)
	}
	return nil
}

// UpdateHealth updates only the health tracking columns on an existing
// config: health_status, last_probe_at (stamped with NOW()), and
// last_probe_error. The api_key_cipher / api_key_nonce / model / base_url
// fields are deliberately NOT touched so a live-path health update cannot
// accidentally corrupt or log the key material. Takes configID (not userID)
// to keep the update local to the exact row the Resolver just exercised,
// even if a concurrent admin operation rotated the active row.
func (r *LLMConfigRepo) UpdateHealth(
	ctx context.Context,
	configID int64,
	status model.LLMHealthStatus,
	lastError *string,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE llm_configs
		 SET health_status = $2,
		     last_probe_at = NOW(),
		     last_probe_error = $3,
		     updated_at = NOW()
		 WHERE config_id = $1 AND is_deleted = 0`,
		configID, status, lastError,
	)
	if err != nil {
		return fmt.Errorf("update llm config health: %w", err)
	}
	return nil
}
