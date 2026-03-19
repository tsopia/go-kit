package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig 描述超时中间件行为。
type TimeoutConfig struct {
	// Timeout 请求处理超时时间。
	Timeout time.Duration

	// OnTimeout 自定义超时响应。
	// 如果回调未写出响应，回退到 504。
	OnTimeout func(*gin.Context)
}

// Timeout 为请求设置处理超时。
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return TimeoutWithConfig(TimeoutConfig{Timeout: timeout})
}

// TimeoutWithConfig 使用自定义配置创建超时中间件。
func TimeoutWithConfig(config TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), config.Timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			if config.OnTimeout != nil {
				config.OnTimeout(c)
				if c.Writer.Written() {
					return
				}
			}
			c.AbortWithStatus(http.StatusGatewayTimeout)
		}
	}
}
