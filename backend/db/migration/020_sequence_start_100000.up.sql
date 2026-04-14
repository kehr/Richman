-- 020_sequence_start_100000: enforce 6-digit minimum for all business entity
-- primary keys. Existing rows are shifted by +99999 (id 1 -> 100000,
-- id 2 -> 100001, ...) so there are no gaps. Sequences are then set to
-- resume beyond the highest migrated id. Tables with no rows get their
-- sequences started at 100000.
--
-- Business entities covered: users, holdings, trades, decision_cards,
-- analysis_results, notification_channels, notification_logs, llm_configs,
-- user_schedule_settings, holding_schedule_overrides.
--
-- Excluded (reference / admin data): plans (already seeded at 100000),
-- invite_codes (admin-managed), asset_catalog (seed reference data),
-- analysis_tasks (UUID primary key, not sequential).
--
-- No explicit BEGIN/COMMIT here: the migration runner already wraps each file
-- in pool.Begin()/tx.Commit() (see backend/internal/migration/runner.go execFile).
-- Embedding a second BEGIN/COMMIT closes the runner's transaction prematurely
-- and the runner's later Commit() then fails on a closed tx.

-- Drop FK constraints so PK updates do not violate referential integrity.
-- Constraints are re-added at the end with identical definitions.
ALTER TABLE holding_schedule_overrides
    DROP CONSTRAINT holding_schedule_overrides_user_id_fkey,
    DROP CONSTRAINT holding_schedule_overrides_holding_id_fkey;
ALTER TABLE llm_configs
    DROP CONSTRAINT llm_configs_user_id_fkey;
ALTER TABLE user_schedule_settings
    DROP CONSTRAINT user_schedule_settings_user_id_fkey;

-- Step 1: shift users primary key and propagate to every table that holds
-- a logical user_id reference (no FK enforcement on most of these).
UPDATE users                     SET user_id = user_id + 99999;
UPDATE holdings                  SET user_id = user_id + 99999;
UPDATE analysis_results          SET user_id = user_id + 99999;
UPDATE analysis_tasks            SET user_id = user_id + 99999;
UPDATE decision_cards            SET user_id = user_id + 99999;
UPDATE notification_channels     SET user_id = user_id + 99999;
UPDATE notification_logs         SET user_id = user_id + 99999;
UPDATE trades                    SET user_id = user_id + 99999;
UPDATE llm_configs               SET user_id = user_id + 99999;
UPDATE user_schedule_settings    SET user_id = user_id + 99999;
UPDATE holding_schedule_overrides SET user_id = user_id + 99999;

-- Step 2: shift holdings primary key and propagate to referencing tables.
UPDATE holdings                  SET holding_id = holding_id + 99999;
UPDATE analysis_results          SET holding_id = holding_id + 99999;
UPDATE decision_cards            SET holding_id = holding_id + 99999;
UPDATE trades                    SET holding_id = holding_id + 99999;
UPDATE holding_schedule_overrides SET holding_id = holding_id + 99999;

-- Step 3: shift remaining business entity primary keys.
UPDATE decision_cards            SET decision_card_id       = decision_card_id       + 99999;
UPDATE analysis_results          SET analysis_result_id     = analysis_result_id     + 99999;
UPDATE notification_channels     SET notification_channel_id = notification_channel_id + 99999;
UPDATE notification_logs         SET notification_log_id    = notification_log_id    + 99999;
UPDATE llm_configs               SET config_id              = config_id              + 99999;
UPDATE user_schedule_settings    SET id                     = id                     + 99999;
UPDATE holding_schedule_overrides SET id                    = id                     + 99999;
UPDATE trades                    SET trade_id               = trade_id               + 99999;

-- Step 4: restore FK constraints with their original ON DELETE semantics.
ALTER TABLE holding_schedule_overrides
    ADD CONSTRAINT holding_schedule_overrides_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(user_id),
    ADD CONSTRAINT holding_schedule_overrides_holding_id_fkey
        FOREIGN KEY (holding_id) REFERENCES holdings(holding_id);
ALTER TABLE llm_configs
    ADD CONSTRAINT llm_configs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE;
ALTER TABLE user_schedule_settings
    ADD CONSTRAINT user_schedule_settings_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(user_id);

-- Step 5: reset sequences.
-- For populated tables: setval(seq, max_id, true) so next value = max_id + 1.
-- For empty tables:     setval(seq, 100000, false) so first inserted row = 100000.
SELECT setval('users_user_id_seq',
    GREATEST(COALESCE((SELECT MAX(user_id) FROM users), 99999), 99999), true);
SELECT setval('holdings_holding_id_seq',
    GREATEST(COALESCE((SELECT MAX(holding_id) FROM holdings), 99999), 99999), true);
SELECT setval('decision_cards_decision_card_id_seq',
    GREATEST(COALESCE((SELECT MAX(decision_card_id) FROM decision_cards), 99999), 99999), true);
SELECT setval('analysis_results_analysis_result_id_seq',
    GREATEST(COALESCE((SELECT MAX(analysis_result_id) FROM analysis_results), 99999), 99999), true);
SELECT setval('notification_channels_notification_channel_id_seq',
    GREATEST(COALESCE((SELECT MAX(notification_channel_id) FROM notification_channels), 99999), 99999), true);
SELECT setval('notification_logs_notification_log_id_seq',
    GREATEST(COALESCE((SELECT MAX(notification_log_id) FROM notification_logs), 99999), 99999), true);
SELECT setval('llm_configs_config_id_seq',
    GREATEST(COALESCE((SELECT MAX(config_id) FROM llm_configs), 99999), 99999), true);
SELECT setval('user_schedule_settings_id_seq',
    GREATEST(COALESCE((SELECT MAX(id) FROM user_schedule_settings), 99999), 99999), true);
SELECT setval('holding_schedule_overrides_id_seq',
    GREATEST(COALESCE((SELECT MAX(id) FROM holding_schedule_overrides), 99999), 99999), true);
SELECT setval('trades_trade_id_seq',
    GREATEST(COALESCE((SELECT MAX(trade_id) FROM trades), 99999), 99999), true);
