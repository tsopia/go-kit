package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConcurrencyLimitConfig 描述并发闸门配置。
type ConcurrencyLimitConfig struct {
	Limit int
	// OnRejected 允许业务自定义限流拒绝响应；如果回调未写出响应，则回退到默认 503。
	OnRejected func(*gin.Context)
}

// ConcurrencyLimit 为单进程全局并发设置上限。
func ConcurrencyLimit(limit int) gin.HandlerFunc {
	return ConcurrencyLimitWithConfig(ConcurrencyLimitConfig{
		Limit: limit,
	})
}

// ConcurrencyLimitWithConfig 根据配置创建并发闸门中间件。
func ConcurrencyLimitWithConfig(config ConcurrencyLimitConfig) gin.HandlerFunc {
	if config.Limit <= 0 {
		panic("middleware: limit must be greater than 0")
	}

	sem := make(chan struct{}, config.Limit)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
		default:
			c.Abort()
			if config.OnRejected != nil {
				config.OnRejected(c)
				if c.Writer.Written() {
					return
				}
			}

			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}

		defer func() {
			<-sem
		}()

		done := make(chan struct{})
		panicCh := make(chan any, 1)

		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					panicCh <- recovered
				}
			}()

			c.Next()
			close(done)
		}()

		select {
		case recovered := <-panicCh:
			panic(recovered)
		case <-done:
		case <-c.Request.Context().Done():
		}
	}
}
