package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richman/backend/internal/api/middleware"
	analysisService "github.com/richman/backend/internal/service/analysis"
)

// AnalysisHandler handles analysis-related HTTP requests.
type AnalysisHandler struct {
	analysisSvc *analysisService.Service
}

// NewAnalysisHandler creates a new AnalysisHandler.
func NewAnalysisHandler(analysisSvc *analysisService.Service) *AnalysisHandler {
	return &AnalysisHandler{analysisSvc: analysisSvc}
}

// RegisterRoutes registers analysis routes on the given router group.
func (h *AnalysisHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/analysis", authMiddleware)
	group.POST("/trigger", h.Trigger)
}

// Trigger handles POST /api/v1/analysis/trigger.
// It starts an async analysis and returns 202 Accepted with a task ID.
func (h *AnalysisHandler) Trigger(c *gin.Context) {
	userID := middleware.GetUserID(c)
	taskID := uuid.New().String()

	h.analysisSvc.TriggerAnalysis(c.Request.Context(), userID, taskID)

	c.JSON(http.StatusAccepted, gin.H{
		"data": gin.H{
			"taskId":  taskID,
			"message": "analysis started",
		},
	})
}
