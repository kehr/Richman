package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID generates a UUID v4 request ID, sets it in the response header,
// and stores a request-scoped logger with the requestId field in the context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)

		logger := zap.L().With(zap.String("requestId", requestID))
		c.Set("logger", logger)

		c.Next()
	}
}
