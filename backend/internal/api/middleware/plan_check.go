package middleware

import "github.com/gin-gonic/gin"

// PlanCheck validates that the user's plan allows the requested operation.
// For MVP, all invite users have full access so this middleware is a pass-through.
func PlanCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
