package portfolio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"github.com/shopspring/decimal"
)

const maxHoldingsPerUser = 5

// Service handles portfolio business logic.
type Service struct {
	holdingRepo *repo.HoldingRepo
	tradeRepo   *repo.TradeRepo
}

// NewService creates a new portfolio Service.
func NewService(holdingRepo *repo.HoldingRepo, tradeRepo *repo.TradeRepo) *Service {
	return &Service{
		holdingRepo: holdingRepo,
		tradeRepo:   tradeRepo,
	}
}

// ListHoldings returns all holdings for a user.
func (s *Service) ListHoldings(ctx context.Context, userID int64) ([]model.Holding, error) {
	holdings, err := s.holdingRepo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list holdings: %w", err)
	}
	return holdings, nil
}

// CreateHolding creates a new holding after checking the max limit.
func (s *Service) CreateHolding(
	ctx context.Context, userID int64,
	input *model.CreateHoldingInput, email string,
) (*model.Holding, error) {
	count, err := s.holdingRepo.CountHoldingsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count holdings: %w", err)
	}
	if count >= maxHoldingsPerUser {
		return nil, model.NewAppError(http.StatusBadRequest, "LIMIT_EXCEEDED",
			fmt.Sprintf("maximum %d holdings allowed per user", maxHoldingsPerUser))
	}

	holding, err := s.holdingRepo.CreateHolding(ctx, userID, input, email)
	if err != nil {
		return nil, fmt.Errorf("create holding: %w", err)
	}
	return holding, nil
}

// UpdateHolding updates a holding owned by the given user.
func (s *Service) UpdateHolding(
	ctx context.Context, userID, holdingID int64,
	input *model.UpdateHoldingInput, email string,
) (*model.Holding, error) {
	existing, err := s.holdingRepo.GetHoldingByID(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("get holding: %w", err)
	}
	if existing == nil || existing.UserID != userID {
		return nil, model.ErrNotFound
	}

	holding, err := s.holdingRepo.UpdateHolding(ctx, holdingID, input, email)
	if err != nil {
		return nil, fmt.Errorf("update holding: %w", err)
	}
	return holding, nil
}

// DeleteHolding soft-deletes a holding owned by the given user.
func (s *Service) DeleteHolding(ctx context.Context, userID, holdingID int64, email string) error {
	existing, err := s.holdingRepo.GetHoldingByID(ctx, holdingID)
	if err != nil {
		return fmt.Errorf("get holding: %w", err)
	}
	if existing == nil || existing.UserID != userID {
		return model.ErrNotFound
	}

	if err := s.holdingRepo.SoftDeleteHolding(ctx, holdingID, email); err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}
	return nil
}

// AddTrade adds a trade to a holding and recalculates the cost basis.
func (s *Service) AddTrade(
	ctx context.Context, userID, holdingID int64,
	input *model.CreateTradeInput, email string,
) (*model.Trade, error) {
	holding, err := s.holdingRepo.GetHoldingByID(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("get holding: %w", err)
	}
	if holding == nil || holding.UserID != userID {
		return nil, model.ErrNotFound
	}

	trade, err := s.tradeRepo.CreateTrade(ctx, holdingID, userID, input, email)
	if err != nil {
		return nil, fmt.Errorf("create trade: %w", err)
	}

	// Recalculate cost from all trades for this holding
	trades, err := s.tradeRepo.ListTradesByHolding(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("list trades for recalc: %w", err)
	}

	result := RecalculateCost(decimal.Zero, decimal.Zero, trades)
	err = s.holdingRepo.UpdateHoldingCost(
		ctx, holdingID, result.CostPrice, result.TotalQuantity, email,
	)
	if err != nil {
		return nil, fmt.Errorf("update holding cost: %w", err)
	}

	return trade, nil
}

// ListTrades returns all trades for a holding owned by the given user.
func (s *Service) ListTrades(ctx context.Context, userID, holdingID int64) ([]model.Trade, error) {
	holding, err := s.holdingRepo.GetHoldingByID(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("get holding: %w", err)
	}
	if holding == nil || holding.UserID != userID {
		return nil, model.ErrNotFound
	}

	trades, err := s.tradeRepo.ListTradesByHolding(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("list trades: %w", err)
	}
	return trades, nil
}
