-- Store the current market price and quantity snapshot at analysis time.
-- current_price enables unrealized P&L computation on the frontend without
-- requiring a separate market-data fetch; quantity snapshots the holding size
-- so P&L stays accurate even if the user adjusts their position later.
ALTER TABLE decision_cards
    ADD COLUMN current_price DECIMAL(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN quantity      DECIMAL(20,6) NOT NULL DEFAULT 0;
