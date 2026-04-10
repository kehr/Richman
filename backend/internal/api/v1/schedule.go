package v1

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	scheduleSvc "github.com/richman/backend/internal/service/schedule"
	"go.uber.org/zap"
)

// ScheduleReloader is the subset of the Scheduler that the schedule handler
// needs to notify after a settings change. Step 4 (scheduler rewrite) will
// implement this on the concrete *analysis.Scheduler; wiring an interface here
// breaks the import cycle and keeps this file compilable while Step 4 is
// still in progress.
type ScheduleReloader interface {
	ReloadUser(ctx context.Context, userID int64) error
}

// HoldingReader is the minimal repo interface needed to verify that a holding
// belongs to the requesting user before processing schedule overrides.
type HoldingReader interface {
	GetHoldingByID(ctx context.Context, holdingID int64) (*model.Holding, error)
}

// CardReader is the minimal repo interface needed to fetch the last analyzed
// time for a holding, which is used by ComputeNextAnalysisAt.
type CardReader interface {
	GetLatestByHolding(ctx context.Context, holdingID int64) (*model.DecisionCard, error)
}

// ScheduleHandler handles schedule settings and per-holding override endpoints.
type ScheduleHandler struct {
	svc         *scheduleSvc.Service
	holdingRepo HoldingReader
	cardRepo    CardReader
	reloader    ScheduleReloader // may be nil when scheduler rewrite is not yet wired
	logger      *zap.Logger
}

// NewScheduleHandler constructs a ScheduleHandler. reloader may be nil; in
// that case settings updates still persist but the in-memory cron is not
// reloaded until the next server restart.
func NewScheduleHandler(
	svc *scheduleSvc.Service,
	holdingRepo HoldingReader,
	cardRepo CardReader,
	reloader ScheduleReloader,
	logger *zap.Logger,
) *ScheduleHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ScheduleHandler{
		svc:         svc,
		holdingRepo: holdingRepo,
		cardRepo:    cardRepo,
		reloader:    reloader,
		logger:      logger,
	}
}

// RegisterRoutes wires schedule routes under the given API group.
// All routes require authentication.
func (h *ScheduleHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	settings := rg.Group("/settings/schedule", authMiddleware)
	settings.GET("", h.GetSettings)
	settings.PUT("", h.UpdateSettings)

	holdings := rg.Group("/holdings", authMiddleware)
	holdings.GET("/:id/schedule", h.GetHoldingSchedule)
	holdings.PUT("/:id/schedule", h.UpdateHoldingSchedule)
}

// scheduleSettingsRequest is the JSON body for PUT /api/v1/settings/schedule.
// The shape mirrors model.UpsertScheduleSettingsInput but uses the API-level
// market sub-object layout the frontend sends.
type scheduleSettingsRequest struct {
	GlobalFrequency     string  `json:"globalFrequency"`
	GlobalFrequencyDays *int32  `json:"globalFrequencyDays"`
	Markets             markets `json:"markets"`
}

type markets struct {
	AShare  marketSettings `json:"a_share"`
	USStock marketSettings `json:"us_stock"`
}

type marketSettings struct {
	Frequency     *string      `json:"frequency"`
	FrequencyDays *int32       `json:"frequencyDays"`
	PreWindow     windowFields `json:"preWindow"`
	PostWindow    windowFields `json:"postWindow"`
}

type windowFields struct {
	Enabled  bool   `json:"enabled"`
	Time     string `json:"time"`
	IsCustom bool   `json:"isCustom"`
}

// scheduleSettingsResponse is the JSON shape for GET and PUT responses.
// It reuses the same market sub-structs as the request because the shapes
// are symmetric (confirmed by TRD).
type scheduleSettingsResponse struct {
	GlobalFrequency     string  `json:"globalFrequency"`
	GlobalFrequencyDays *int32  `json:"globalFrequencyDays"`
	Markets             markets `json:"markets"`
}

