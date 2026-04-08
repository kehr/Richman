package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/service/onboarding"
)

// OnboardingHandler exposes per-user onboarding status endpoints
// (PRD §2.3, §6.2).
type OnboardingHandler struct {
	service *onboarding.Service
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(service *onboarding.Service) *OnboardingHandler {
	return &OnboardingHandler{service: service}
}

// RegisterRoutes wires the onboarding routes under the given group. All
// routes require authentication.
func (h *OnboardingHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/onboarding", authMiddleware)
	group.GET("", h.GetStatus)
	group.POST("/complete", h.MarkCompleted)
	group.POST("/skip", h.MarkSkipped)
	group.DELETE("", h.Reset)
}

// GetStatus handles GET /api/v1/onboarding.
func (h *OnboardingHandler) GetStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status, err := h.service.GetStatus(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// MarkCompleted handles POST /api/v1/onboarding/complete.
func (h *OnboardingHandler) MarkCompleted(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status, err := h.service.MarkCompleted(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// MarkSkipped handles POST /api/v1/onboarding/skip.
func (h *OnboardingHandler) MarkSkipped(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status, err := h.service.MarkSkipped(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

// Reset handles DELETE /api/v1/onboarding. Clears both onboarding_completed_at
// and onboarding_skipped_at atomically so the user is treated as not yet
// onboarded. Used by the Settings AccountTab CTA when a user wants to re-run
// the flow.
func (h *OnboardingHandler) Reset(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status, err := h.service.Reset(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}
