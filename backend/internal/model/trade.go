package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade represents a single buy or sell transaction for a holding.
type Trade struct {
	TradeID   int64           `json:"tradeId"`
	HoldingID int64           `json:"holdingId"`
	UserID    int64           `json:"userId"`
	Direction string          `json:"direction"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	TradedAt  time.Time       `json:"tradedAt"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// CreateTradeInput contains the data required to create a new trade.
type CreateTradeInput struct {
	Direction string          `json:"direction" binding:"required,oneof=buy sell"`
	Price     decimal.Decimal `json:"price" binding:"required"`
	Quantity  decimal.Decimal `json:"quantity" binding:"required"`
	TradedAt  time.Time       `json:"tradedAt" binding:"required"`
}
