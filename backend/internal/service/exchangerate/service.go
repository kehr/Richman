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

const defaultTTL = time.Hour

// provider defines one exchange rate data source.
type provider struct {
	name string
	url  string
	// parse extracts USD and HKD rates (expressed as "1 CNY = X foreign")
	// from the raw API response body.
	parse func(body []byte) (usd, hkd float64, err error)
}

// providers is the ordered fallback chain. Sources are tried in sequence;
// the first successful response wins.
var providers = []provider{
	{
		name: "open.er-api.com",
		url:  "https://open.er-api.com/v6/latest/CNY",
		parse: func(body []byte) (float64, float64, error) {
			var r struct {
				Result string             `json:"result"`
				Rates  map[string]float64 `json:"rates"`
			}
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, 0, err
			}
			if r.Result != "success" {
				return 0, 0, fmt.Errorf("result: %s", r.Result)
			}
			return r.Rates["USD"], r.Rates["HKD"], nil
		},
	},
	{
		name: "exchangerate-api.com",
		url:  "https://api.exchangerate-api.com/v4/latest/CNY",
		parse: func(body []byte) (float64, float64, error) {
			var r struct {
				Rates map[string]float64 `json:"rates"`
			}
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, 0, err
			}
			return r.Rates["USD"], r.Rates["HKD"], nil
		},
	},
	{
		// jsdelivr mirrors @fawazahmed0/currency-api — keys are lowercase.
		name: "jsdelivr/fawazahmed0",
		url:  "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/cny.json",
		parse: func(body []byte) (float64, float64, error) {
			var r struct {
				CNY map[string]float64 `json:"cny"`
			}
			if err := json.Unmarshal(body, &r); err != nil {
				return 0, 0, err
			}
			return r.CNY["usd"], r.CNY["hkd"], nil
		},
	},
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

// GetRates returns cached rates when fresh; otherwise fetches from the provider
// chain. On total failure, returns the last known rates so callers degrade
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
		s.logger.Warn("all exchange rate providers failed, using cached rates", zap.Error(err))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cached
}

// refresh tries each provider in order and stops at the first success.
func (s *Service) refresh(ctx context.Context) error {
	for _, p := range providers {
		usd, hkd, err := s.fetchFrom(ctx, p)
		if err != nil {
			s.logger.Warn("exchange rate provider failed", zap.String("provider", p.name), zap.Error(err))
			continue
		}
		if usd <= 0 || hkd <= 0 {
			s.logger.Warn("exchange rate provider returned zero rates", zap.String("provider", p.name))
			continue
		}

		newValues := map[string]float64{
			"CNY": 1.0,
			"USD": usd,
			"HKD": hkd,
		}
		s.mu.Lock()
		s.cached = Rates{Values: newValues, UpdatedAt: time.Now().UTC()}
		s.fetchedAt = time.Now()
		s.mu.Unlock()

		s.logger.Info("exchange rates refreshed",
			zap.String("provider", p.name),
			zap.Float64("USD", usd),
			zap.Float64("HKD", hkd),
		)
		return nil
	}
	return fmt.Errorf("all %d providers exhausted", len(providers))
}

// fetchFrom fetches and parses rates from a single provider.
func (s *Service) fetchFrom(ctx context.Context, p provider) (usd, hkd float64, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, http.NoBody)
	if err != nil {
		return 0, 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("read body: %w", err)
	}

	return p.parse(body)
}
