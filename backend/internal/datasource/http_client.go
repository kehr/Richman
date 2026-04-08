package datasource

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	baseBackoff       = 1 * time.Second
)

// NewHTTPClient creates a shared HTTP client with sensible defaults.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultTimeout,
	}
}

// FetchWithRetry performs an HTTP GET with exponential backoff retries.
// It returns the response body bytes on success or ErrDataSourceUnavailable after exhausting retries.
func FetchWithRetry(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	var lastErr error

	for attempt := range defaultMaxRetries {
		if attempt > 0 {
			backoff := baseBackoff * time.Duration(math.Pow(2, float64(attempt-1)))
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ErrDataSourceUnavailable, ctx.Err())
			case <-time.After(backoff):
			}
		}

		body, err := doGet(ctx, client, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("%w: %v", ErrDataSourceUnavailable, lastErr)
}

// doGet performs a single HTTP GET request and returns the response body.
func doGet(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return body, nil
}
