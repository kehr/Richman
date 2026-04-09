package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// DashboardConfigRepo is the narrow LLM-config surface the dashboard
// handler needs: a single "does this user have an active row, and if so
// what is its health" lookup. Defined locally so the handler's tests can
// fake it without importing the full repo package.
type DashboardConfigRepo interface {
	GetActiveByUserID(ctx context.Context, userID int64) (*model.LLMConfig, error)
}

// DashboardCardRepo is the narrow decision-card surface the dashboard
// handler needs: a boolean "are any latest cards stuck on template/mixed"
// check powered by the migration 012 synthesis_source column.
type DashboardCardRepo interface {
	NeedsReanalysis(ctx context.Context, userID int64) (bool, error)
}

// DashboardHandler owns GET /api/v1/dashboard/summary. The summary is a
// lightweight aggregate surfaced on every dashboard mount; today only
// the llmStatus subtree is wired through because the page already
// assembles its other numbers from /decision-cards and /holdings. The
// struct intentionally leaves room for future summary fields.
type DashboardHandler struct {
	configRepo             DashboardConfigRepo
	cardRepo               DashboardCardRepo
	systemDefaultAvailable bool
	logger                 *zap.Logger
}

// NewDashboardHandler creates a new DashboardHandler. systemDefaultAvailable
// is injected from main.go because the LLM provider factory result lives
// at the composition root; the handler does not own the provider
// instance itself.
func NewDashboardHandler(
	configRepo DashboardConfigRepo,
	cardRepo DashboardCardRepo,
	systemDefaultAvailable bool,
	logger *zap.Logger,
) *DashboardHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DashboardHandler{
		configRepo:             configRepo,
		cardRepo:               cardRepo,
		systemDefaultAvailable: systemDefaultAvailable,
		logger:                 logger,
	}
}

// RegisterRoutes wires the dashboard routes under the given group. All
// routes require authentication.
func (h *DashboardHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/dashboard", authMiddleware)
	group.GET("/summary", h.GetSummary)
}

// LLMProviderHealth is the closed union for the per-user LLM health
// dimension of the llmStatus DTO. The frontend dashboard banner branches
// on this value so the wire strings are part of the API contract.
type LLMProviderHealth string

// LLMProviderHealth constants. Kept as plain string constants rather
// than a typed enum so a JSON encoder never adds a quoting quirk.
const (
	HealthStatusHealthy       LLMProviderHealth = "healthy"
	HealthStatusFailing       LLMProviderHealth = "failing"
	HealthStatusNotConfigured LLMProviderHealth = "not_configured"
)

// LLMStatusDTO mirrors the dashboard-summary.llmStatus subtree consumed
// by the frontend dashboard-summary feature. See
// frontend/src/features/dashboard-summary/types.ts for the matching
// TypeScript interface.
type LLMStatusDTO struct {
	Configured             bool              `json:"configured"`
	UserProviderHealth     LLMProviderHealth `json:"userProviderHealth"`
	SystemDefaultAvailable bool              `json:"systemDefaultAvailable"`
	NeedsReanalysis        bool              `json:"needsReanalysis"`
}

// DashboardSummaryDTO is the full GET /api/v1/dashboard/summary payload.
// Today it is just the llmStatus subtree; future summary fields slot in
// alongside it without another schema round trip.
type DashboardSummaryDTO struct {
	LLMStatus LLMStatusDTO `json:"llmStatus"`
}

// GetSummary handles GET /api/v1/dashboard/summary.
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	health := HealthStatusNotConfigured
	configured := false
	cfg, err := h.configRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, llm.ErrConfigNotFound) {
			// Log and degrade gracefully: a DB hiccup on the config read
			// should not take the whole dashboard summary down. Treat
			// the user as "not configured" and keep the llmStatus
			// accurate for the other dimensions.
			h.logger.Warn("dashboard: get llm config failed",
				zap.Int64("user_id", userID), zap.Error(err))
		}
	} else {
		configured = true
		switch cfg.HealthStatus {
		case model.HealthHealthy:
			health = HealthStatusHealthy
		case model.HealthFailing:
			health = HealthStatusFailing
		default:
			// Unknown is the post-save / pre-probe state. Treat it as
			// healthy-ish for the dashboard signal so we do not show a
			// "failing" banner before the first probe has had a chance
			// to run. Operators that care about unknown can inspect
			// /settings/llm directly.
			health = HealthStatusHealthy
		}
	}

	// configured in the response is the OR of "user has a config" and
	// "system default is reachable" — the banner disappears entirely
	// when neither layer is usable and the frontend renders the
	// degraded empty state instead.
	llmConfigured := configured || h.systemDefaultAvailable

	needs := false
	if n, nErr := h.cardRepo.NeedsReanalysis(ctx, userID); nErr != nil {
		h.logger.Warn("dashboard: needs reanalysis query failed",
			zap.Int64("user_id", userID), zap.Error(nErr))
	} else {
		needs = n
	}

	c.JSON(http.StatusOK, gin.H{
		"data": DashboardSummaryDTO{
			LLMStatus: LLMStatusDTO{
				Configured:             llmConfigured,
				UserProviderHealth:     health,
				SystemDefaultAvailable: h.systemDefaultAvailable,
				NeedsReanalysis:        needs,
			},
		},
	})
}
