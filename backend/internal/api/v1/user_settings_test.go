package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	usersettings "github.com/richman/backend/internal/service/user_settings"
)

// fakeSettingsRepo implements usersettings.UserRepo with an in-memory user.
type fakeSettingsRepo struct {
	user *model.User
}

func (f *fakeSettingsRepo) GetUserByID(_ context.Context, _ int64) (*model.User, error) {
	if f.user == nil {
		return nil, nil
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeSettingsRepo) GetTotalCapitalCNY(_ context.Context, _ int64) (*float64, error) {
	if f.user == nil || f.user.TotalCapitalCNY == nil {
		return nil, nil
	}
	v := *f.user.TotalCapitalCNY
	return &v, nil
}

func (f *fakeSettingsRepo) UpdateUserSettings(
	_ context.Context, _ int64, patch *repo.UserSettingsPatch,
) (*model.User, error) {
	if f.user == nil {
		return nil, nil
	}
	if patch.ClearTotalCapitalCNY {
		f.user.TotalCapitalCNY = nil
	} else if patch.TotalCapitalCNY != nil {
		v := *patch.TotalCapitalCNY
		f.user.TotalCapitalCNY = &v
	}
	if patch.RiskPreference != nil {
		f.user.RiskPreference = *patch.RiskPreference
	}
	if patch.Categories != nil {
		f.user.Categories = append([]string(nil), (*patch.Categories)...)
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeSettingsRepo) UpdateRiskPreference(_ context.Context, _ int64, _ string) error {
	return nil
}

func (f *fakeSettingsRepo) UpdateEmailPush(_ context.Context, _ int64, _ bool) error {
	return nil
}

func newSettingsTestRouter(svc *usersettings.Service, authedUserID int64) *gin.Engine {
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
	h := NewUserSettingsHandler(svc)
	h.RegisterRoutes(r.Group("/api/v1"), auth)
	return r
}

func baseSettingsUser() *model.User {
	totalCap := 50000.0
	return &model.User{
		UserID:          7,
		Email:           "u@example.com",
		Role:            "user",
		RiskPreference:  model.RiskPreferenceNeutral,
		Categories:      []string{model.AssetTypeGoldETF},
		TotalCapitalCNY: &totalCap,
	}
}

func decodeSettings(t *testing.T, body []byte) *usersettings.UserSettings {
	t.Helper()
	var envelope struct {
		Data *usersettings.UserSettings `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, string(body))
	}
	return envelope.Data
}

func TestUserSettingsAPI_Get(t *testing.T) {
	svc := usersettings.NewService(&fakeSettingsRepo{user: baseSettingsUser()})
	r := newSettingsTestRouter(svc, 7)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/settings", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeSettings(t, w.Body.Bytes())
	if got == nil || got.UserID != 7 {
		t.Fatalf("expected userID=7, got %+v", got)
	}
	if got.TotalCapitalCNY == nil || *got.TotalCapitalCNY != 50000 {
		t.Errorf("expected totalCapital=50000, got %+v", got.TotalCapitalCNY)
	}
	if got.RiskPreference != model.RiskPreferenceNeutral {
		t.Errorf("risk pref: %s", got.RiskPreference)
	}
}

func TestUserSettingsAPI_PatchClear(t *testing.T) {
	svc := usersettings.NewService(&fakeSettingsRepo{user: baseSettingsUser()})
	r := newSettingsTestRouter(svc, 7)

	body := []byte(`{"clearTotalCapitalCny":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeSettings(t, w.Body.Bytes())
	if got.TotalCapitalCNY != nil {
		t.Errorf("expected nil after clear, got %+v", got.TotalCapitalCNY)
	}
}

func TestUserSettingsAPI_PatchSetCapitalAndRisk(t *testing.T) {
	svc := usersettings.NewService(&fakeSettingsRepo{user: baseSettingsUser()})
	r := newSettingsTestRouter(svc, 7)

	body := []byte(`{"totalCapitalCny":100000,"riskPreference":"aggressive"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeSettings(t, w.Body.Bytes())
	if got.TotalCapitalCNY == nil || *got.TotalCapitalCNY != 100000 {
		t.Errorf("totalCapital: %+v", got.TotalCapitalCNY)
	}
	if got.RiskPreference != model.RiskPreferenceAggressive {
		t.Errorf("riskPreference: %s", got.RiskPreference)
	}
}

func TestUserSettingsAPI_PatchInvalidRisk(t *testing.T) {
	svc := usersettings.NewService(&fakeSettingsRepo{user: baseSettingsUser()})
	r := newSettingsTestRouter(svc, 7)

	body := []byte(`{"riskPreference":"reckless"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/user/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestUserSettingsAPI_Unauthorized(t *testing.T) {
	svc := usersettings.NewService(&fakeSettingsRepo{user: baseSettingsUser()})
	r := newSettingsTestRouter(svc, 0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/settings", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", w.Code)
	}
}
