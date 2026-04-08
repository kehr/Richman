package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	usersettings "github.com/richman/backend/internal/service/user_settings"
)

// UserSettingsHandler exposes user profile settings endpoints (PRD §8.3,
// TRD §5.3). Validation lives in the service layer; this handler is a thin
// wrapper around bind / call / respond.
type UserSettingsHandler struct {
	service *usersettings.Service
}

// NewUserSettingsHandler creates a new UserSettingsHandler.
func NewUserSettingsHandler(service *usersettings.Service) *UserSettingsHandler {
	return &UserSettingsHandler{service: service}
}

// RegisterRoutes wires user settings routes under the given group. All routes
// require authentication.
func (h *UserSettingsHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := rg.Group("/user/settings", authMiddleware)
	group.GET("", h.Get)
	group.PATCH("", h.Patch)
}

// Get handles GET /api/v1/user/settings.
func (h *UserSettingsHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	settings, err := h.service.GetUserSettings(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// Patch handles PATCH /api/v1/user/settings. Body fields are sparse — any
// nil pointer means "leave unchanged"; ClearTotalCapitalCny=true clears the
// total capital back to NULL.
func (h *UserSettingsHandler) Patch(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var patch usersettings.PatchUserSettings
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	settings, err := h.service.PatchUserSettings(c.Request.Context(), userID, &patch)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}
