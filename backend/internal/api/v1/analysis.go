package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/richman/backend/internal/api/middleware"
	analysisService "github.com/richman/backend/internal/service/analysis"
)

// reanalyzeAllWindow is the cool-down between two reanalyze-all requests
// from the same user. Intentionally coarse (10 minutes) so a dashboard
// banner double-click cannot stampede the synthesis pipeline and rack up
// extra LLM tokens. Swap this for a config knob when operators need to
// tune it per-environment.
const reanalyzeAllWindow = 10 * time.Minute

// AnalysisHandler handles analysis-related HTTP requests.
type AnalysisHandler struct {
	analysisSvc *analysisService.Service
}

// NewAnalysisHandler creates a new AnalysisHandler.
func NewAnalysisHandler(analysisSvc *analysisService.Service) *AnalysisHandler {
	return &AnalysisHandler{analysisSvc: analysisSvc}
}

// RegisterRoutes registers analysis routes on the given router group.
// Reanalyze-all is installed behind a per-user 1/10min rate limit so the
// dashboard banner CTA cannot be used to hammer the synthesis pipeline.
func (h *AnalysisHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/analysis", authMiddleware)
	group.POST("/trigger", h.Trigger)
	group.POST("/reanalyze-all", middleware.PerUserRateLimit(reanalyzeAllWindow), h.ReanalyzeAll)
	group.GET("/tasks/:taskId", h.GetTask)
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

// GetTask handles GET /api/v1/analysis/tasks/:taskId.
// Returns the current analysis task status for the authenticated user.
func (h *AnalysisHandler) GetTask(c *gin.Context) {
	taskID := c.Param("taskId")
	userID := middleware.GetUserID(c)

	task := h.analysisSvc.GetTaskStore().Get(taskID)
	if task == nil || task.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

// ReanalyzeAll handles POST /api/v1/analysis/reanalyze-all. It triggers
// the same background pipeline as /trigger but is guarded by the per-user
// rate limiter installed in RegisterRoutes. The response shape mirrors
// /trigger so frontend callers can reuse their existing task-polling UI.
func (h *AnalysisHandler) ReanalyzeAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	taskID := uuid.New().String()

	h.analysisSvc.TriggerReanalyzeAll(c.Request.Context(), userID, taskID)

	c.JSON(http.StatusAccepted, gin.H{
		"data": gin.H{
			"taskId":  taskID,
			"message": "reanalysis started",
		},
	})
}
