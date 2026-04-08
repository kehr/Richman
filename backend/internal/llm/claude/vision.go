package claude

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

const (
	defaultVisionModel   = "claude-sonnet-4-20250514"
	defaultVisionTimeout = 30 * time.Second
)

// VisionClient implements llm.VisionProvider via the Anthropic Messages API
// vision capability. It is intentionally separate from Client (text) so the
// two can evolve independently, though they share auth conventions.
type VisionClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// VisionOption configures a VisionClient.
type VisionOption func(*VisionClient)

// WithVisionModel overrides the model id.
func WithVisionModel(model string) VisionOption {
	return func(c *VisionClient) {
		if model != "" {
			c.model = model
		}
	}
}

// WithVisionBaseURL overrides the API endpoint (for tests / self-hosted proxy).
func WithVisionBaseURL(url string) VisionOption {
	return func(c *VisionClient) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// WithVisionTimeout overrides the HTTP client timeout.
func WithVisionTimeout(d time.Duration) VisionOption {
	return func(c *VisionClient) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// NewVisionClient constructs a Claude vision client.
func NewVisionClient(apiKey string, logger *zap.Logger, opts ...VisionOption) *VisionClient {
	c := &VisionClient{
		apiKey:  apiKey,
		model:   defaultVisionModel,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultVisionTimeout,
		},
		logger: logger,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the vision provider name.
func (c *VisionClient) Name() string {
	return "claude"
}

// visionMessagesRequest is the request body for a vision-enabled Messages API call.
type visionMessagesRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	System      string          `json:"system,omitempty"`
	Messages    []visionMessage `json:"messages"`
}

type visionMessage struct {
	Role    string        `json:"role"`
	Content []visionBlock `json:"content"`
}

// visionBlock supports both image and text content blocks. Unused fields
// serialize as zero-value; the Anthropic API ignores unknown keys.
type visionBlock struct {
	Type   string        `json:"type"`
	Text   string        `json:"text,omitempty"`
	Source *visionSource `json:"source,omitempty"`
}

type visionSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// visionMessagesResponse mirrors the text response but we keep it local to
// avoid coupling to the text client's structs.
type visionMessagesResponse struct {
	Content []visionContentBlock `json:"content"`
	Model   string               `json:"model"`
	Usage   visionUsage          `json:"usage"`
	Error   *visionAPIError      `json:"error,omitempty"`
}

type visionContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type visionUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type visionAPIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// maxVisionImageBytes bounds the raw image payload size accepted by this
// client before base64 encoding. Anthropic's documented limit is ~5 MB for
// the encoded payload, so 5 MB raw leaves headroom for the base64 overhead
// and message envelope while catching obviously-oversized screenshots
// locally rather than round-tripping to the API.
const maxVisionImageBytes = 5 * 1024 * 1024

// allowedVisionMIMEs lists the image MIME types accepted by the Claude
// Messages API. Callers passing anything else (for example image/bmp or
// image/svg+xml) get a typed ErrVisionInvalidRequest up front, so the
// screenshot service can treat it as non-retryable without probing the API.
var allowedVisionMIMEs = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
}

// AnalyzeImage sends image bytes plus prompts to the Claude Messages API.
func (c *VisionClient) AnalyzeImage(ctx context.Context, req *llm.VisionRequest) (*llm.VisionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", llm.ErrVisionInvalidRequest)
	}
	if len(req.ImageData) == 0 {
		return nil, fmt.Errorf("%w: image data is empty", llm.ErrVisionInvalidRequest)
	}
	if req.ImageMIME == "" {
		return nil, fmt.Errorf("%w: image mime is empty", llm.ErrVisionInvalidRequest)
	}
	if _, ok := allowedVisionMIMEs[req.ImageMIME]; !ok {
		return nil, fmt.Errorf(
			"%w: unsupported image mime %q (allowed: image/jpeg, image/png, image/gif, image/webp)",
			llm.ErrVisionInvalidRequest, req.ImageMIME,
		)
	}
	if len(req.ImageData) > maxVisionImageBytes {
		return nil, fmt.Errorf(
			"%w: image payload %d bytes exceeds %d byte limit",
			llm.ErrVisionInvalidRequest, len(req.ImageData), maxVisionImageBytes,
		)
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	temperature := req.Temperature
	if temperature < 0 {
		temperature = 0.4
	}

	encoded := base64.StdEncoding.EncodeToString(req.ImageData)

	body := visionMessagesRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		System:      req.SystemPrompt,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionBlock{
					{
						Type: "image",
						Source: &visionSource{
							Type:      "base64",
							MediaType: req.ImageMIME,
							Data:      encoded,
						},
					},
					{
						Type: "text",
						Text: req.UserPrompt,
					},
				},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal request: %v", llm.ErrVisionInvalidRequest, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %v", llm.ErrVisionInvalidRequest, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", apiVersion)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		// Context cancellation / deadline bubble up as typed timeout errors so
		// the caller (screenshot service) can classify and degrade gracefully.
		if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", llm.ErrVisionTimeout, ctxErr)
		}
		return nil, fmt.Errorf("%w: %v", llm.ErrVisionNetwork, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %v", llm.ErrVisionNetwork, err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("claude vision api error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			return nil, fmt.Errorf("%w: status=%d body=%s", llm.ErrVisionRateLimited, resp.StatusCode, string(respBody))
		case resp.StatusCode >= 500:
			return nil, fmt.Errorf("%w: status=%d body=%s", llm.ErrVisionServer, resp.StatusCode, string(respBody))
		default:
			return nil, fmt.Errorf("%w: status=%d body=%s", llm.ErrVisionClient, resp.StatusCode, string(respBody))
		}
	}

	var result visionMessagesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", llm.ErrVisionDecode, err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%w: [%s] %s", llm.ErrVisionClient, result.Error.Type, result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("%w: empty content array", llm.ErrVisionDecode)
	}

	content := ""
	for _, block := range result.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	c.logger.Info("claude vision analyze",
		zap.String("model", result.Model),
		zap.Int("input_tokens", result.Usage.InputTokens),
		zap.Int("output_tokens", result.Usage.OutputTokens),
		zap.Duration("latency", latency),
	)

	return &llm.VisionResponse{
		Content: content,
		Model:   result.Model,
		UsageHint: map[string]any{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
			"latency_ms":    latency.Milliseconds(),
		},
	}, nil
}
