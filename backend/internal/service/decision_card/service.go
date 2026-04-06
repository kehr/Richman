package decisioncard

import (
	"context"
	"fmt"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
)

// Service provides query operations for decision cards.
type Service struct {
	cardRepo *repo.DecisionCardRepo
}

// NewService creates a new decision card Service.
func NewService(cardRepo *repo.DecisionCardRepo) *Service {
	return &Service{cardRepo: cardRepo}
}

// ListLatest returns the most recent decision card for each holding of a user.
func (s *Service) ListLatest(ctx context.Context, userID int64) ([]model.DecisionCard, error) {
	cards, err := s.cardRepo.ListLatestByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list latest cards: %w", err)
	}
	return cards, nil
}

// GetByID returns a single decision card, verifying it belongs to the given user.
func (s *Service) GetByID(ctx context.Context, userID, cardID int64) (*model.DecisionCard, error) {
	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}
	if card == nil || card.UserID != userID {
		return nil, model.ErrNotFound
	}
	return card, nil
}

// ListHistory returns recent decision cards for a user across all holdings.
func (s *Service) ListHistory(ctx context.Context, userID int64, limit int) ([]model.DecisionCard, error) {
	cards, err := s.cardRepo.ListHistory(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list card history: %w", err)
	}
	return cards, nil
}
