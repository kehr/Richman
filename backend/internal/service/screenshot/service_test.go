package screenshot

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap/zaptest"
)

// fakeVision is a hand-rolled VisionProvider for deterministic tests.
type fakeVision struct {
	calls    int
	lastReq  llm.VisionRequest
	response *llm.VisionResponse
	err      error
}

func (f *fakeVision) AnalyzeImage(_ context.Context, req llm.VisionRequest) (*llm.VisionResponse, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f *fakeVision) Name() string { return "fake" }

func newTestService(t *testing.T, v llm.VisionProvider, opts Options) *Service {
	t.Helper()
	return NewService(v, zaptest.NewLogger(t), opts)
}

func smallPNG() []byte {
	// Minimal 1x1 png is fine; content is not inspected in tests.
	return bytes.Repeat([]byte{0x89}, 16)
}

const validHighConfidenceJSON = `{
	"holdings": [
		{
			"assetName":   {"value": "AAPL", "confidence": 0.95},
			"assetCode":   {"value": "AAPL", "confidence": 0.95},
			"costPrice":   {"value": "150",  "confidence": 0.9},
			"positionPct": {"value": "25",   "confidence": 0.9},
			"assetTypeGuess": "us_stock"
		}
	]
}`

func TestService_Recognize_HappyPath(t *testing.T) {
	fv := &fakeVision{
		response: &llm.VisionResponse{Content: validHighConfidenceJSON, Model: "fake"},
	}
	svc := newTestService(t, fv, Options{})

	resp, err := svc.Recognize(context.Background(), 42, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OverallStatus != StatusOK {
		t.Errorf("want status ok, got %q", resp.OverallStatus)
	}
	if len(resp.Holdings) != 1 || resp.Holdings[0].AssetName.Value != "AAPL" {
		t.Errorf("holdings mismatch: %+v", resp.Holdings)
	}
	if fv.calls != 1 {
		t.Errorf("vision call count: want 1, got %d", fv.calls)
	}
	if fv.lastReq.SystemPrompt == "" || fv.lastReq.UserPrompt == "" {
		t.Error("prompts were not forwarded to vision provider")
	}
}

func TestService_Recognize_RejectsOversizeImage(t *testing.T) {
	fv := &fakeVision{}
	svc := newTestService(t, fv, Options{})

	big := bytes.Repeat([]byte{0x00}, MaxImageBytes+1)
	_, err := svc.Recognize(context.Background(), 7, RecognizeRequest{
		ImageData: big,
		ImageMIME: "image/png",
	})
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != 413 || appErr.Code != "FILE_TOO_LARGE" {
		t.Errorf("unexpected AppError: %+v", appErr)
	}
	if fv.calls != 0 {
		t.Error("vision provider must not be called for oversize images")
	}
}

func TestService_Recognize_RejectsUnsupportedMIME(t *testing.T) {
	svc := newTestService(t, &fakeVision{}, Options{})
	_, err := svc.Recognize(context.Background(), 1, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/bmp",
	})
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("want VALIDATION_ERROR AppError, got %v", err)
	}
}

