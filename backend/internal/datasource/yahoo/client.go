package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/richman/backend/internal/datasource"
	"go.uber.org/zap"
)

const chartBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart"

// Client fetches market data from Yahoo Finance public APIs.
type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// New creates a new Yahoo Finance client.
func New(logger *zap.Logger) *Client {
	return &Client{
		httpClient: datasource.NewHTTPClient(),
		logger:     logger,
	}
}

// chartResponse models the Yahoo Finance v8 chart API JSON structure.
type chartResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []jsonFloat `json:"open"`
					High   []jsonFloat `json:"high"`
					Low    []jsonFloat `json:"low"`
					Close  []jsonFloat `json:"close"`
					Volume []jsonFloat `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// jsonFloat handles null values in Yahoo Finance JSON number arrays.
type jsonFloat struct {
	Value float64
	Valid bool
}

func (f *jsonFloat) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		f.Valid = false
		return nil
	}
	f.Valid = true
	return json.Unmarshal(data, &f.Value)
}

// FetchPriceHistory fetches daily OHLCV for US stocks, ETFs, or commodities.
func (c *Client) FetchPriceHistory(ctx context.Context, symbol string, days int) ([]datasource.PriceData, error) {
	if days <= 0 {
		days = 30
	}

	url := fmt.Sprintf("%s/%s?range=%dd&interval=1d", chartBaseURL, symbol, days)
	c.logger.Info("fetching price history from yahoo", zap.String("symbol", symbol), zap.String("url", url))

	body, err := datasource.FetchWithRetry(ctx, c.httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("fetch yahoo chart for %s: %w", symbol, err)
	}

	return ParseChartResponse(body)
}

// ParseChartResponse parses Yahoo Finance chart JSON into PriceData slices.
// Exported for testing.
func ParseChartResponse(data []byte) ([]datasource.PriceData, error) {
	var resp chartResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("%w: %v", datasource.ErrInvalidResponse, err)
	}

	if resp.Chart.Error != nil {
		return nil, fmt.Errorf("%w: %s - %s",
			datasource.ErrInvalidResponse, resp.Chart.Error.Code, resp.Chart.Error.Description)
	}

	if len(resp.Chart.Result) == 0 {
		return nil, fmt.Errorf("%w: empty result set", datasource.ErrInvalidResponse)
	}

	result := resp.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("%w: no quote data", datasource.ErrInvalidResponse)
	}

	quote := result.Indicators.Quote[0]
	n := len(result.Timestamp)
	prices := make([]datasource.PriceData, 0, n)

	for i := range n {
		if i >= len(quote.Open) || i >= len(quote.High) || i >= len(quote.Low) ||
			i >= len(quote.Close) || i >= len(quote.Volume) {
			break
		}
		if !quote.Close[i].Valid {
			continue
		}

		prices = append(prices, datasource.PriceData{
			Date:   time.Unix(result.Timestamp[i], 0).UTC().Truncate(24 * time.Hour),
			Open:   quote.Open[i].Value,
			High:   quote.High[i].Value,
			Low:    quote.Low[i].Value,
			Close:  quote.Close[i].Value,
			Volume: quote.Volume[i].Value,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("%w: all data points were null", datasource.ErrInvalidResponse)
	}

	return prices, nil
}

// FetchQuote fetches the latest price for a symbol (uses 1-day range).
func (c *Client) FetchQuote(ctx context.Context, symbol string) (*datasource.PriceData, error) {
	prices, err := c.FetchPriceHistory(ctx, symbol, 5)
	if err != nil {
		return nil, err
	}
	if len(prices) == 0 {
		return nil, fmt.Errorf("%w: no recent data for %s", datasource.ErrInvalidResponse, symbol)
	}
	latest := prices[len(prices)-1]
	return &latest, nil
}
