package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// fakeDashboardConfigRepo implements DashboardConfigRepo with an
// in-memory single-row store. The handler only calls GetActiveByUserID
// so the other repo methods do not need stubs.
type fakeDashboardConfigRepo struct {
	cfg *model.LLMConfig
	err error
}

func (f *fakeDashboardConfigRepo) GetActiveByUserID(
	_ context.Context, _ int64,
) (*model.LLMConfig, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.cfg == nil {
		return nil, llm.ErrConfigNotFound
	}
	cp := *f.cfg
	return &cp, nil
}

type fakeDashboardCardRepo struct {
	needs bool
}

func (f *fakeDashboardCardRepo) NeedsReanalysis(
	_ context.Context, _ int64,
) (bool, error) {
	return f.needs, nil
}

func newDashboardRouter(h *DashboardHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, int64(7))
		c.Next()
	}
	h.RegisterRoutes(r.Group("/api/v1"), auth)
	return r
}

func decodeDashboardSummary(t *testing.T, body []byte) DashboardSummaryDTO {
	t.Helper()
	var envelope struct {
		Data DashboardSummaryDTO `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return envelope.Data
}

func TestDashboardSummary_NotConfiguredNoSystemDefault(t *testing.T) {
	h := NewDashboardHandler(
		&fakeDashboardConfigRepo{},
		&fakeDashboardCardRepo{needs: false},
		false,
		zap.NewNop(),
	)
	r := newDashboardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	got := decodeDashboardSummary(t, w.Body.Bytes())
	if got.LLMStatus.Configured {
		t.Error("expected configured=false when no user config and no system default")
	}
	if got.LLMStatus.UserProviderHealth != HealthStatusNotConfigured {
		t.Errorf("want not_configured, got %s", got.LLMStatus.UserProviderHealth)
	}
}

func TestDashboardSummary_SystemDefaultOnly(t *testing.T) {
	h := NewDashboardHandler(
		&fakeDashboardConfigRepo{},
		&fakeDashboardCardRepo{needs: false},
		true,
		zap.NewNop(),
	)
	r := newDashboardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeDashboardSummary(t, w.Body.Bytes())
	if !got.LLMStatus.Configured {
		t.Error("expected configured=true when system default is available")
	}
	if got.LLMStatus.UserProviderHealth != HealthStatusNotConfigured {
		t.Errorf("user provider still not configured, got %s", got.LLMStatus.UserProviderHealth)
	}
	if !got.LLMStatus.SystemDefaultAvailable {
		t.Error("expected systemDefaultAvailable=true")
	}
}

func TestDashboardSummary_UserHealthy(t *testing.T) {
	cfg := &model.LLMConfig{
		ConfigID:     1,
		UserID:       7,
		ProviderType: model.ProviderClaude,
		HealthStatus: model.HealthHealthy,
	}
	h := NewDashboardHandler(
		&fakeDashboardConfigRepo{cfg: cfg},
		&fakeDashboardCardRepo{needs: false},
		true,
		zap.NewNop(),
	)
	r := newDashboardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeDashboardSummary(t, w.Body.Bytes())
	if got.LLMStatus.UserProviderHealth != HealthStatusHealthy {
		t.Errorf("want healthy, got %s", got.LLMStatus.UserProviderHealth)
	}
}

func TestDashboardSummary_UserFailingNeedsReanalysis(t *testing.T) {
	cfg := &model.LLMConfig{
		ConfigID:     1,
		UserID:       7,
		ProviderType: model.ProviderClaude,
		HealthStatus: model.HealthFailing,
	}
	h := NewDashboardHandler(
		&fakeDashboardConfigRepo{cfg: cfg},
		&fakeDashboardCardRepo{needs: true},
		true,
		zap.NewNop(),
	)
	r := newDashboardRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := decodeDashboardSummary(t, w.Body.Bytes())
	if got.LLMStatus.UserProviderHealth != HealthStatusFailing {
		t.Errorf("want failing, got %s", got.LLMStatus.UserProviderHealth)
	}
	if !got.LLMStatus.NeedsReanalysis {
		t.Error("expected needsReanalysis=true when repo reports template/mixed cards")
	}
}
