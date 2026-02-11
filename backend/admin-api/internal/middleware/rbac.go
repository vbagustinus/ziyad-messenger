package middleware

import (
	"net/http"

	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
)

var adminRoles = map[string]bool{
	"super_admin": true,
	"admin":        true,
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get(auth.ClaimsKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims, ok := val.(*auth.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if !adminRoles[claims.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
