package claude

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

func newTestVisionClient(t *testing.T, handler http.HandlerFunc) (*VisionClient, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := NewVisionClient("test-key", zap.NewNop(),
		WithVisionBaseURL(server.URL),
		WithVisionModel("claude-sonnet-4-20250514"),
		WithVisionTimeout(2*time.Second),
	)
	return client, server
}

func sampleRequest() llm.VisionRequest {
	return llm.VisionRequest{
		SystemPrompt: "you are a parser",
		UserPrompt:   "extract holdings",
		ImageData:    []byte{0x89, 0x50, 0x4E, 0x47}, // PNG magic bytes
		ImageMIME:    "image/png",
		MaxTokens:    1024,
		Temperature:  0.2,
	}
}

func TestVisionClient_AnalyzeImage_Success(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("expected X-API-Key=test-key, got %s", got)
		}
		if got := r.Header.Get("Anthropic-Version"); got != "2023-06-01" {
			t.Errorf("expected anthropic-version header, got %s", got)
		}

		body, _ := io.ReadAll(r.Body)
		var payload visionMessagesRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Model != "claude-sonnet-4-20250514" {
			t.Errorf("unexpected model: %s", payload.Model)
		}
		if payload.System != "you are a parser" {
			t.Errorf("unexpected system prompt: %s", payload.System)
		}
		if len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 {
			t.Fatalf("unexpected messages structure: %+v", payload.Messages)
		}
		img := payload.Messages[0].Content[0]
		if img.Type != "image" || img.Source == nil {
			t.Fatalf("expected image block, got %+v", img)
		}
		if img.Source.Type != "base64" || img.Source.MediaType != "image/png" {
			t.Errorf("unexpected image source: %+v", img.Source)
		}
		decoded, err := base64.StdEncoding.DecodeString(img.Source.Data)
		if err != nil {
			t.Fatalf("base64 decode failed: %v", err)
		}
		if string(decoded) != string([]byte{0x89, 0x50, 0x4E, 0x47}) {
			t.Errorf("image bytes mismatch after base64 round-trip")
		}
		text := payload.Messages[0].Content[1]
		if text.Type != "text" || text.Text != "extract holdings" {
			t.Errorf("unexpected text block: %+v", text)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"content":[{"type":"text","text":"{\"holdings\":[]}"}],
			"model":"claude-sonnet-4-20250514",
			"usage":{"input_tokens":123,"output_tokens":45}
		}`))
	})
	defer server.Close()

	resp, err := client.AnalyzeImage(context.Background(), sampleRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != `{"holdings":[]}` {
		t.Errorf("unexpected content: %q", resp.Content)
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
	if resp.UsageHint["input_tokens"].(int) != 123 {
		t.Errorf("expected input_tokens=123")
	}
}

func TestVisionClient_AnalyzeImage_ServerError(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"type":"api_error","message":"overloaded"}}`))
	})
	defer server.Close()

	_, err := client.AnalyzeImage(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, llm.ErrVisionServer) {
		t.Errorf("expected ErrVisionServer, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_RateLimited(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit","message":"slow down"}}`))
	})
	defer server.Close()

	_, err := client.AnalyzeImage(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, llm.ErrVisionRateLimited) {
		t.Errorf("expected ErrVisionRateLimited, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_BadJSON(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not a json`))
	})
	defer server.Close()

	_, err := client.AnalyzeImage(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, llm.ErrVisionDecode) {
		t.Errorf("expected ErrVisionDecode, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_ContextTimeout(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Block until the client context is cancelled so we exercise the
		// timeout branch deterministically.
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.AnalyzeImage(ctx, sampleRequest())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, llm.ErrVisionTimeout) {
		t.Errorf("expected ErrVisionTimeout, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_ClientError(t *testing.T) {
	client, server := newTestVisionClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request","message":"bad"}}`))
	})
	defer server.Close()

	_, err := client.AnalyzeImage(context.Background(), sampleRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, llm.ErrVisionClient) {
		t.Errorf("expected ErrVisionClient, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_InvalidRequest(t *testing.T) {
	client := NewVisionClient("k", zap.NewNop())
	_, err := client.AnalyzeImage(context.Background(), llm.VisionRequest{ImageMIME: "image/png"})
	if err == nil || !errors.Is(err, llm.ErrVisionInvalidRequest) {
		t.Errorf("expected ErrVisionInvalidRequest for empty image, got %v", err)
	}

	_, err = client.AnalyzeImage(context.Background(), llm.VisionRequest{ImageData: []byte{0x01}})
	if err == nil || !errors.Is(err, llm.ErrVisionInvalidRequest) {
		t.Errorf("expected ErrVisionInvalidRequest for empty mime, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_RejectsUnsupportedMIME(t *testing.T) {
	client := NewVisionClient("k", zap.NewNop())
	_, err := client.AnalyzeImage(context.Background(), llm.VisionRequest{
		ImageData: []byte{0x01, 0x02},
		ImageMIME: "image/bmp",
	})
	if err == nil || !errors.Is(err, llm.ErrVisionInvalidRequest) {
		t.Fatalf("expected ErrVisionInvalidRequest for image/bmp, got %v", err)
	}
}

func TestVisionClient_AnalyzeImage_RejectsOversizedPayload(t *testing.T) {
	client := NewVisionClient("k", zap.NewNop())
	huge := make([]byte, maxVisionImageBytes+1)
	_, err := client.AnalyzeImage(context.Background(), llm.VisionRequest{
		ImageData: huge,
		ImageMIME: "image/png",
	})
	if err == nil || !errors.Is(err, llm.ErrVisionInvalidRequest) {
		t.Fatalf("expected ErrVisionInvalidRequest for oversized payload, got %v", err)
	}
}

func TestRegisterVision_FallsBackToClaudeKey(t *testing.T) {
	// Verify the VISION_API_KEY → CLAUDE_API_KEY fallback documented in
	// register.go. If VisionAPIKey is empty, the registry factory must pick
	// up the text APIKey field so single-vendor deployments can authenticate
	// vision requests without duplicating the key env var.
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider:       "claude",
			ClaudeAPIKey:   "text-fallback-key",
			VisionProvider: "claude",
			VisionAPIKey:   "",
		},
	}
	provider, err := llm.NewVisionProvider(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewVisionProvider: %v", err)
	}
	vc, ok := provider.(*VisionClient)
	if !ok {
		t.Fatalf("expected *VisionClient, got %T", provider)
	}
	if vc.apiKey != "text-fallback-key" {
		t.Errorf("expected vision client to fall back to text api key, got %q", vc.apiKey)
	}
}

// Compile-time assertion that VisionClient satisfies the interface.
var _ llm.VisionProvider = (*VisionClient)(nil)
