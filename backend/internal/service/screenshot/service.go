// Package screenshot implements the portfolio screenshot recognition
// service. It takes a raw image byte slice, dispatches a vision-model
// request and returns a structured preview for the user to confirm.
//
// The service is deliberately stateless: it never persists the image
// and never writes holdings. Persistence happens only after the user
// confirms the preview via the existing POST /api/v1/holdings endpoint.
package screenshot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// Limits and defaults. See TRD §4.5 and §4.6.
const (
	// MaxImageBytes is the maximum image size accepted at the service
	// layer. Kept aligned with the vision client's own cap so that a
	// rejected upload never reaches the upstream provider.
	MaxImageBytes = 5 * 1024 * 1024

	// DefaultRateLimitPerUser is the per-user request budget per window.
	DefaultRateLimitPerUser = 10
	// DefaultRateLimitWindow is the length of the rate-limit window.
	DefaultRateLimitWindow = time.Hour

	// VisionMaxTokens caps the vision response length.
	VisionMaxTokens = 1024
	// VisionTemperature keeps the extraction deterministic.
	VisionTemperature = 0.0
)

// Supported image mime types. Anything else is rejected before hitting
// the upstream provider.
var supportedMIMEs = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
	"image/gif":  {},
}

// RecognizeRequest carries the raw image and its MIME type.
type RecognizeRequest struct {
	ImageData []byte
	ImageMIME string
}

// RecognizeResponse is the structured preview returned to the client.
// See TRD §4.3.
type RecognizeResponse struct {
	Holdings      []RecognizedHolding `json:"holdings"`
	OverallStatus string              `json:"overallStatus"`
	Warning       string              `json:"warning,omitempty"`
}

// RecognizedHolding groups the fields extracted for a single position.
type RecognizedHolding struct {
	AssetName      Field  `json:"assetName"`
	AssetCode      Field  `json:"assetCode"`
	CostPrice      Field  `json:"costPrice"`
	PositionPct    Field  `json:"positionPct"`
	AssetTypeGuess string `json:"assetTypeGuess"`
}

// Field is a single extracted value plus a calibrated confidence.
type Field struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Options tunes the Service. Zero values fall back to sensible defaults.
type Options struct {
	RateLimit       int
	RateLimitWindow time.Duration
	// Now is injected so tests can simulate the passage of time.
	Now func() time.Time
}

// Service performs vision-model recognition of portfolio screenshots.
// Safe for concurrent use.
type Service struct {
	vision llm.VisionProvider
	logger *zap.Logger

	limit  int
	window time.Duration
	now    func() time.Time

	mu       sync.Mutex
	attempts map[int64][]time.Time
}

// NewService wires a Service around a VisionProvider. Passing a nil
// provider is allowed so the rest of the system can boot in degraded
// mode; in that case Recognize returns a "failed" preview for every
// request.
func NewService(vision llm.VisionProvider, logger *zap.Logger, opts Options) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	limit := opts.RateLimit
	if limit <= 0 {
		limit = DefaultRateLimitPerUser
	}
	window := opts.RateLimitWindow
	if window <= 0 {
		window = DefaultRateLimitWindow
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		vision:   vision,
		logger:   logger,
		limit:    limit,
		window:   window,
		now:      now,
		attempts: make(map[int64][]time.Time),
	}
}

// Recognize validates the upload, consumes the user's rate-limit quota,
// calls the vision provider and parses the response. It never persists
// the image bytes. All degraded states are mapped to RecognizeResponse
// with a descriptive warning, except size / rate-limit / auth failures
// which return an *model.AppError so the handler can emit the correct
// HTTP status.
func (s *Service) Recognize(ctx context.Context, userID int64, req RecognizeRequest) (*RecognizeResponse, error) {
	if userID <= 0 {
		return nil, model.ErrUnauthorized
	}

	if err := s.validate(req); err != nil {
		return nil, err
	}

	if !s.allow(userID) {
		return nil, model.NewAppError(
			429,
			"RATE_LIMITED",
			fmt.Sprintf("screenshot recognition is limited to %d requests per hour", s.limit),
		)
	}

	if s.vision == nil {
		s.logger.Warn("screenshot vision provider unavailable")
		return failedResponse("识别服务暂时不可用"), nil
	}

	visionResp, err := s.vision.AnalyzeImage(ctx, llm.VisionRequest{
		SystemPrompt: SystemPrompt,
		UserPrompt:   UserPrompt,
		ImageData:    req.ImageData,
		ImageMIME:    req.ImageMIME,
		MaxTokens:    VisionMaxTokens,
		Temperature:  VisionTemperature,
	})
	if err != nil {
		return s.handleVisionError(err), nil
	}

	parsed, parseErr := Parse(visionResp.Content)
	if parseErr != nil {
		s.logger.Warn("screenshot llm response was not valid json",
			zap.String("provider", s.vision.Name()),
			zap.Error(parseErr),
		)
		return failedResponse("识别结果解析失败，请重试"), nil
	}

	return parsed, nil
}

// validate performs cheap pre-flight checks that never need the network.
func (s *Service) validate(req RecognizeRequest) error {
	if len(req.ImageData) == 0 {
		return model.NewValidationError("image data is required")
	}
	if len(req.ImageData) > MaxImageBytes {
		return model.NewAppError(
			413,
			"FILE_TOO_LARGE",
			fmt.Sprintf("image must be no larger than %d bytes", MaxImageBytes),
		)
	}
	if _, ok := supportedMIMEs[req.ImageMIME]; !ok {
		return model.NewValidationError(
			"unsupported image type; expected image/png, image/jpeg, image/webp or image/gif",
		)
	}
	return nil
}

// allow implements a fixed-window counter per user.
// We prune expired timestamps on each call so the map never grows
// unboundedly for active users. This is intentionally simple and
// in-process; Redis can replace it later without changing the API.
func (s *Service) allow(userID int64) bool {
	now := s.now()
	cutoff := now.Add(-s.window)

	s.mu.Lock()
	defer s.mu.Unlock()

	timestamps := s.attempts[userID]
	// Drop stale entries in-place.
	kept := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= s.limit {
		s.attempts[userID] = kept
		return false
	}
	s.attempts[userID] = append(kept, now)
	return true
}

// handleVisionError converts a typed VisionProvider error into a
// user-friendly RecognizeResponse. All errors degrade to StatusFailed
// so the frontend can show a retry affordance without inspecting HTTP
// status codes.
func (s *Service) handleVisionError(err error) *RecognizeResponse {
	switch {
	case errors.Is(err, llm.ErrVisionTimeout):
		s.logger.Warn("screenshot vision call timed out", zap.Error(err))
		return failedResponse("识别超时，请稍后重试")
	case errors.Is(err, llm.ErrVisionRateLimited):
		s.logger.Warn("screenshot vision upstream rate limited", zap.Error(err))
		return failedResponse("识别服务繁忙，请稍后重试")
	case errors.Is(err, llm.ErrVisionInvalidRequest):
		s.logger.Warn("screenshot vision rejected request", zap.Error(err))
		return failedResponse("图像无法处理，请更换截图")
	default:
		s.logger.Warn("screenshot vision call failed", zap.Error(err))
		return failedResponse("识别服务暂时不可用")
	}
}

// failedResponse builds a pre-graded failure payload with a warning.
func failedResponse(warning string) *RecognizeResponse {
	return &RecognizeResponse{
		Holdings:      []RecognizedHolding{},
		OverallStatus: StatusFailed,
		Warning:       warning,
	}
}
