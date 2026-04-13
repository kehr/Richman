package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/config"
)

// CORS adds cross-origin resource sharing headers to responses.
// In dev mode it allows all origins (*). In non-dev mode it restricts to the
// origins listed in cfg.CORS.AllowedOrigins (CORS_ALLOWED_ORIGINS env var,
// comma-separated). An unmatched origin receives no Allow-Origin header.
func CORS(cfg *config.Config) gin.HandlerFunc {
	// Build an O(1) lookup set from the configured allowed origins.
	allowedSet := make(map[string]struct{}, len(cfg.CORS.AllowedOrigins))
	for _, o := range cfg.CORS.AllowedOrigins {
		allowedSet[strings.TrimSpace(o)] = struct{}{}
	}

	return func(c *gin.Context) {
		if cfg.IsDev() {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			origin := c.GetHeader("Origin")
			if origin != "" {
				if _, ok := allowedSet[origin]; ok {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
				}
				// Non-matching origins receive no Allow-Origin header (blocked).
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
