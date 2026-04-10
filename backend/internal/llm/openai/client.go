package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

const (
	// defaultBaseURL is the OpenAI API v1 base path. ChatCompletion appends
	// /chat/completions so callers (and the openai_compatible provider) can
	// set a plain base URL without knowing the endpoint suffix.
	defaultBaseURL = "https://api.openai.com/v1"
	defaultModel   = "gpt-4o"
)

// Client implements the llm.Provider interface using the OpenAI Chat Completions API.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// Option configures the OpenAI client.
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

// NewClient creates a new OpenAI API client.
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
	return "openai"
}

// chatRequest is the request body for the OpenAI Chat Completions API.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the response body from the OpenAI Chat Completions API.
type chatResponse struct {
	Choices []choice    `json:"choices"`
	Model   string      `json:"model"`
	Usage   chatUsage   `json:"usage"`
	Error   *chatAPIErr `json:"error,omitempty"`
}

type choice struct {
	Message chatMessage `json:"message"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatAPIErr struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ChatCompletion sends a chat completion request to the OpenAI API.
func (c *Client) ChatCompletion(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.3
	}

	messages := make([]chatMessage, 0, 2)
	if req.SystemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: req.UserPrompt})

	body := chatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		c.logger.Warn("openai api error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("openai api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("openai api error: [%s] %s", result.Error.Type, result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai api returned no choices")
	}

	c.logger.Info("openai chat completion",
		zap.String("model", result.Model),
		zap.Int("tokens_used", result.Usage.TotalTokens),
		zap.Duration("latency", latency),
	)

	return &llm.ChatResponse{
		Content:    result.Choices[0].Message.Content,
		TokensUsed: result.Usage.TotalTokens,
		Model:      result.Model,
		Latency:    latency,
	}, nil
}
