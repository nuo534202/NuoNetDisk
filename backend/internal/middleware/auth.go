package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nuonuo/nuonetdisk/internal/auth"
)

type AuthMiddleware struct {
	jwtService  *auth.JWTService
	publicPaths map[string]bool
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		publicPaths: map[string]bool{
			"POST /api/v1/auth/register": true,
			"POST /api/v1/auth/login":    true,
			"POST /api/v1/auth/refresh":  true,
		},
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.Method + " " + c.Request.URL.Path

		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/shares/token/") && c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		if m.publicPaths[path] {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "MISSING_TOKEN",
					"message": "Authorization header is required",
				},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_TOKEN",
					"message": "Authorization header must be Bearer {token}",
				},
			})
			return
		}

		claims, err := m.jwtService.ValidateAccessToken(parts[1])
		if err != nil {
			code := "INVALID_TOKEN"
			if strings.Contains(err.Error(), "expired") {
				code = "TOKEN_EXPIRED"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    code,
					"message": "Invalid or expired access token",
				},
			})
			return
		}

		c.Set("user_id", claims.UserID.String())
		c.Set("email", claims.Email)
		c.Next()
	}
}
