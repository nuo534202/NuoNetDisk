package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Admin access required",
				},
			})
			return
		}
		c.Next()
	}
}

// RequireSuperAdmin restricts access to the super admin account configured via
// ADMIN_EMAIL. It must run after RequireAuth and RequireAdmin so that the
// "email" context value is populated.
func RequireSuperAdmin(adminEmail string) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")
		currentEmail, _ := email.(string)
		if adminEmail == "" || !strings.EqualFold(currentEmail, adminEmail) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Super admin access required",
				},
			})
			return
		}
		c.Next()
	}
}

func RequireNonAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if exists && isAdmin.(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "This endpoint is not available for admin accounts",
				},
			})
			return
		}
		c.Next()
	}
}
