package exchangerate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/richman/backend/internal/datasource/yahoo"
	"go.uber.org/zap"
)

const (
	defaultTTL = time.Hour
	// Yahoo tickers: "XYZABC=X" means 1 XYZ = ? ABC
	// "USDCNY=X" -> 1 USD = ? CNY  -> invert to get 1 CNY = ? USD
	// "HKDCNY=X" -> 1 HKD = ? CNY  -> invert to get 1 CNY = ? HKD
	tickerUSD = "USDCNY=X"
	tickerHKD = "HKDCNY=X"
)

// Rates holds exchange rates expressed as "1 CNY = X foreign currency".
// CNY is always present with value 1.0.
type Rates struct {
	Values    map[string]float64 `json:"rates"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

// Service fetches and caches forex exchange rates with a TTL-based lazy refresh.
type Service struct {
	yahoo  *yahoo.Client
	logger *zap.Logger

	mu        sync.RWMutex
	cached    Rates
	fetchedAt time.Time
	ttl       time.Duration
}

// NewService constructs a Service with a 1-hour TTL.
// The initial cache contains only CNY=1.0 until the first GetRates call.
func NewService(yahooClient *yahoo.Client, logger *zap.Logger) *Service {
	return &Service{
		yahoo:  yahooClient,
		logger: logger,
		ttl:    defaultTTL,
		cached: Rates{Values: map[string]float64{"CNY": 1.0}},
	}
}

// GetRates returns cached rates when fresh; otherwise fetches from Yahoo Finance.
// On fetch failure, returns the last known rates so callers degrade gracefully.
func (s *Service) GetRates(ctx context.Context) Rates {
	s.mu.RLock()
	if !s.fetchedAt.IsZero() && time.Since(s.fetchedAt) < s.ttl {
		rates := s.cached
		s.mu.RUnlock()
		return rates
	}
	s.mu.RUnlock()

	if err := s.refresh(ctx); err != nil {
		s.logger.Warn("exchange rate refresh failed, using cached rates", zap.Error(err))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cached
}

// refresh fetches USDCNY=X and HKDCNY=X from Yahoo Finance, inverts the
// pair prices to produce "1 CNY = X foreign" rates, and updates the cache.
func (s *Service) refresh(ctx context.Context) error {
	newValues := map[string]float64{"CNY": 1.0}
	var firstErr error

	usdcny, err := s.yahoo.FetchForexRate(ctx, tickerUSD)
	if err != nil {
		s.logger.Warn("failed to fetch USD/CNY rate", zap.Error(err))
		firstErr = err
	} else if usdcny > 0 {
		newValues["USD"] = 1.0 / usdcny
	}

	hkdcny, err := s.yahoo.FetchForexRate(ctx, tickerHKD)
	if err != nil {
		s.logger.Warn("failed to fetch HKD/CNY rate", zap.Error(err))
		if firstErr == nil {
			firstErr = err
		}
	} else if hkdcny > 0 {
		newValues["HKD"] = 1.0 / hkdcny
	}

	s.mu.Lock()
	for k, v := range newValues {
		s.cached.Values[k] = v
	}
	if firstErr == nil {
		s.cached.UpdatedAt = time.Now().UTC()
		s.fetchedAt = time.Now()
	}
	s.mu.Unlock()

	if firstErr != nil {
		return fmt.Errorf("partial refresh failure: %w", firstErr)
	}
	return nil
}
