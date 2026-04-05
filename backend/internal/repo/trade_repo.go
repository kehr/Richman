package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// TradeRepo handles trade data access operations.
type TradeRepo struct {
	pool *pgxpool.Pool
}

// NewTradeRepo creates a new TradeRepo.
func NewTradeRepo(pool *pgxpool.Pool) *TradeRepo {
	return &TradeRepo{pool: pool}
}

// ListTradesByHolding returns all trades for a given holding.
func (r *TradeRepo) ListTradesByHolding(ctx context.Context, holdingID int64) ([]model.Trade, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT trade_id, holding_id, user_id, direction, price, quantity, traded_at, created_at, updated_at
		 FROM trades
		 WHERE holding_id = $1 AND is_deleted = 0
		 ORDER BY traded_at ASC`,
		holdingID,
	)
	if err != nil {
		return nil, fmt.Errorf("query trades: %w", err)
	}
	defer rows.Close()

	var trades []model.Trade
	for rows.Next() {
		var t model.Trade
		if err := rows.Scan(
			&t.TradeID, &t.HoldingID, &t.UserID, &t.Direction,
			&t.Price, &t.Quantity, &t.TradedAt,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan trade: %w", err)
		}
		trades = append(trades, t)
	}
	return trades, nil
}

// CreateTrade inserts a new trade and returns it.
func (r *TradeRepo) CreateTrade(
	ctx context.Context, holdingID, userID int64,
	input *model.CreateTradeInput, creator string,
) (*model.Trade, error) {
	var t model.Trade
	err := r.pool.QueryRow(ctx,
		`INSERT INTO trades (holding_id, user_id, direction, price, quantity, traded_at, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		 RETURNING trade_id, holding_id, user_id, direction, price, quantity, traded_at, created_at, updated_at`,
		holdingID, userID, input.Direction, input.Price, input.Quantity, input.TradedAt, creator,
	).Scan(
		&t.TradeID, &t.HoldingID, &t.UserID, &t.Direction,
		&t.Price, &t.Quantity, &t.TradedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert trade: %w", err)
	}
	return &t, nil
}
