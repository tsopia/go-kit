package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ConcurrencyLimitConfig 描述并发闸门配置。
type ConcurrencyLimitConfig struct {
	Limit int
}

// ConcurrencyLimit 为单进程全局并发设置上限。
func ConcurrencyLimit(limit int) gin.HandlerFunc {
	return ConcurrencyLimitWithConfig(ConcurrencyLimitConfig{
		Limit: limit,
	})
}

// ConcurrencyLimitWithConfig 根据配置创建并发闸门中间件。
func ConcurrencyLimitWithConfig(config ConcurrencyLimitConfig) gin.HandlerFunc {
	sem := make(chan struct{}, config.Limit)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() {
				<-sem
			}()
			c.Next()
		default:
			c.AbortWithStatus(http.StatusServiceUnavailable)
		}
	}
}
