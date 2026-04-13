package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	feedbackSvc "github.com/richman/backend/internal/service/feedback"
	"go.uber.org/zap"
)

// FeedbackHandler handles v2 feedback endpoints.
type FeedbackHandler struct {
	feedbackSvc *feedbackSvc.Service
	logger      *zap.Logger
}

// NewFeedbackHandler creates a new FeedbackHandler.
func NewFeedbackHandler(feedbackSvc *feedbackSvc.Service, logger *zap.Logger) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackSvc: feedbackSvc,
		logger:      logger,
	}
}

// createFeedback handles POST /api/v2/feedback.
// Creates a new user feedback entry for an analysis.
func (h *FeedbackHandler) createFeedback(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var input feedbackSvc.CreateFeedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	feedbackID, err := h.feedbackSvc.Create(c.Request.Context(), userID, &input)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{"feedbackId": feedbackID},
	})
}
