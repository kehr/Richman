package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// RoleAdmin is the user.Role value granted manual / batch / restoration powers.
	RoleAdmin = "admin"
)

// RequireAdmin aborts the request with 403 unless the upstream Auth middleware
// stored a role == "admin" on the gin context. Must be chained AFTER Auth so
// the role key is present.
//
// Used by management endpoints (e.g. POST /api/v2/analysis/trigger-batch)
// where the action is not safe for ordinary users -- richman-backend-v2-trd
// SS8.8 (manual batch retry) and SS8.7 (cron mutex bypass).
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "missing role; admin auth required",
				},
			})
			return
		}
		role, _ := raw.(string)
		if role != RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "admin role required",
				},
			})
			return
		}
		c.Next()
	}
}
