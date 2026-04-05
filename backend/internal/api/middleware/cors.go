package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/config"
)

// CORS adds cross-origin resource sharing headers to responses.
// In dev mode it allows all origins. In prod mode it can be restricted.
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.IsDev() {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			origin := c.GetHeader("Origin")
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
