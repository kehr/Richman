package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLog logs HTTP request details including method, path, status code,
// and latency using the request-scoped zap logger.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		logger := zap.L()
		if l, exists := c.Get("logger"); exists {
			if lg, ok := l.(*zap.Logger); ok {
				logger = lg
			}
		}

		logger.Info("request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.Int("bodySize", c.Writer.Size()),
		)
	}
}
