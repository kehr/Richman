-- Down migration: resets sequences back to 1.
-- NOTE: existing row IDs are NOT reverted (reversing PK shifts with FK
-- propagation is unsafe in production). Run this only in local dev
-- environments where data can be discarded.

ALTER SEQUENCE users_user_id_seq RESTART WITH 1;
ALTER SEQUENCE holdings_holding_id_seq RESTART WITH 1;
ALTER SEQUENCE decision_cards_decision_card_id_seq RESTART WITH 1;
ALTER SEQUENCE analysis_results_analysis_result_id_seq RESTART WITH 1;
ALTER SEQUENCE notification_channels_notification_channel_id_seq RESTART WITH 1;
ALTER SEQUENCE notification_logs_notification_log_id_seq RESTART WITH 1;
ALTER SEQUENCE llm_configs_config_id_seq RESTART WITH 1;
ALTER SEQUENCE user_schedule_settings_id_seq RESTART WITH 1;
ALTER SEQUENCE holding_schedule_overrides_id_seq RESTART WITH 1;
ALTER SEQUENCE trades_trade_id_seq RESTART WITH 1;
