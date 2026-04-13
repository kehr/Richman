package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	analysisService "github.com/richman/backend/internal/service/analysis"
)

// TaskHandler handles task status query requests.
type TaskHandler struct {
	taskStore *analysisService.TaskStore
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskStore *analysisService.TaskStore) *TaskHandler {
	return &TaskHandler{taskStore: taskStore}
}

// RegisterRoutes registers task routes on the given router group.
// Deprecated: task status is superseded by GET /api/v2/analysis/jobs/:jobId.
func (h *TaskHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/tasks", authMiddleware, middleware.Deprecation())
	group.GET("/:taskId", h.GetStatus)
}

// GetStatus handles GET /api/v1/tasks/:taskId.
func (h *TaskHandler) GetStatus(c *gin.Context) {
	taskID := c.Param("taskId")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "taskId is required",
			},
		})
		return
	}

	status := h.taskStore.Get(taskID)
	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "task not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": status,
	})
}
