package llm

import (
	"context"
	"errors"
)

// VisionProvider defines the interface for LLM providers that can analyze images.
// This interface is decoupled from the text-only Provider interface so vision
// backends can evolve independently.
type VisionProvider interface {
	// AnalyzeImage sends an image plus prompts and returns the assistant reply.
	AnalyzeImage(ctx context.Context, req VisionRequest) (*VisionResponse, error)
	// Name returns the vision provider name.
	Name() string
}

// VisionRequest holds the parameters for a vision analysis call.
type VisionRequest struct {
	SystemPrompt string
	UserPrompt   string
	ImageData    []byte
	ImageMIME    string // e.g. "image/png", "image/jpeg"
	MaxTokens    int
	Temperature  float64
}

// VisionResponse holds the result of a vision analysis call.
type VisionResponse struct {
	Content   string
	Model     string
	UsageHint map[string]any
}

// Typed errors so callers can classify failures and apply fallback strategies.
var (
	// ErrVisionTimeout indicates the request was cancelled or deadline exceeded.
	ErrVisionTimeout = errors.New("vision: request timeout or cancelled")
	// ErrVisionRateLimited indicates the upstream returned HTTP 429.
	ErrVisionRateLimited = errors.New("vision: rate limited by upstream")
	// ErrVisionClient indicates a non-429 4xx error (caller/request issue).
	ErrVisionClient = errors.New("vision: client error from upstream")
	// ErrVisionServer indicates a 5xx error from upstream.
	ErrVisionServer = errors.New("vision: server error from upstream")
	// ErrVisionDecode indicates a response JSON parse failure.
	ErrVisionDecode = errors.New("vision: failed to decode response")
	// ErrVisionNetwork indicates a transport-level error (DNS, conn reset, etc.).
	ErrVisionNetwork = errors.New("vision: network error")
	// ErrVisionInvalidRequest indicates the request was invalid before sending.
	ErrVisionInvalidRequest = errors.New("vision: invalid request")
)
