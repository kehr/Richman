package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/service/auth"
)

const (
	// ContextKeyUserID is the gin context key for the authenticated user's ID.
	ContextKeyUserID = "userID"
	// ContextKeyEmail is the gin context key for the authenticated user's email.
	ContextKeyEmail = "email"
	// ContextKeyRole is the gin context key for the authenticated user's role.
	ContextKeyRole = "role"
)

// Auth validates the Bearer token in the Authorization header.
// It sets the userID, email, and role in the context on success.
func Auth(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "missing authorization header",
				},
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "invalid authorization format",
				},
			})
			return
		}

		claims, err := authService.ValidateJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "invalid or expired token",
				},
			})
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyRole, claims.Role)

		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the gin context.
func GetUserID(c *gin.Context) int64 {
	id, _ := c.Get(ContextKeyUserID)
	userID, _ := id.(int64)
	return userID
}
