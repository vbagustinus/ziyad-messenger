package middleware

import (
	"admin-service/internal/audit"
	"admin-service/internal/auth"

	"github.com/gin-gonic/gin"
)

func Audit(action, targetResource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		val, ok := c.Get(auth.ClaimsKey)
		if !ok {
			return
		}
		claims, ok := val.(*auth.Claims)
		if !ok {
			return
		}
		_ = audit.Log(claims.UserID, claims.Username, action, targetResource, c.Request.URL.Path, c.ClientIP())
	}
}
