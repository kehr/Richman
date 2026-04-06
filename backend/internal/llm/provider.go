package llm

import (
	"context"
	"time"
)

// Provider defines the interface for LLM chat completion providers.
type Provider interface {
	// ChatCompletion sends a prompt and returns the response text.
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// Name returns the provider name.
	Name() string
}

// ChatRequest holds the parameters for an LLM chat completion call.
type ChatRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
}

// ChatResponse holds the result of an LLM chat completion call.
type ChatResponse struct {
	Content    string
	TokensUsed int
	Model      string
	Latency    time.Duration
}
