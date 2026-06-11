package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func redactSensitivePaths(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "shares" && i+1 < len(parts) {
			parts[i+1] = ":token"
		}
	}
	return strings.Join(parts, "/")
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		method := c.Request.Method
		path := redactSensitivePaths(c.Request.URL.Path)
		status := c.Writer.Status()

		log.Printf("[%d] %s %s (%s)", status, method, path, duration)
	}
}
