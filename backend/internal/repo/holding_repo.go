package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
	"github.com/shopspring/decimal"
)

// HoldingRepo handles holding data access operations.
type HoldingRepo struct {
	pool *pgxpool.Pool
}

// NewHoldingRepo creates a new HoldingRepo.
func NewHoldingRepo(pool *pgxpool.Pool) *HoldingRepo {
	return &HoldingRepo{pool: pool}
}

// ListHoldingsByUser returns all active holdings for a user.
func (r *HoldingRepo) ListHoldingsByUser(ctx context.Context, userID int64) ([]model.Holding, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT holding_id, user_id, asset_code, asset_name, asset_type, category,
		        cost_price, position_ratio, quantity, created_at, updated_at
		 FROM holdings
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY holding_id ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query holdings: %w", err)
	}
	defer rows.Close()

	var holdings []model.Holding
	for rows.Next() {
		var h model.Holding
		var category sql.NullString
		if err := rows.Scan(
			&h.HoldingID, &h.UserID, &h.AssetCode, &h.AssetName, &h.AssetType, &category,
			&h.CostPrice, &h.PositionRatio, &h.Quantity,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		if category.Valid {
			s := category.String
			h.Category = &s
		}
		holdings = append(holdings, h)
	}
	return holdings, nil
}

// GetHoldingByID returns a single holding by ID. Returns nil if not found.
func (r *HoldingRepo) GetHoldingByID(ctx context.Context, holdingID int64) (*model.Holding, error) {
	var h model.Holding
	var category sql.NullString
	err := r.pool.QueryRow(ctx,
		`SELECT holding_id, user_id, asset_code, asset_name, asset_type, category,
		        cost_price, position_ratio, quantity, created_at, updated_at
		 FROM holdings
		 WHERE holding_id = $1 AND is_deleted = 0`,
		holdingID,
	).Scan(
		&h.HoldingID, &h.UserID, &h.AssetCode, &h.AssetName, &h.AssetType, &category,
		&h.CostPrice, &h.PositionRatio, &h.Quantity,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query holding by id: %w", err)
	}
	if category.Valid {
		s := category.String
		h.Category = &s
	}
	return &h, nil
}

// CountHoldingsByUser returns the number of active holdings for a user.
func (r *HoldingRepo) CountHoldingsByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM holdings WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count holdings: %w", err)
	}
	return count, nil
}

// CreateHolding inserts a new holding and returns it.
func (r *HoldingRepo) CreateHolding(
	ctx context.Context, userID int64,
	input *model.CreateHoldingInput, creator string,
) (*model.Holding, error) {
	var h model.Holding
	var category sql.NullString
	// Use sql.NullString symmetrically on write side to match the read side.
	// A nil input.Category maps to SQL NULL via Valid=false.
	categoryArg := sql.NullString{}
	if input.Category != nil {
		categoryArg = sql.NullString{String: *input.Category, Valid: true}
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO holdings
		 (user_id, asset_code, asset_name, asset_type, category,
		  cost_price, position_ratio, quantity, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		 RETURNING holding_id, user_id, asset_code, asset_name, asset_type, category,
		           cost_price, position_ratio, quantity, created_at, updated_at`,
		userID, input.AssetCode, input.AssetName, input.AssetType, categoryArg,
		input.CostPrice, input.PositionRatio, input.Quantity,
		creator,
	).Scan(
		&h.HoldingID, &h.UserID, &h.AssetCode, &h.AssetName, &h.AssetType, &category,
		&h.CostPrice, &h.PositionRatio, &h.Quantity,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert holding: %w", err)
	}
	if category.Valid {
		s := category.String
		h.Category = &s
	}
	return &h, nil
}

// UpdateHolding updates specified fields on a holding and returns the updated row.
func (r *HoldingRepo) UpdateHolding(
	ctx context.Context, holdingID int64,
	input *model.UpdateHoldingInput, modifier string,
) (*model.Holding, error) {
	var h model.Holding
	var category sql.NullString
	// Symmetric NullString on write side matches the read path. For sparse
	// updates, a nil Category means "leave unchanged" thanks to the
	// COALESCE($4, category) guard in the SQL below.
	categoryArg := sql.NullString{}
	if input.Category != nil {
		categoryArg = sql.NullString{String: *input.Category, Valid: true}
	}
	err := r.pool.QueryRow(ctx,
		`UPDATE holdings SET
			cost_price = COALESCE($1, cost_price),
			position_ratio = COALESCE($2, position_ratio),
			quantity = COALESCE($3, quantity),
			category = COALESCE($4, category),
			modifier = $5,
			updated_at = NOW()
		 WHERE holding_id = $6 AND is_deleted = 0
		 RETURNING holding_id, user_id, asset_code, asset_name, asset_type, category,
		           cost_price, position_ratio, quantity, created_at, updated_at`,
		input.CostPrice, input.PositionRatio, input.Quantity, categoryArg,
		modifier, holdingID,
	).Scan(
		&h.HoldingID, &h.UserID, &h.AssetCode, &h.AssetName, &h.AssetType, &category,
		&h.CostPrice, &h.PositionRatio, &h.Quantity,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update holding: %w", err)
	}
	if category.Valid {
		s := category.String
		h.Category = &s
	}
	return &h, nil
}

// UpdateHoldingCost updates cost_price and quantity directly (used after trade recalculation).
func (r *HoldingRepo) UpdateHoldingCost(
	ctx context.Context, holdingID int64,
	costPrice, quantity decimal.Decimal, modifier string,
) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE holdings SET cost_price = $1, quantity = $2, modifier = $3, updated_at = NOW()
		 WHERE holding_id = $4 AND is_deleted = 0`,
		costPrice, quantity, modifier, holdingID,
	)
	if err != nil {
		return fmt.Errorf("update holding cost: %w", err)
	}
	return nil
}

// SoftDeleteHolding marks a holding as deleted.
func (r *HoldingRepo) SoftDeleteHolding(ctx context.Context, holdingID int64, modifier string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE holdings SET is_deleted = 1, modifier = $1, updated_at = NOW()
		 WHERE holding_id = $2 AND is_deleted = 0`,
		modifier, holdingID,
	)
	if err != nil {
		return fmt.Errorf("soft delete holding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("holding not found")
	}
	return nil
}

// ListHoldingsByAssetType returns all active holdings of a given asset type across all users.
func (r *HoldingRepo) ListHoldingsByAssetType(ctx context.Context, assetType string) ([]model.Holding, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT holding_id, user_id, asset_code, asset_name, asset_type, category,
		        cost_price, position_ratio, quantity, created_at, updated_at
		 FROM holdings
		 WHERE asset_type = $1 AND is_deleted = 0
		 ORDER BY user_id, created_at DESC`,
		assetType,
	)
	if err != nil {
		return nil, fmt.Errorf("query holdings by asset type: %w", err)
	}
	defer rows.Close()

	var holdings []model.Holding
	for rows.Next() {
		var h model.Holding
		var category sql.NullString
		if err := rows.Scan(
			&h.HoldingID, &h.UserID, &h.AssetCode, &h.AssetName, &h.AssetType, &category,
			&h.CostPrice, &h.PositionRatio, &h.Quantity,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		if category.Valid {
			s := category.String
			h.Category = &s
		}
		holdings = append(holdings, h)
	}
	return holdings, nil
}
