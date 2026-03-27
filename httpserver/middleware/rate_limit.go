package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig 描述速率限制行为。
type RateLimitConfig struct {
	// Rate 每秒允许的请求数。
	Rate float64

	// Burst 突发上限，默认为 max(1, int(Rate))。
	Burst int

	// OnRejected 自定义拒绝响应。
	// 如果回调已写出响应，不再回退默认 429。
	OnRejected func(*gin.Context)
}

// RateLimit 创建全局速率限制中间件。
func RateLimit(rps float64) gin.HandlerFunc {
	return RateLimitWithConfig(RateLimitConfig{Rate: rps})
}

// RateLimitWithConfig 使用自定义配置创建速率限制中间件。
func RateLimitWithConfig(config RateLimitConfig) gin.HandlerFunc {
	if config.Rate <= 0 {
		slog.Warn("middleware: RateLimit called with rate <= 0, all requests will pass through")
		return func(c *gin.Context) { c.Next() }
	}

	burst := config.Burst
	if burst <= 0 {
		burst = int(config.Rate)
		if burst < 1 {
			burst = 1
		}
	}

	limiter := rate.NewLimiter(rate.Limit(config.Rate), burst)

	return func(c *gin.Context) {
		if limiter.Allow() {
			c.Next()
			return
		}

		c.Abort()

		if config.OnRejected != nil {
			config.OnRejected(c)
			if c.Writer.Written() {
				return
			}
		}

		// 固定 Retry-After: 1。
		// 不使用 limiter.Reserve().Delay() 是因为 Reserve() 会消费一个 token，
		// 导致 reject 路径额外降低后续请求的通过率。
		c.Header("Retry-After", "1")
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
}
