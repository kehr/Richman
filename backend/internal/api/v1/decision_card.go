package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	decisioncard "github.com/richman/backend/internal/service/decision_card"
)

// DecisionCardHandler handles decision card query requests.
type DecisionCardHandler struct {
	cardSvc *decisioncard.Service
}

// NewDecisionCardHandler creates a new DecisionCardHandler.
func NewDecisionCardHandler(cardSvc *decisioncard.Service) *DecisionCardHandler {
	return &DecisionCardHandler{cardSvc: cardSvc}
}

// RegisterRoutes registers decision card routes on the given router group.
func (h *DecisionCardHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/decision-cards", authMiddleware)
	group.GET("", h.ListLatest)
	group.GET("/history", h.ListHistory)
	group.GET("/:id", h.GetByID)
}

// ListLatest handles GET /api/v1/decision-cards.
func (h *DecisionCardHandler) ListLatest(c *gin.Context) {
	userID := middleware.GetUserID(c)

	cards, err := h.cardSvc.ListLatest(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cards,
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

	c.JSON(http.StatusOK, gin.H{
		"data": card,
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

	c.JSON(http.StatusOK, gin.H{
		"data": cards,
	})
}
