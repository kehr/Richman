package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	briefingSvc "github.com/richman/backend/internal/service/briefing"
	"go.uber.org/zap"
)

// BriefingHandler handles v2 briefing endpoints.
type BriefingHandler struct {
	briefingSvc *briefingSvc.Service
	logger      *zap.Logger
}

// NewBriefingHandler creates a new BriefingHandler.
func NewBriefingHandler(briefingSvc *briefingSvc.Service, logger *zap.Logger) *BriefingHandler {
	return &BriefingHandler{
		briefingSvc: briefingSvc,
		logger:      logger,
	}
}

// getBriefing handles GET /api/v2/briefing.
// Returns the authenticated user's daily portfolio briefing.
func (h *BriefingHandler) getBriefing(c *gin.Context) {
	userID := middleware.GetUserID(c)

	briefing, err := h.briefingSvc.GetBriefing(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": briefing})
}
