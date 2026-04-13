-- 021_rename_tables_rm_prefix: rename all 14 existing richman tables to use rm_ prefix.
-- PostgreSQL automatically updates FK references when a table is renamed, so no
-- constraint manipulation is needed. Sequences and index names retain their old
-- names (still functional) and will be updated by convention going forward.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

ALTER TABLE users                   RENAME TO rm_users;
ALTER TABLE plans                   RENAME TO rm_plans;
ALTER TABLE invite_codes            RENAME TO rm_invite_codes;
ALTER TABLE asset_catalog           RENAME TO rm_asset_catalog;
ALTER TABLE holdings                RENAME TO rm_holdings;
ALTER TABLE trades                  RENAME TO rm_trades;
ALTER TABLE analysis_results        RENAME TO rm_analysis_results;
ALTER TABLE decision_cards          RENAME TO rm_decision_cards;
ALTER TABLE notification_channels   RENAME TO rm_notification_channels;
ALTER TABLE notification_logs       RENAME TO rm_notification_logs;
ALTER TABLE analysis_tasks          RENAME TO rm_analysis_tasks;
ALTER TABLE llm_configs             RENAME TO rm_llm_configs;
ALTER TABLE user_schedule_settings  RENAME TO rm_user_schedule_settings;
ALTER TABLE holding_schedule_overrides RENAME TO rm_holding_schedule_overrides;
