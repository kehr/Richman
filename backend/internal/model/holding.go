package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Holding represents a user's position in a specific asset.
type Holding struct {
	HoldingID     int64           `json:"holdingId"`
	UserID        int64           `json:"userId"`
	AssetCode     string          `json:"assetCode"`
	AssetName     string          `json:"assetName"`
	AssetType     string          `json:"assetType"`
	Category      *string         `json:"category,omitempty"`
	CostPrice     decimal.Decimal `json:"costPrice"`
	PositionRatio decimal.Decimal `json:"positionRatio"`
	Quantity      decimal.Decimal `json:"quantity"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

// CreateHoldingInput contains the data required to create a new holding.
type CreateHoldingInput struct {
	AssetCode     string          `json:"assetCode" binding:"required"`
	AssetName     string          `json:"assetName" binding:"required"`
	AssetType     string          `json:"assetType" binding:"required"`
	Category      *string         `json:"category,omitempty"`
	CostPrice     decimal.Decimal `json:"costPrice"`
	PositionRatio decimal.Decimal `json:"positionRatio"`
	Quantity      decimal.Decimal `json:"quantity"`
}

// UpdateHoldingInput contains the data allowed to be updated on a holding.
type UpdateHoldingInput struct {
	CostPrice     *decimal.Decimal `json:"costPrice"`
	PositionRatio *decimal.Decimal `json:"positionRatio"`
	Quantity      *decimal.Decimal `json:"quantity"`
	Category      *string          `json:"category,omitempty"`
}