// toSettingsResponse converts the service/model type to the API response shape.
func toSettingsResponse(s *model.UserScheduleSettings) scheduleSettingsResponse {
	return scheduleSettingsResponse{
		GlobalFrequency:     s.GlobalFrequency,
		GlobalFrequencyDays: s.GlobalFrequencyDays,
		Markets: markets{
			AShare: marketSettings{
				Frequency:     s.AShareFrequency,
				FrequencyDays: s.AShareFrequencyDays,
				PreWindow: windowFields{
					Enabled:  s.ASharePreEnabled,
					Time:     s.ASharePreTime,
					IsCustom: s.ASharePreCustom,
				},
				PostWindow: windowFields{
					Enabled:  s.ASharePostEnabled,
					Time:     s.ASharePostTime,
					IsCustom: s.ASharePostCustom,
				},
			},
			USStock: marketSettings{
				Frequency:     s.USFrequency,
				FrequencyDays: s.USFrequencyDays,
				PreWindow: windowFields{
					Enabled:  s.USPreEnabled,
					Time:     s.USPreTime,
					IsCustom: s.USPreCustom,
				},
				PostWindow: windowFields{
					Enabled:  s.USPostEnabled,
					Time:     s.USPostTime,
					IsCustom: s.USPostCustom,
				},
			},
		},
	}
}

// GetSettings handles GET /api/v1/settings/schedule.
func (h *ScheduleHandler) GetSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	settings, err := h.svc.GetUserScheduleSettings(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toSettingsResponse(settings)})
}

