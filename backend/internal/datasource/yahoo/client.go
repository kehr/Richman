package yahoo

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/richman/backend/internal/datasource"
	"go.uber.org/zap"
)

const (
	chartBaseURL = "https://query1.finance.yahoo.com/v8/finance/chart"
	stooqBaseURL = "https://stooq.com/q/d/l/"
)

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
// It first tries Yahoo Finance; on failure it falls back to Stooq.
func (c *Client) FetchPriceHistory(ctx context.Context, symbol string, days int) ([]datasource.PriceData, error) {
	if days <= 0 {
		days = 30
	}

	url := fmt.Sprintf("%s/%s?range=%dd&interval=1d", chartBaseURL, symbol, days)
	c.logger.Info("fetching price history from yahoo", zap.String("symbol", symbol), zap.String("url", url))

	body, yahooErr := datasource.FetchWithRetry(ctx, c.httpClient, url)
	if yahooErr == nil {
		prices, parseErr := ParseChartResponse(body)
		if parseErr == nil {
			return prices, nil
		}
		yahooErr = parseErr
	}

	c.logger.Warn("yahoo fetch failed, falling back to stooq",
		zap.String("symbol", symbol),
		zap.Error(yahooErr),
	)

	prices, stooqErr := c.fetchFromStooq(ctx, symbol, days)
	if stooqErr != nil {
		return nil, fmt.Errorf("yahoo: %w; stooq: %v", yahooErr, stooqErr)
	}
	return prices, nil
}

// fetchFromStooq fetches daily OHLCV from Stooq as a fallback data source.
// d1 is set to today - days*2 to buffer for weekends and holidays;
// the result is trimmed to the last `days` data points.
func (c *Client) fetchFromStooq(ctx context.Context, symbol string, days int) ([]datasource.PriceData, error) {
	now := time.Now().UTC()
	d2 := now.Format("20060102")
	d1 := now.AddDate(0, 0, -days*2).Format("20060102")

	url := fmt.Sprintf("%s?s=%s.US&i=d&d1=%s&d2=%s", stooqBaseURL, symbol, d1, d2)
	c.logger.Info("fetching price history from stooq", zap.String("symbol", symbol), zap.String("url", url))

	body, err := datasource.FetchWithRetry(ctx, c.httpClient, url)
	if err != nil {
		return nil, fmt.Errorf("fetch stooq for %s: %w", symbol, err)
	}

	return parseStooqCSV(body, days)
}

// parseStooqCSV parses Stooq CSV response into PriceData slices.
// It skips the header row, handles missing Volume gracefully,
// sorts by date ascending, and returns the last `limit` data points.
func parseStooqCSV(data []byte, limit int) ([]datasource.PriceData, error) {
	r := csv.NewReader(bytes.NewReader(data))

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: parse stooq csv: %v", datasource.ErrInvalidResponse, err)
	}

	// records[0] is the header row; skip it
	if len(records) <= 1 {
		return []datasource.PriceData{}, nil
	}

	prices := make([]datasource.PriceData, 0, len(records)-1)
	for _, row := range records[1:] {
		if len(row) < 5 {
			continue
		}

		date, err := time.Parse("2006-01-02", row[0])
		if err != nil {
			continue
		}

		open, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			continue
		}
		high, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			continue
		}
		closePrice, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			continue
		}

		var volume float64
		if len(row) >= 6 && row[5] != "" {
			volume, _ = strconv.ParseFloat(row[5], 64)
		}

		prices = append(prices, datasource.PriceData{
			Date:   date.UTC(),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		})
	}

	// Sort ascending by date (Stooq returns newest first)
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})

	if len(prices) > limit {
		prices = prices[len(prices)-limit:]
	}

	return prices, nil
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

// FetchForexRate fetches the latest exchange rate for a forex pair ticker.
// ticker uses Yahoo Finance format, e.g. "USDCNY=X" returns how many CNY
// equals 1 USD (the Close price of the pair). Callers that need the inverse
// rate (1 CNY = X USD) must compute 1/result themselves.
func (c *Client) FetchForexRate(ctx context.Context, ticker string) (float64, error) {
	quote, err := c.FetchQuote(ctx, ticker)
	if err != nil {
		return 0, fmt.Errorf("fetch forex rate for %s: %w", ticker, err)
	}
	if quote.Close <= 0 {
		return 0, fmt.Errorf("invalid forex rate %.6f for %s", quote.Close, ticker)
	}
	return quote.Close, nil
}
