package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/service/onboarding"
)

// fakeOnbUserRepo is the minimum UserRepo the onboarding service needs.
type fakeOnbUserRepo struct {
	user *model.User
}

func (f *fakeOnbUserRepo) GetUserByID(_ context.Context, _ int64) (*model.User, error) {
	if f.user == nil {
		return nil, nil
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeOnbUserRepo) MarkOnboardingCompleted(_ context.Context, _ int64) (*model.User, error) {
	if f.user == nil {
		return nil, nil
	}
	if f.user.OnboardingCompletedAt == nil {
		now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
		f.user.OnboardingCompletedAt = &now
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeOnbUserRepo) ResetOnboarding(_ context.Context, _ int64) (*model.User, error) {
	if f.user == nil {
		return nil, nil
	}
	f.user.OnboardingCompletedAt = nil
	f.user.OnboardingSkippedAt = nil
	cp := *f.user
	return &cp, nil
}

type stubEnv struct{ prod bool }

func (s stubEnv) IsProduction() bool { return s.prod }

func newOnboardingTestRouter(svc *onboarding.Service, authedUserID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
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
	h := NewOnboardingHandler(svc)
	h.RegisterRoutes(r.Group("/api/v1"), auth)
	return r
}

func baseOnbUser() *model.User {
	return &model.User{
		UserID:         42,
		Email:          "alice@example.com",
		Role:           "user",
		RiskPreference: model.RiskPreferenceNeutral,
		Categories:     []string{},
	}
}

func decodeStatus(t *testing.T, body []byte) *onboarding.Status {
	t.Helper()
	var envelope struct {
		Data *onboarding.Status `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, string(body))
	}
	return envelope.Data
}

func TestOnboardingAPI_GetStatusDefault(t *testing.T) {
	svc := onboarding.NewService(&fakeOnbUserRepo{user: baseOnbUser()}, stubEnv{prod: false})
	r := newOnboardingTestRouter(svc, 42)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeStatus(t, w.Body.Bytes())
	if got == nil || got.Completed {
		t.Errorf("expected incomplete status, got %+v", got)
	}
}

func TestOnboardingAPI_Unauthorized(t *testing.T) {
	svc := onboarding.NewService(&fakeOnbUserRepo{user: baseOnbUser()}, stubEnv{prod: false})
	r := newOnboardingTestRouter(svc, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", w.Code)
	}
}

func TestOnboardingAPI_MarkCompletedThenGet(t *testing.T) {
	repo := &fakeOnbUserRepo{user: baseOnbUser()}
	svc := onboarding.NewService(repo, stubEnv{prod: false})
	r := newOnboardingTestRouter(svc, 42)

	// POST complete
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/complete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("complete status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	st := decodeStatus(t, w.Body.Bytes())
	if !st.Completed || st.CompletedAt == nil {
		t.Fatalf("expected completed with timestamp, got %+v", st)
	}

	// GET should now reflect completion.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	st2 := decodeStatus(t, w2.Body.Bytes())
	if !st2.Completed {
		t.Errorf("expected GetStatus to reflect completion, got %+v", st2)
	}
}

func TestOnboardingAPI_ResetDev(t *testing.T) {
	u := baseOnbUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	svc := onboarding.NewService(&fakeOnbUserRepo{user: u}, stubEnv{prod: false})
	r := newOnboardingTestRouter(svc, 42)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	st := decodeStatus(t, w.Body.Bytes())
	if st.Completed {
		t.Errorf("expected reset to leave Completed=false, got %+v", st)
	}
}

func TestOnboardingAPI_ResetForbiddenInProduction(t *testing.T) {
	u := baseOnbUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	svc := onboarding.NewService(&fakeOnbUserRepo{user: u}, stubEnv{prod: true})
	r := newOnboardingTestRouter(svc, 42)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if envelope.Error.Code != "ONBOARDING_RESET_FORBIDDEN" {
		t.Errorf("code: want ONBOARDING_RESET_FORBIDDEN, got %q", envelope.Error.Code)
	}
}

func TestOnboardingAPI_GetStatusNotFound(t *testing.T) {
	svc := onboarding.NewService(&fakeOnbUserRepo{}, stubEnv{prod: false})
	r := newOnboardingTestRouter(svc, 42)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d body=%s", w.Code, w.Body.String())
	}
}
