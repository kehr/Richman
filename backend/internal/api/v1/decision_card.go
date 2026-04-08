package v1

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/analysis/recommendation"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	decisioncard "github.com/richman/backend/internal/service/decision_card"
	usersettings "github.com/richman/backend/internal/service/user_settings"
)

// CapitalProvider returns the user's optional total capital (CNY) used for
// projecting *Pct fields onto absolute *Amount fields at the API layer
// (TRD §5.3). It is intentionally narrow so the handler does not depend on
// the full auth or user_settings service.
type CapitalProvider interface {
	GetTotalCapitalCNY(ctx context.Context, userID int64) (*float64, error)
}

// DecisionCardHandler handles decision card query requests.
type DecisionCardHandler struct {
	cardSvc *decisioncard.Service
	capital CapitalProvider
}

// NewDecisionCardHandler creates a new DecisionCardHandler. The capital
// provider may be nil; in that case Amount fields are never populated.
func NewDecisionCardHandler(cardSvc *decisioncard.Service, capital CapitalProvider) *DecisionCardHandler {
	return &DecisionCardHandler{cardSvc: cardSvc, capital: capital}
}

// RegisterRoutes registers decision card routes on the given router group.
func (h *DecisionCardHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/decision-cards", authMiddleware)
	group.GET("", h.ListLatest)
	group.GET("/history", h.ListHistory)
	group.GET("/:id", h.GetByID)
}

// DecisionCardDTO is the API response shape for a decision card. It mirrors
// model.DecisionCard but adds the optional *Amount projection fields and
// keeps the structured recommendation/badge fields exposed (TRD §2.5).
type DecisionCardDTO struct {
	CardID            int64     `json:"cardId"`
	UserID            int64     `json:"userId"`
	HoldingID         int64     `json:"holdingId"`
	AssetCode         string    `json:"assetCode"`
	AssetName         string    `json:"assetName"`
	AssetType         string    `json:"assetType"`
	CostPrice         float64   `json:"costPrice"`
	PositionRatio     float64   `json:"positionRatio"`
	PositionAmount    *float64  `json:"positionAmount,omitempty"`
	TrendDirection    string    `json:"trendDirection"`
	TrendSummary      string    `json:"trendSummary"`
	PositionDirection string    `json:"positionDirection"`
	PositionSummary   string    `json:"positionSummary"`
	CatalystDirection string    `json:"catalystDirection"`
	CatalystSummary   string    `json:"catalystSummary"`
	Confidence        float64   `json:"confidence"`
	ActionAdvice      string    `json:"actionAdvice"`
	DetailedAdvice    string    `json:"detailedAdvice"`
	RiskWarnings      []string  `json:"riskWarnings"`
	TodayHighlights   string    `json:"todayHighlights"`
	WeightTrend       float64   `json:"weightTrend"`
	WeightPosition    float64   `json:"weightPosition"`
	WeightCatalyst    float64   `json:"weightCatalyst"`
	AnalyzedAt        time.Time `json:"analyzedAt"`
	CreatedAt         time.Time `json:"createdAt"`

	// Structured recommendation and badge diff fields (migration 006).
	Recommendation       recommendation.Recommendation `json:"recommendation"`
	ActionLevel          int                           `json:"actionLevel"`
	TargetPositionRatio  float64                       `json:"targetPositionRatio"`
	TargetPositionAmount *float64                      `json:"targetPositionAmount,omitempty"`
	BadgeState           string                        `json:"badgeState"`
	ConfidenceDelta      float64                       `json:"confidenceDelta"`
	PrevCardID           *int64                        `json:"prevCardId,omitempty"`
	ExecutionFingerprint string                        `json:"executionFingerprint"`
}

// toDecisionCardDTO projects a model.DecisionCard onto the API response DTO
// without populating any *Amount fields. Call user_settings.AttachAmounts
// after this projection to fill in amount fields when total capital is set.
func toDecisionCardDTO(c *model.DecisionCard) DecisionCardDTO {
	return DecisionCardDTO{
		CardID:               c.CardID,
		UserID:               c.UserID,
		HoldingID:            c.HoldingID,
		AssetCode:            c.AssetCode,
		AssetName:            c.AssetName,
		AssetType:            c.AssetType,
		CostPrice:            c.CostPrice,
		PositionRatio:        c.PositionRatio,
		TrendDirection:       c.TrendDirection,
		TrendSummary:         c.TrendSummary,
		PositionDirection:    c.PositionDirection,
		PositionSummary:      c.PositionSummary,
		CatalystDirection:    c.CatalystDirection,
		CatalystSummary:      c.CatalystSummary,
		Confidence:           c.Confidence,
		ActionAdvice:         c.ActionAdvice,
		DetailedAdvice:       c.DetailedAdvice,
		RiskWarnings:         c.RiskWarnings,
		TodayHighlights:      c.TodayHighlights,
		WeightTrend:          c.WeightTrend,
		WeightPosition:       c.WeightPosition,
		WeightCatalyst:       c.WeightCatalyst,
		AnalyzedAt:           c.AnalyzedAt,
		CreatedAt:            c.CreatedAt,
		Recommendation:       c.Recommendation,
		ActionLevel:          c.ActionLevel,
		TargetPositionRatio:  c.TargetPositionRatio,
		BadgeState:           c.BadgeState,
		ConfidenceDelta:      c.ConfidenceDelta,
		PrevCardID:           c.PrevCardID,
		ExecutionFingerprint: c.ExecutionFingerprint,
	}
}

// resolveCapital fetches the optional total capital for a user. Errors are
// swallowed (logged via the higher-level access log) so that a transient
// failure to read user profile does not break decision card delivery.
func (h *DecisionCardHandler) resolveCapital(ctx context.Context, userID int64) *float64 {
	if h.capital == nil {
		return nil
	}
	totalCap, err := h.capital.GetTotalCapitalCNY(ctx, userID)
	if err != nil {
		return nil
	}
	return totalCap
}

// ListLatest handles GET /api/v1/decision-cards.
func (h *DecisionCardHandler) ListLatest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	cards, err := h.cardSvc.ListLatest(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	capital := h.resolveCapital(c.Request.Context(), userID)
	dtos := make([]DecisionCardDTO, len(cards))
	for i := range cards {
		dtos[i] = toDecisionCardDTO(&cards[i])
		usersettings.AttachAmounts(&dtos[i], capital)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dtos,
	})
}

// GetByID handles GET /api/v1/decision-cards/:id.
func (h *DecisionCardHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	cardID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid card id",
			},
		})
		return
	}

	card, err := h.cardSvc.GetByID(c.Request.Context(), userID, cardID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	dto := toDecisionCardDTO(card)
	usersettings.AttachAmounts(&dto, h.resolveCapital(c.Request.Context(), userID))

	c.JSON(http.StatusOK, gin.H{
		"data": dto,
	})
}

// ListHistory handles GET /api/v1/decision-cards/history.
func (h *DecisionCardHandler) ListHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit := 20
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	cards, err := h.cardSvc.ListHistory(c.Request.Context(), userID, limit)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	capital := h.resolveCapital(c.Request.Context(), userID)
	dtos := make([]DecisionCardDTO, len(cards))
	for i := range cards {
		dtos[i] = toDecisionCardDTO(&cards[i])
		usersettings.AttachAmounts(&dtos[i], capital)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dtos,
	})
}
