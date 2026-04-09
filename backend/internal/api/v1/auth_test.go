package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newTestContext spins up a gin test context with a recorder, a request, and
// an optional request-scoped logger. Returns the context, recorder, and the
// log observer for inspection.
func newTestContext(
	t *testing.T, attachLogger bool,
) (*gin.Context, *httptest.ResponseRecorder, *observer.ObservedLogs) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zapcore.ErrorLevel)
	reqLogger := zap.New(core).With(zap.String("requestId", "req-test-123"))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", http.NoBody)
	if attachLogger {
		c.Set("logger", reqLogger)
	}
	return c, rec, logs
}

func TestHandleServiceError_AppError_DoesNotLog(t *testing.T) {
	// Replace global logger with an observed one so we can assert the
	// AppError branch does NOT emit any log via the fallback path either.
	core, logs := observer.New(zapcore.ErrorLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	c, rec, _ := newTestContext(t, false)
	appErr := model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")

	handleServiceError(c, appErr)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"]["code"] != "UNAUTHORIZED" {
		t.Errorf("want code UNAUTHORIZED, got %q", body["error"]["code"])
	}
	if body["error"]["message"] != "invalid email or password" {
		t.Errorf("want original message, got %q", body["error"]["message"])
	}
	if logs.Len() != 0 {
		t.Errorf("AppError branch must not emit ERROR logs, got %d entries", logs.Len())
	}
}

func TestHandleServiceError_NonAppError_LogsWithRequestLogger(t *testing.T) {
	c, rec, logs := newTestContext(t, true)
	underlying := errors.New(`column "onboarding_skipped_at" does not exist`)
	wrapped := fmt.Errorf("find user: %w", underlying)

	handleServiceError(c, wrapped)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want status 500, got %d", rec.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"]["code"] != "INTERNAL_ERROR" {
		t.Errorf("want code INTERNAL_ERROR, got %q", body["error"]["code"])
	}

	if logs.Len() != 1 {
		t.Fatalf("want 1 ERROR log, got %d", logs.Len())
	}
	entry := logs.All()[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Errorf("want ERROR level, got %s", entry.Level)
	}
	if entry.Message != "unhandled service error" {
		t.Errorf("unexpected log message: %q", entry.Message)
	}

	fields := entry.ContextMap()
	if fields["requestId"] != "req-test-123" {
		t.Errorf("want requestId from request-scoped logger, got %v", fields["requestId"])
	}
	if fields["path"] != "/api/v1/auth/login" {
		t.Errorf("want path field, got %v", fields["path"])
	}
	if fields["method"] != http.MethodPost {
		t.Errorf("want method field, got %v", fields["method"])
	}
	// zap.Error writes the full wrapped chain into the "error" field.
	errField, ok := fields["error"].(string)
	if !ok {
		t.Fatalf("want error field as string, got %T", fields["error"])
	}
	if errField != wrapped.Error() {
		t.Errorf("want full wrapped error chain %q, got %q", wrapped.Error(), errField)
	}
}

func TestHandleServiceError_NonAppError_FallsBackToGlobalLogger(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	c, rec, _ := newTestContext(t, false)
	handleServiceError(c, errors.New("pool exhausted"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want status 500, got %d", rec.Code)
	}
	if logs.Len() != 1 {
		t.Fatalf("want 1 ERROR log on global logger, got %d", logs.Len())
	}
	entry := logs.All()[0]
	if entry.Message != "unhandled service error" {
		t.Errorf("unexpected log message: %q", entry.Message)
	}
}

func TestHandleServiceError_NilError_NoOp(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	c, rec, _ := newTestContext(t, false)
	handleServiceError(c, nil)

	if rec.Code != http.StatusOK {
		t.Errorf("nil err should not write response, got status %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("nil err should not write body, got %q", rec.Body.String())
	}
	if logs.Len() != 0 {
		t.Errorf("nil err must not emit logs, got %d entries", logs.Len())
	}
}
