package middleware

import "github.com/gin-gonic/gin"

// Deprecation injects RFC 8594 deprecation headers on sunset endpoints.
// Downstream clients can detect these headers to display warnings or migrate.
func Deprecation() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Deprecation", "true")
		c.Header("Sunset", "2027-01-01")
		c.Next()
	}
}
