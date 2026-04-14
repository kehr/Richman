package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	userSettingsSvc "github.com/richman/backend/internal/service/user_settings"
	"go.uber.org/zap"
)

// UserHandler handles v2 user endpoints.
type UserHandler struct {
	userSettingsSvc *userSettingsSvc.Service
	logger          *zap.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userSettingsSvc *userSettingsSvc.Service, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userSettingsSvc: userSettingsSvc,
		logger:          logger,
	}
}

// patchRiskPreferenceRequest is the request body for PATCH /api/v2/user/risk-preference.
type patchRiskPreferenceRequest struct {
	RiskPreference string `json:"riskPreference" binding:"required"`
}

// updateRiskPreference handles PATCH /api/v2/user/risk-preference.
func (h *UserHandler) updateRiskPreference(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req patchRiskPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	if err := h.userSettingsSvc.UpdateRiskPreference(c.Request.Context(), userID, req.RiskPreference); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "risk preference updated"}})
}

// patchEmailPushRequest is the request body for PATCH /api/v2/user/email-push.
// The field name mirrors the GET response shape so the frontend can use a
// single EmailPushPrefs interface for both directions.
type patchEmailPushRequest struct {
	EmailPushEnabled *bool `json:"emailPushEnabled" binding:"required"`
}

// getEmailPush handles GET /api/v2/user/email-push and returns the current
// flag for the authenticated user. The response shape matches the PATCH
// response so the frontend can reuse the same type.
func (h *UserHandler) getEmailPush(c *gin.Context) {
	userID := middleware.GetUserID(c)

	enabled, err := h.userSettingsSvc.GetEmailPushEnabled(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"emailPushEnabled": enabled}})
}

// updateEmailPush handles PATCH /api/v2/user/email-push.
func (h *UserHandler) updateEmailPush(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req patchEmailPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	enabled := *req.EmailPushEnabled
	if err := h.userSettingsSvc.UpdateEmailPush(c.Request.Context(), userID, enabled); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"emailPushEnabled": enabled}})
}