// UpdateSettings handles PUT /api/v1/settings/schedule.
func (h *ScheduleHandler) UpdateSettings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	var req scheduleSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	input := &model.UpsertScheduleSettingsInput{
		GlobalFrequency:     req.GlobalFrequency,
		GlobalFrequencyDays: req.GlobalFrequencyDays,

		ASharePreEnabled:    req.Markets.AShare.PreWindow.Enabled,
		ASharePreTime:       req.Markets.AShare.PreWindow.Time,
		ASharePreCustom:     req.Markets.AShare.PreWindow.IsCustom,
		ASharePostEnabled:   req.Markets.AShare.PostWindow.Enabled,
		ASharePostTime:      req.Markets.AShare.PostWindow.Time,
		ASharePostCustom:    req.Markets.AShare.PostWindow.IsCustom,
		AShareFrequency:     req.Markets.AShare.Frequency,
		AShareFrequencyDays: req.Markets.AShare.FrequencyDays,

		USPreEnabled:    req.Markets.USStock.PreWindow.Enabled,
		USPreTime:       req.Markets.USStock.PreWindow.Time,
		USPreCustom:     req.Markets.USStock.PreWindow.IsCustom,
		USPostEnabled:   req.Markets.USStock.PostWindow.Enabled,
		USPostTime:      req.Markets.USStock.PostWindow.Time,
		USPostCustom:    req.Markets.USStock.PostWindow.IsCustom,
		USFrequency:     req.Markets.USStock.Frequency,
		USFrequencyDays: req.Markets.USStock.FrequencyDays,

		Modifier: email,
	}

	settings, err := h.svc.UpsertUserScheduleSettings(c.Request.Context(), userID, input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Notify the scheduler to reload cron entries for this user. A nil reloader
	// or a reload error is non-fatal: the persisted settings are authoritative
	// and will be picked up on the next server restart or scheduled reload.
	if h.reloader != nil {
		if reloadErr := h.reloader.ReloadUser(c.Request.Context(), userID); reloadErr != nil {
			// Non-fatal: persisted settings are authoritative; the cron will
			// pick them up on the next server restart or scheduled reload.
			h.logger.Warn("schedule reload failed after settings update",
				zap.Int64("user_id", userID),
				zap.Error(reloadErr),
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": toSettingsResponse(settings)})
}

// holdingScheduleResponse is the JSON body for GET/PUT /api/v1/holdings/:id/schedule.
type holdingScheduleResponse struct {
	HoldingID      int64      `json:"holdingId"`
	Frequency      *string    `json:"frequency"`
	FrequencyDays  *int32     `json:"frequencyDays"`
	Window         *string    `json:"window"`
	NextAnalysisAt *time.Time `json:"nextAnalysisAt"`
}

// holdingScheduleRequest is the JSON body for PUT /api/v1/holdings/:id/schedule.
type holdingScheduleRequest struct {
	Frequency     *string `json:"frequency"`
	FrequencyDays *int32  `json:"frequencyDays"`
	Window        *string `json:"window"`
}

// assetTypeToMarket maps a holding's AssetType to the schedule service market
// constant. Returns "a_share" as the default so unknown types do not break.
func assetTypeToMarket(assetType string) string {
	switch assetType {
	case "us_stock":
		return scheduleSvc.MarketUSStock
	default:
		return scheduleSvc.MarketAShare
	}
}

// shanghaiNow returns the current time in Asia/Shanghai. On zone-load failure
// (should not happen in practice) it returns time.Now() in local time.
func shanghaiNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}

// buildHoldingScheduleResponse assembles the response for both GET and PUT.
// It computes nextAnalysisAt using the last decision card for the holding.
func (h *ScheduleHandler) buildHoldingScheduleResponse(
	ctx context.Context,
	holdingID int64,
	userID int64,
	market string,
	override *model.HoldingScheduleOverride,
) holdingScheduleResponse {
	resp := holdingScheduleResponse{
		HoldingID: holdingID,
	}
	if override != nil {
		resp.Frequency = override.Frequency
		resp.FrequencyDays = override.FrequencyDays
		resp.Window = override.Window
	}

	// Fetch lastAnalyzedAt from the most recent decision card. A missing card
	// (first analysis) or any fetch error is treated as nil — the service will
	// return the next upcoming window without a minimum-interval constraint.
	var lastAnalyzedAt *time.Time
	card, cardErr := h.cardRepo.GetLatestByHolding(ctx, holdingID)
	if cardErr == nil && card != nil {
		lastAnalyzedAt = &card.AnalyzedAt
	}

	now := shanghaiNow()
	next, err := h.svc.ComputeNextAnalysisAt(ctx, userID, holdingID, market, lastAnalyzedAt, now)
	if err == nil {
		resp.NextAnalysisAt = &next
	}

	return resp
}

// GetHoldingSchedule handles GET /api/v1/holdings/:id/schedule.
func (h *ScheduleHandler) GetHoldingSchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid holding id"},
		})
		return
	}

	// Verify the holding belongs to the requesting user.
	holding, err := h.holdingRepo.GetHoldingByID(c.Request.Context(), holdingID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if holding == nil || holding.UserID != userID {
		handleServiceError(c, model.ErrNotFound)
		return
	}

	override, err := h.svc.GetHoldingScheduleOverride(c.Request.Context(), userID, holdingID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	market := assetTypeToMarket(holding.AssetType)
	resp := h.buildHoldingScheduleResponse(c.Request.Context(), holdingID, userID, market, override)
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// UpdateHoldingSchedule handles PUT /api/v1/holdings/:id/schedule.
func (h *ScheduleHandler) UpdateHoldingSchedule(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid holding id"},
		})
		return
	}

	// Verify ownership before accepting any body.
	holding, err := h.holdingRepo.GetHoldingByID(c.Request.Context(), holdingID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if holding == nil || holding.UserID != userID {
		handleServiceError(c, model.ErrNotFound)
		return
	}

	var req holdingScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	input := &model.UpsertHoldingScheduleOverrideInput{
		Frequency:     req.Frequency,
		FrequencyDays: req.FrequencyDays,
		Window:        req.Window,
		Modifier:      email,
	}

	override, err := h.svc.UpsertHoldingScheduleOverride(c.Request.Context(), userID, holdingID, input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	market := assetTypeToMarket(holding.AssetType)
	resp := h.buildHoldingScheduleResponse(c.Request.Context(), holdingID, userID, market, override)
	c.JSON(http.StatusOK, gin.H{"data": resp})
}
