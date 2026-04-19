package richson

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/richman/backend/internal/config"
	"go.uber.org/zap"
)

const (
	asyncTimeout = 5 * time.Second
	// syncTimeout must exceed richson's internal _run_agent timeout (60s in
	// content.py) plus a small buffer; otherwise richman aborts while richson
	// keeps running, producing duplicate LLM cost.
	syncTimeout  = 65 * time.Second
	lightTimeout = 10 * time.Second
	// radarTimeout covers /events/radar specifically. The endpoint fans
	// out to FRED (cold path observed ~16s) and Polymarket (~4s) via
	// asyncio.gather; the richson FRED httpx timeout is 18s, so the
	// backend budget must be strictly larger to observe a real response
	// rather than cancel mid-flight. 20s is the smallest safe value.
	radarTimeout  = 20 * time.Second
	healthTimeout = 3 * time.Second
	retryDelay    = 2 * time.Second
)

// richsonErrorMap maps richson error codes to HTTP status codes.
// Codes mirror those raised inside richson; keep this map in sync with
// richson/src/richson/api/*.py and richson/src/richson/core/pipeline.py.
var richsonErrorMap = map[string]int{
	"ANALYSIS_IN_PROGRESS":    http.StatusConflict,
	"DATA_SOURCE_UNAVAILABLE": http.StatusBadGateway,
	"LLM_INVALID_RESPONSE":    http.StatusBadGateway,
	"PIPELINE_ERROR":          http.StatusBadGateway,
	"ASSET_NOT_FOUND":         http.StatusNotFound,
	"JOB_NOT_FOUND":           http.StatusNotFound,
	"INSUFFICIENT_HISTORY":    http.StatusBadRequest,
	"UNAUTHORIZED":            http.StatusUnauthorized,
}

// RichsonError represents an error returned by the richson sidecar.
type RichsonError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *RichsonError) Error() string {
	return fmt.Sprintf("richson error %s: %s", e.Code, e.Message)
}

// contextKey is the unexported type used for values stored in context.
type contextKey string

const requestIDKey contextKey = "request_id"

// Client communicates with the richson Python sidecar over HTTP.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
	healthy    atomic.Bool
}

// NewClient constructs a Client from the given configuration and logger.
// The initial healthy state is false; a background cron should call
// HealthCheck periodically to update it.
func NewClient(cfg config.RichsonConfig, logger *zap.Logger) *Client {
	c := &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			// No global timeout; each request sets its own via context.
			Timeout: 0,
		},
		logger: logger,
	}
	// Initial state is unhealthy until the first successful health check.
	c.healthy.Store(false)
	return c
}

// IsHealthy reports whether the last health check succeeded.
func (c *Client) IsHealthy() bool {
	return c.healthy.Load()
}

// setHeaders attaches auth and tracing headers to the request.
func (c *Client) setHeaders(req *http.Request, requestID string) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", requestID)
}

// extractRequestID returns the request ID from ctx if present, or generates a new UUID.
func extractRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		return id
	}
	return uuid.New().String()
}

// isRetryable returns true for network-level errors and HTTP 502/503 responses.
func isRetryable(err error, statusCode int) bool {
	if err != nil {
		// Any network or transport error is retryable.
		return true
	}
	return statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable
}

// doRequest performs one HTTP request with a per-request timeout derived from
// the given deadline duration. It applies one retry on retryable failures.
// Returns the raw response body on success, or an error.
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte, timeout time.Duration, maxRetries int) ([]byte, error) {
	requestID := extractRequestID(ctx)
	url := c.baseURL + path

	var lastErr error
	var lastStatus int

	attempts := maxRetries + 1
	for i := range attempts {
		if i > 0 {
			// Wait before retry, but respect context cancellation.
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled before retry: %w", ctx.Err())
			case <-time.After(retryDelay):
			}
		}

		tctx, cancel := context.WithTimeout(ctx, timeout)
		respBody, statusCode, err := c.executeRequest(tctx, method, url, body, requestID)
		cancel()

		if err == nil {
			return respBody, nil
		}

		if !isRetryable(err, statusCode) {
			return nil, err
		}

		lastErr = err
		lastStatus = statusCode
		c.logger.Warn("richson request failed, will retry",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("attempt", i+1),
			zap.Int("status", lastStatus),
			zap.Error(err),
		)
	}

	return nil, lastErr
}

// executeRequest performs a single HTTP call and returns the body, status code, and error.
// A non-2xx status is converted to a *RichsonError when richson provides a structured body.
func (c *Client) executeRequest(ctx context.Context, method, url string, body []byte, requestID string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = http.NoBody
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req, requestID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, c.parseErrorResponse(respBody, resp.StatusCode)
	}

	return respBody, resp.StatusCode, nil
}

// parseErrorResponse converts a richson error body into a *RichsonError.
func (c *Client) parseErrorResponse(body []byte, statusCode int) error {
	var errResp ErrorResponse
	if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error.Code != "" {
		httpStatus := statusCode
		if mapped, ok := richsonErrorMap[errResp.Error.Code]; ok {
			httpStatus = mapped
		}
		return &RichsonError{
			Code:       errResp.Error.Code,
			Message:    errResp.Error.Message,
			HTTPStatus: httpStatus,
		}
	}
	return fmt.Errorf("richson returned HTTP %d: %s", statusCode, string(body))
}

// decodeDataResponse unwraps a {"data": <T>} envelope into dst.
func decodeDataResponse[T any](body []byte) (T, error) {
	var envelope DataResponse[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		var zero T
		return zero, fmt.Errorf("decode response: %w", err)
	}
	return envelope.Data, nil
}

// marshalBody encodes v to JSON for use as a request body.
func marshalBody(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return b, nil
}

