// Package v2 contains the v2 API handlers and route registration.
package v2

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// v2 error codes returned by v2 API endpoints.
var (
	ErrRichsonUnavailable = model.NewAppError(503, "RICHSON_UNAVAILABLE", "quantitative service unavailable")
	ErrAnalysisInProgress = model.NewAppError(409, "ANALYSIS_IN_PROGRESS", "analysis already in progress")
	ErrAssetNotFound      = model.NewAppError(404, "ASSET_NOT_FOUND", "asset not found")
	ErrFeedbackInvalid    = model.NewAppError(400, "FEEDBACK_INVALID", "invalid feedback data")
	ErrRateLimitExceeded  = model.NewAppError(429, "RATE_LIMIT_EXCEEDED", "too many requests")
	ErrAccountDeletion    = model.NewAppError(400, "ACCOUNT_DELETION_FAILED", "account deletion failed")
	ErrInviteCodeInvalid  = model.NewAppError(400, "INVITE_CODE_INVALID", "invalid invite code")
	ErrDisclaimerRequired = model.NewAppError(400, "DISCLAIMER_REQUIRED", "disclaimer must be accepted")
)

// handleServiceError maps service errors to appropriate HTTP responses.
// AppError values are forwarded directly; any other error is logged and mapped
// to a 500. The request-scoped logger is used when available.
func handleServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{
			"error": gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}
	logger := zap.L()
	if v, exists := c.Get("logger"); exists {
		if lg, ok := v.(*zap.Logger); ok && lg != nil {
			logger = lg
		}
	}
	logger.Error("unhandled v2 service error",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Error(err),
	)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "internal server error",
		},
	})
}
