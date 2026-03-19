package middleware

import (
	"math"
	"net/http"
	"strconv"

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
		panic("middleware: rate must be greater than 0")
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

		// 计算到下一个令牌的真实等待秒数（向上取整，最小 1 秒）
		delay := limiter.Reserve().Delay()
		secs := int(math.Ceil(delay.Seconds()))
		if secs < 1 {
			secs = 1
		}
		c.Header("Retry-After", strconv.Itoa(secs))
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
}