// IsRichsonError returns the *RichsonError if err is one, otherwise nil.
func IsRichsonError(err error) (*RichsonError, bool) {
	var re *RichsonError
	if errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

// ---- Public Methods ----

// TriggerAssetAnalysis sends POST /jobs/analyze-asset and returns a JobResponse.
func (c *Client) TriggerAssetAnalysis(ctx context.Context, req TriggerAssetAnalysisRequest) (*JobResponse, error) {
	req.RequestID = extractRequestID(ctx)
	body, err := marshalBody(req)
	if err != nil {
		return nil, err
	}

	raw, err := c.doRequest(ctx, http.MethodPost, "/jobs/analyze-asset", body, asyncTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[JobResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// TriggerBatchAnalysis sends POST /jobs/batch-analyze and returns a BatchJobResponse.
func (c *Client) TriggerBatchAnalysis(ctx context.Context, req TriggerBatchAnalysisRequest) (*BatchJobResponse, error) {
	req.RequestID = extractRequestID(ctx)
	body, err := marshalBody(req)
	if err != nil {
		return nil, err
	}

	raw, err := c.doRequest(ctx, http.MethodPost, "/jobs/batch-analyze", body, asyncTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[BatchJobResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetJobStatus sends GET /jobs/{jobId} and returns a JobDetailResponse.
func (c *Client) GetJobStatus(ctx context.Context, jobID string) (*JobDetailResponse, error) {
	path := "/jobs/" + jobID
	raw, err := c.doRequest(ctx, http.MethodGet, path, nil, lightTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[JobDetailResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AnalyzeHolding sends POST /analyze/holding and returns a HoldingAnalysisResponse.
func (c *Client) AnalyzeHolding(ctx context.Context, req AnalyzeHoldingRequest) (*HoldingAnalysisResponse, error) {
	req.RequestID = extractRequestID(ctx)
	body, err := marshalBody(req)
	if err != nil {
		return nil, err
	}

	raw, err := c.doRequest(ctx, http.MethodPost, "/analyze/holding", body, syncTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[HoldingAnalysisResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDemoPlan sends POST /analyze/demo-plan and returns a DemoPlanResponse.
func (c *Client) GetDemoPlan(ctx context.Context, req DemoPlanRequest) (*DemoPlanResponse, error) {
	req.RequestID = extractRequestID(ctx)
	body, err := marshalBody(req)
	if err != nil {
		return nil, err
	}

	raw, err := c.doRequest(ctx, http.MethodPost, "/analyze/demo-plan", body, lightTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[DemoPlanResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMarketRegime sends GET /market/regime and returns a MarketRegimeResponse.
func (c *Client) GetMarketRegime(ctx context.Context) (*MarketRegimeResponse, error) {
	raw, err := c.doRequest(ctx, http.MethodGet, "/market/regime", nil, lightTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[MarketRegimeResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOHLCV sends GET /market/ohlcv/{code} and returns an OHLCVResponse.
func (c *Client) GetOHLCV(ctx context.Context, code string) (*OHLCVResponse, error) {
	path := "/market/ohlcv/" + code
	raw, err := c.doRequest(ctx, http.MethodGet, path, nil, lightTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[OHLCVResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetScoreHistory sends GET /assets/{code}/score-history and returns a ScoreHistoryResponse.
func (c *Client) GetScoreHistory(ctx context.Context, code string) (*ScoreHistoryResponse, error) {
	path := "/assets/" + code + "/score-history"
	raw, err := c.doRequest(ctx, http.MethodGet, path, nil, lightTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[ScoreHistoryResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetEventsRadar sends GET /events/radar and returns an EventsRadarResponse.
// Uses radarTimeout (not lightTimeout) because the endpoint fans out to slow
// external APIs (FRED, Polymarket) and the cold path needs more than 10s.
// Retry count is 0: the endpoint is read-only and idempotent, but a real
// timeout here almost always means upstream is slow, so a retry doubles the
// user wait without changing the outcome. If it matters, richson's scheduler
// warmup keeps the cache hot.
func (c *Client) GetEventsRadar(ctx context.Context) (*EventsRadarResponse, error) {
	raw, err := c.doRequest(ctx, http.MethodGet, "/events/radar", nil, radarTimeout, 0)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[EventsRadarResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateWeeklyInsight sends POST /content/weekly-insight and returns a WeeklyInsightResponse.
func (c *Client) GenerateWeeklyInsight(ctx context.Context, req WeeklyInsightRequest) (*WeeklyInsightResponse, error) {
	req.RequestID = extractRequestID(ctx)
	body, err := marshalBody(req)
	if err != nil {
		return nil, err
	}

	raw, err := c.doRequest(ctx, http.MethodPost, "/content/weekly-insight", body, syncTimeout, 1)
	if err != nil {
		return nil, err
	}

	result, err := decodeDataResponse[WeeklyInsightResponse](raw)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HealthCheck sends GET /health and updates the internal healthy flag.
// It uses a short 3s timeout and does not retry.
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	tctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()

	requestID := extractRequestID(ctx)
	raw, _, err := c.executeRequest(tctx, http.MethodGet, c.baseURL+"/health", nil, requestID)
	if err != nil {
		c.healthy.Store(false)
		c.logger.Warn("richson health check failed", zap.Error(err))
		return nil, err
	}

	var resp HealthResponse
	if jsonErr := json.Unmarshal(raw, &resp); jsonErr != nil {
		c.healthy.Store(false)
		return nil, fmt.Errorf("decode health response: %w", jsonErr)
	}

	healthy := resp.Status == "healthy" || resp.Status == "degraded"
	c.healthy.Store(healthy)

	c.logger.Debug("richson health check completed",
		zap.String("status", resp.Status),
		zap.Bool("healthy", healthy),
	)

	return &resp, nil
}