func TestService_Recognize_RejectsMissingUser(t *testing.T) {
	svc := newTestService(t, &fakeVision{}, Options{})
	_, err := svc.Recognize(context.Background(), 0, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

func TestService_Recognize_RateLimitTriggers(t *testing.T) {
	fv := &fakeVision{
		response: &llm.VisionResponse{Content: validHighConfidenceJSON},
	}
	svc := newTestService(t, fv, Options{RateLimit: 10, RateLimitWindow: time.Hour})

	req := RecognizeRequest{ImageData: smallPNG(), ImageMIME: "image/png"}
	for i := 0; i < 10; i++ {
		if _, err := svc.Recognize(context.Background(), 99, req); err != nil {
			t.Fatalf("request %d unexpectedly failed: %v", i+1, err)
		}
	}
	_, err := svc.Recognize(context.Background(), 99, req)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != 429 {
		t.Fatalf("want 429 AppError on 11th call, got %v", err)
	}
	if fv.calls != 10 {
		t.Errorf("want 10 upstream calls, got %d", fv.calls)
	}
}

func TestService_Recognize_RateLimitIsPerUser(t *testing.T) {
	fv := &fakeVision{response: &llm.VisionResponse{Content: validHighConfidenceJSON}}
	svc := newTestService(t, fv, Options{RateLimit: 2})

	req := RecognizeRequest{ImageData: smallPNG(), ImageMIME: "image/png"}
	for i := 0; i < 2; i++ {
		if _, err := svc.Recognize(context.Background(), 1, req); err != nil {
			t.Fatalf("user 1 call %d failed: %v", i, err)
		}
	}
	// User 2 must still have quota.
	if _, err := svc.Recognize(context.Background(), 2, req); err != nil {
		t.Errorf("user 2 should not be rate-limited: %v", err)
	}
	// User 1 is now over quota.
	if _, err := svc.Recognize(context.Background(), 1, req); err == nil {
		t.Error("user 1 should be rate-limited")
	}
}

func TestService_Recognize_RateLimitWindowExpires(t *testing.T) {
	fv := &fakeVision{response: &llm.VisionResponse{Content: validHighConfidenceJSON}}
	current := time.Unix(1_700_000_000, 0)
	svc := newTestService(t, fv, Options{
		RateLimit:       1,
		RateLimitWindow: time.Minute,
		Now:             func() time.Time { return current },
	})

	req := RecognizeRequest{ImageData: smallPNG(), ImageMIME: "image/png"}
	if _, err := svc.Recognize(context.Background(), 5, req); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if _, err := svc.Recognize(context.Background(), 5, req); err == nil {
		t.Fatal("second call within window should be denied")
	}
	// Advance past the window boundary.
	current = current.Add(2 * time.Minute)
	if _, err := svc.Recognize(context.Background(), 5, req); err != nil {
		t.Errorf("call after window should succeed: %v", err)
	}
}

func TestService_Recognize_VisionTimeoutDegrades(t *testing.T) {
	fv := &fakeVision{err: llm.ErrVisionTimeout}
	svc := newTestService(t, fv, Options{})

	resp, err := svc.Recognize(context.Background(), 11, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("timeout should not bubble up as error: %v", err)
	}
	if resp.OverallStatus != StatusFailed {
		t.Errorf("want status failed, got %q", resp.OverallStatus)
	}
	if resp.Warning == "" {
		t.Error("warning should be populated on failure")
	}
}

func TestService_Recognize_VisionServerErrorDegrades(t *testing.T) {
	fv := &fakeVision{err: llm.ErrVisionServer}
	svc := newTestService(t, fv, Options{})

	resp, err := svc.Recognize(context.Background(), 12, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("server error should not bubble up: %v", err)
	}
	if resp.OverallStatus != StatusFailed {
		t.Errorf("want status failed, got %q", resp.OverallStatus)
	}
}

func TestService_Recognize_InvalidJSONDegrades(t *testing.T) {
	fv := &fakeVision{response: &llm.VisionResponse{Content: "not json"}}
	svc := newTestService(t, fv, Options{})

	resp, err := svc.Recognize(context.Background(), 13, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("parse failure should not bubble up: %v", err)
	}
	if resp.OverallStatus != StatusFailed {
		t.Errorf("want status failed, got %q", resp.OverallStatus)
	}
}

func TestService_Recognize_LowConfidenceStatus(t *testing.T) {
	low := `{"holdings":[{"assetName":{"value":"?","confidence":0.2},"assetCode":{"value":"","confidence":0},"costPrice":{"value":"","confidence":0},"positionPct":{"value":"","confidence":0},"assetTypeGuess":""}]}`
	fv := &fakeVision{response: &llm.VisionResponse{Content: low}}
	svc := newTestService(t, fv, Options{})

	resp, err := svc.Recognize(context.Background(), 14, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OverallStatus != StatusLowQuality {
		t.Errorf("want low_quality, got %q", resp.OverallStatus)
	}
}

func TestService_Recognize_NilVisionProviderDegrades(t *testing.T) {
	svc := newTestService(t, nil, Options{})
	resp, err := svc.Recognize(context.Background(), 15, RecognizeRequest{
		ImageData: smallPNG(),
		ImageMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("nil provider should degrade, not error: %v", err)
	}
	if resp.OverallStatus != StatusFailed {
		t.Errorf("want status failed, got %q", resp.OverallStatus)
	}
}

func TestService_Recognize_EmptyImageData(t *testing.T) {
	svc := newTestService(t, &fakeVision{}, Options{})
	_, err := svc.Recognize(context.Background(), 16, RecognizeRequest{
		ImageData: nil,
		ImageMIME: "image/png",
	})
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("want validation error, got %v", err)
	}
}
