package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type visitorInfo struct {
	tokens    float64
	lastCheck time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorInfo
	rate     float64
	burst    int
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitorInfo),
		rate:     rate,
		burst:    burst,
	}

	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, info := range rl.visitors {
			if now.Sub(info.lastCheck) > 10*time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.visitors[key]
	if !exists {
		rl.visitors[key] = &visitorInfo{
			tokens:    float64(rl.burst) - 1,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(info.lastCheck).Seconds()
	info.tokens += elapsed * rl.rate
	if info.tokens > float64(rl.burst) {
		info.tokens = float64(rl.burst)
	}

	if info.tokens < 1 {
		return false
	}

	info.tokens--
	info.lastCheck = now
	return true
}

type RateLimitConfig struct {
	IPRate    float64
	IPBurst   int
	UserRate  float64
	UserBurst int
	AuthRate  float64
	AuthBurst int
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		IPRate:    100,
		IPBurst:   100,
		UserRate:  1000,
		UserBurst: 1000,
		AuthRate:  10,
		AuthBurst: 10,
	}
}

func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	ipLimiter := NewRateLimiter(cfg.IPRate, cfg.IPBurst)
	userLimiter := NewRateLimiter(cfg.UserRate, cfg.UserBurst)
	authLimiter := NewRateLimiter(cfg.AuthRate, cfg.AuthBurst)

	return func(c *gin.Context) {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		}

		isAuthRoute := strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth/")

		if isAuthRoute {
			if !authLimiter.Allow(ip) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "RATE_LIMITED",
						"message": "Too many requests. Please try again later.",
					},
				})
				return
			}
		} else {
			if !ipLimiter.Allow(ip) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "RATE_LIMITED",
						"message": "Too many requests. Please try again later.",
					},
				})
				return
			}
		}

		if userIDStr, exists := c.Get("user_id"); exists {
			userID := userIDStr.(uuid.UUID).String()
			if !userLimiter.Allow(userID) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "RATE_LIMITED",
						"message": "Too many requests. Please try again later.",
					},
				})
				return
			}
		}

		c.Next()
	}
}
