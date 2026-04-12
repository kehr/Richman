package quote

import (
	"context"
	"errors"
	"time"

	"github.com/richman/backend/internal/datasource"
	"go.uber.org/zap"
)

// Service provides asset quote data with in-memory caching.
type Service struct {
	provider QuoteProvider
	cache    *memoryCache
	logger   *zap.Logger
	cacheTTL time.Duration
}

// NewService creates a new quote service.
func NewService(provider QuoteProvider, logger *zap.Logger) *Service {
	return &Service{
		provider: provider,
		cache:    newMemoryCache(),
		logger:   logger,
		cacheTTL: 120 * time.Second,
	}
}

// GetQuote returns a QuoteDTO for the given asset. It returns a valid DTO
// with Source="unavailable" for unsupported asset types instead of an error.
func (s *Service) GetQuote(ctx context.Context, assetType, assetCode string) (*QuoteDTO, error) {
	key := assetType + ":" + assetCode

	if cached, ok := s.cache.Get(key); ok {
		s.logger.Debug("quote cache hit", zap.String("key", key))
		return cached, nil
	}

	snap, err := s.provider.FetchQuote(ctx, QuoteRequest{
		AssetType: assetType,
		AssetCode: assetCode,
		Days:      45,
	})
	if errors.Is(err, datasource.ErrUnsupportedAssetType) {
		dto := &QuoteDTO{
			AssetCode: assetCode,
			AssetType: assetType,
			Source:    "unavailable",
			FetchedAt: time.Now().UTC(),
			History:   []HistoryPoint{},
		}
		s.cache.Set(key, dto, s.cacheTTL)
		return dto, nil
	}
	if err != nil {
		s.logger.Warn("quote fetch failed",
			zap.String("assetType", assetType),
			zap.String("assetCode", assetCode),
			zap.Error(err),
		)
		return nil, err
	}

	dto := s.toDTO(snap, assetCode, assetType)
	s.cache.Set(key, dto, s.cacheTTL)
	return dto, nil
}

func (s *Service) toDTO(snap *QuoteSnapshot, assetCode, assetType string) *QuoteDTO {
	dto := &QuoteDTO{
		AssetCode: assetCode,
		AssetType: assetType,
		Source:    snap.Source,
		FetchedAt: snap.FetchedAt,
		History:   make([]HistoryPoint, 0, len(snap.History)),
	}

	for _, p := range snap.History {
		dto.History = append(dto.History, HistoryPoint{
			Date:   p.Date,
			Open:   p.Open,
			High:   p.High,
			Low:    p.Low,
			Close:  p.Close,
			Volume: p.Volume,
		})
	}

	if snap.Current.Close > 0 {
		changeAbs := snap.Current.Close - snap.Previous.Close
		var changePct float64
		if snap.Previous.Close > 0 {
			changePct = (changeAbs / snap.Previous.Close) * 100
		}
		dto.Current = &CurrentQuote{
			Price:     snap.Current.Close,
			Date:      snap.Current.Date,
			ChangeAbs: changeAbs,
			ChangePct: changePct,
		}
	}

	return dto
}
