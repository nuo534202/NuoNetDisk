package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/nuonuo/nuonetdisk/internal/auth"
)

func Apply(r *gin.Engine, jwtService *auth.JWTService, corsOrigins []string, rateLimitCfg RateLimitConfig) {
	r.Use(Recovery())
	r.Use(CORS(corsOrigins))
	r.Use(Logger())
	r.Use(RateLimit(rateLimitCfg))

	authMw := NewAuthMiddleware(jwtService)
	r.Use(authMw.RequireAuth())
}
