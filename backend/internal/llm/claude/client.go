package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1/messages"
	defaultModel   = "claude-sonnet-4-20250514"
	apiVersion     = "2023-06-01"
)

// Client implements the llm.Provider interface using the Claude Messages API.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// Option configures the Claude client.
type Option func(*Client)

// WithModel sets the model to use.
func WithModel(model string) Option {
	return func(c *Client) {
		if model != "" {
			c.model = model
		}
	}
}

// WithBaseURL overrides the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

// NewClient creates a new Claude API client.
func NewClient(apiKey string, logger *zap.Logger, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		model:   defaultModel,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "claude"
}

// messagesRequest is the request body for the Claude Messages API.
type messagesRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	System      string    `json:"system,omitempty"`
	Messages    []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// messagesResponse is the response body from the Claude Messages API.
type messagesResponse struct {
	Content []contentBlock `json:"content"`
	Model   string         `json:"model"`
	Usage   usage          `json:"usage"`
	Error   *apiError      `json:"error,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ChatCompletion sends a chat completion request to the Claude API.
func (c *Client) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.3
	}

	body := messagesRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		System:      req.SystemPrompt,
		Messages: []message{
			{Role: "user", Content: req.UserPrompt},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", apiVersion)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("claude api error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("claude api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result messagesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("claude api error: [%s] %s", result.Error.Type, result.Error.Message)
	}

	content := ""
	for _, block := range result.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	tokensUsed := result.Usage.InputTokens + result.Usage.OutputTokens
	c.logger.Info("claude chat completion",
		zap.String("model", result.Model),
		zap.Int("tokens_used", tokensUsed),
		zap.Duration("latency", latency),
	)

	return &llm.ChatResponse{
		Content:    content,
		TokensUsed: tokensUsed,
		Model:      result.Model,
		Latency:    latency,
	}, nil
}
