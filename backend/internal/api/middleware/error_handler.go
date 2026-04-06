package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// ErrorHandler is a recovery middleware that catches panics and returns
// a structured JSON error response.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger := zap.L()
				if l, exists := c.Get("logger"); exists {
					if lg, ok := l.(*zap.Logger); ok {
						logger = lg
					}
				}
				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()

		c.Next()

		// Handle errors set during request processing.
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "internal server error",
				},
			})
		}
	}
}
