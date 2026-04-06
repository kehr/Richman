package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	notificationSvc "github.com/richman/backend/internal/service/notification"
)

// NotificationHandler handles notification channel HTTP requests.
type NotificationHandler struct {
	notifService *notificationSvc.Service
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notifService *notificationSvc.Service) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

// RegisterRoutes registers notification routes on the given router group.
// All routes require authentication.
func (h *NotificationHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	channels := rg.Group("/notification/channels", authMiddleware)
	channels.GET("", h.ListChannels)
	channels.POST("", h.CreateChannel)
	channels.PUT("/:id", h.UpdateChannel)
	channels.DELETE("/:id", h.DeleteChannel)
}

// ListChannels handles GET /api/v1/notification/channels.
func (h *NotificationHandler) ListChannels(c *gin.Context) {
	userID := middleware.GetUserID(c)

	channels, err := h.notifService.ListChannels(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": channels,
	})
}

// CreateChannel handles POST /api/v1/notification/channels.
func (h *NotificationHandler) CreateChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	var input model.CreateChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ch, err := h.notifService.CreateChannel(c.Request.Context(), userID, &input, email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": ch,
	})
}

// UpdateChannel handles PUT /api/v1/notification/channels/:id.
func (h *NotificationHandler) UpdateChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid channel id",
			},
		})
		return
	}

	var input model.UpdateChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	ch, err := h.notifService.UpdateChannel(c.Request.Context(), userID, channelID, &input, email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ch,
	})
}

// DeleteChannel handles DELETE /api/v1/notification/channels/:id.
func (h *NotificationHandler) DeleteChannel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid channel id",
			},
		})
		return
	}

	if err := h.notifService.DeleteChannel(c.Request.Context(), userID, channelID, email); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "channel deleted"},
	})
}
