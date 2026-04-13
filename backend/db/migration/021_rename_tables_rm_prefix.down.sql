-- 021_rename_tables_rm_prefix (down): reverse the rm_ prefix rename.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

ALTER TABLE rm_holding_schedule_overrides RENAME TO holding_schedule_overrides;
ALTER TABLE rm_user_schedule_settings  RENAME TO user_schedule_settings;
ALTER TABLE rm_llm_configs             RENAME TO llm_configs;
ALTER TABLE rm_analysis_tasks          RENAME TO analysis_tasks;
ALTER TABLE rm_notification_logs       RENAME TO notification_logs;
ALTER TABLE rm_notification_channels   RENAME TO notification_channels;
ALTER TABLE rm_decision_cards          RENAME TO decision_cards;
ALTER TABLE rm_analysis_results        RENAME TO analysis_results;
ALTER TABLE rm_trades                  RENAME TO trades;
ALTER TABLE rm_holdings                RENAME TO holdings;
ALTER TABLE rm_asset_catalog           RENAME TO asset_catalog;
ALTER TABLE rm_invite_codes            RENAME TO invite_codes;
ALTER TABLE rm_plans                   RENAME TO plans;
ALTER TABLE rm_users                   RENAME TO users;
