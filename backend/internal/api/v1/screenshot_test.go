package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/service/screenshot"
	"go.uber.org/zap"
)

type stubVision struct {
	content string
	err     error
}

func (s *stubVision) AnalyzeImage(_ context.Context, _ llm.VisionRequest) (*llm.VisionResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &llm.VisionResponse{Content: s.content, Model: "stub"}, nil
}
func (s *stubVision) Name() string { return "stub" }

const okPayload = `{"holdings":[{"assetName":{"value":"AAPL","confidence":0.95},"assetCode":{"value":"AAPL","confidence":0.95},"costPrice":{"value":"150","confidence":0.9},"positionPct":{"value":"25","confidence":0.9},"assetTypeGuess":"us_stock"}]}`

func newTestRouter(svc *screenshot.Service, authedUserID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Minimal auth stub: if authedUserID > 0 inject it into context,
	// otherwise respond with 401 to simulate the real middleware.
	auth := func(c *gin.Context) {
		if authedUserID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "auth required"},
			})
			return
		}
		c.Set(middleware.ContextKeyUserID, authedUserID)
		c.Next()
	}
	h := NewScreenshotHandler(svc)
	h.RegisterRoutes(r.Group("/api/v1"), auth)
	return r
}

func buildMultipart(t *testing.T, field, filename, mime string, body []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + field + `"; filename="` + filename + `"`}
	if mime != "" {
		h["Content-Type"] = []string{mime}
	}
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}

func TestScreenshotHandler_HappyPath(t *testing.T) {
	svc := screenshot.NewService(&stubVision{content: okPayload}, zap.NewNop(), screenshot.Options{})
	r := newTestRouter(svc, 42)

	body, ct := buildMultipart(t, "file", "portfolio.png", "image/png", []byte("fake-png-bytes"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", body)
	req.Header.Set("Content-Type", ct)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data screenshot.RecognizeResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.OverallStatus != screenshot.StatusOK {
		t.Errorf("want ok, got %q", resp.Data.OverallStatus)
	}
	if len(resp.Data.Holdings) != 1 {
		t.Errorf("want 1 holding, got %d", len(resp.Data.Holdings))
	}
}

func TestScreenshotHandler_Unauthenticated(t *testing.T) {
	svc := screenshot.NewService(&stubVision{content: okPayload}, zap.NewNop(), screenshot.Options{})
	r := newTestRouter(svc, 0)

	body, ct := buildMultipart(t, "file", "portfolio.png", "image/png", []byte("fake"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", body)
	req.Header.Set("Content-Type", ct)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestScreenshotHandler_MissingFile(t *testing.T) {
	svc := screenshot.NewService(&stubVision{content: okPayload}, zap.NewNop(), screenshot.Options{})
	r := newTestRouter(svc, 42)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("note", "no file here")
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestScreenshotHandler_WrongContentType(t *testing.T) {
	svc := screenshot.NewService(&stubVision{content: okPayload}, zap.NewNop(), screenshot.Options{})
	r := newTestRouter(svc, 42)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", bytes.NewReader([]byte(`{"x":1}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestScreenshotHandler_RateLimited(t *testing.T) {
	svc := screenshot.NewService(
		&stubVision{content: okPayload},
		zap.NewNop(),
		screenshot.Options{RateLimit: 1},
	)
	r := newTestRouter(svc, 42)

	doCall := func() *httptest.ResponseRecorder {
		body, ct := buildMultipart(t, "file", "a.png", "image/png", []byte("x"))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", body)
		req.Header.Set("Content-Type", ct)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	if w := doCall(); w.Code != http.StatusOK {
		t.Fatalf("first call status = %d", w.Code)
	}
	if w := doCall(); w.Code != http.StatusTooManyRequests {
		t.Fatalf("second call status = %d, body=%s", w.Code, w.Body.String())
	}
}

func TestScreenshotHandler_VisionDegrades(t *testing.T) {
	svc := screenshot.NewService(
		&stubVision{err: llm.ErrVisionTimeout},
		zap.NewNop(),
		screenshot.Options{},
	)
	r := newTestRouter(svc, 42)

	body, ct := buildMultipart(t, "file", "a.png", "image/png", []byte("x"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portfolio/import-screenshot", body)
	req.Header.Set("Content-Type", ct)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data screenshot.RecognizeResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.OverallStatus != screenshot.StatusFailed {
		t.Errorf("want failed, got %q", resp.Data.OverallStatus)
	}
	if resp.Data.Warning == "" {
		t.Error("expected warning to be populated")
	}
}
