package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultTTL = time.Hour
	// open.er-api.com returns rates relative to the base currency (CNY).
	// Response: {"rates": {"USD": 0.146, "HKD": 1.144, ...}}
	// No API key required; free tier is sufficient for 1-hour TTL usage.
	erAPIURL = "https://open.er-api.com/v6/latest/CNY"
)

// erAPIResponse models the open.er-api.com /v6/latest response.
type erAPIResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// Rates holds exchange rates expressed as "1 CNY = X foreign currency".
// CNY is always present with value 1.0.
type Rates struct {
	Values    map[string]float64 `json:"rates"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

// Service fetches and caches forex exchange rates with a TTL-based lazy refresh.
type Service struct {
	httpClient *http.Client
	logger     *zap.Logger

	mu        sync.RWMutex
	cached    Rates
	fetchedAt time.Time
	ttl       time.Duration
}

// NewService constructs a Service with a 1-hour TTL.
// The initial cache contains only CNY=1.0 until the first GetRates call.
func NewService(logger *zap.Logger) *Service {
	return &Service{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		ttl:        defaultTTL,
		cached:     Rates{Values: map[string]float64{"CNY": 1.0}},
	}
}

// GetRates returns cached rates when fresh; otherwise fetches from the exchange
// rate API. On fetch failure, returns the last known rates so callers degrade
// gracefully.
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

// refresh fetches CNY-based rates from open.er-api.com and updates the cache.
// Rates are already expressed as "1 CNY = X foreign" so no inversion is needed.
func (s *Service) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, erAPIURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("build exchange rate request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch exchange rates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read exchange rate response: %w", err)
	}

	var parsed erAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse exchange rate response: %w", err)
	}
	if parsed.Result != "success" {
		return fmt.Errorf("exchange rate API returned non-success result: %s", parsed.Result)
	}

	newValues := map[string]float64{"CNY": 1.0}
	for _, currency := range []string{"USD", "HKD"} {
		if rate, ok := parsed.Rates[currency]; ok && rate > 0 {
			newValues[currency] = rate
		} else {
			s.logger.Warn("exchange rate missing or zero", zap.String("currency", currency))
		}
	}

	s.mu.Lock()
	s.cached = Rates{
		Values:    newValues,
		UpdatedAt: time.Now().UTC(),
	}
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	s.logger.Info("exchange rates refreshed",
		zap.Float64("USD", newValues["USD"]),
		zap.Float64("HKD", newValues["HKD"]),
	)
	return nil
}
