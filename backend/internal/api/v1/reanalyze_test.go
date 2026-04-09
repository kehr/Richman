package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
)

// rateLimitRouter builds a minimal gin engine that wires the per-user
// rate limit middleware to a stub handler. Using the middleware directly
// (rather than AnalysisHandler) keeps this test hermetic from the
// analysis pipeline while still exercising the RATE_LIMITED contract.
func rateLimitRouter(authedUserID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, authedUserID)
		c.Next()
	}
	r.POST("/api/v1/analysis/reanalyze-all",
		auth,
		middleware.PerUserRateLimit(reanalyzeAllWindow),
		func(c *gin.Context) {
			c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"taskId": "t1"}})
		},
	)
	return r
}

func TestReanalyzeAll_RateLimited(t *testing.T) {
	r := rateLimitRouter(42)

	// First call: allowed.
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/reanalyze-all", http.NoBody)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusAccepted {
		t.Fatalf("first call: want 202, got %d body=%s", w1.Code, w1.Body.String())
	}

	// Second call within the window: blocked.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/reanalyze-all", http.NoBody)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("second call: want 429, got %d body=%s", w2.Code, w2.Body.String())
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429")
	}
}

func TestReanalyzeAll_SeparateUsersIndependent(t *testing.T) {
	// Rate limit is per-user, not global. User A's hit must not block B.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authAs := func(userID int64) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set(middleware.ContextKeyUserID, userID)
			c.Next()
		}
	}
	limiter := middleware.PerUserRateLimit(reanalyzeAllWindow)
	stub := func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"taskId": "t1"}})
	}
	r.POST("/api/v1/a", authAs(1), limiter, stub)
	r.POST("/api/v1/b", authAs(2), limiter, stub)

	// User 1 hits the endpoint twice via path /a.
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/api/v1/a", http.NoBody))
	if w1.Code != http.StatusAccepted {
		t.Fatalf("a1: want 202, got %d", w1.Code)
	}
	// User 2 must not be affected by user 1's hit.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/api/v1/b", http.NoBody))
	if w2.Code != http.StatusAccepted {
		t.Fatalf("b1 (different user): want 202, got %d", w2.Code)
	}
}
