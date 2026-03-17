package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout 为请求设置处理超时。
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

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
			return
		case <-ctx.Done():
			c.AbortWithStatus(http.StatusGatewayTimeout)
		}
	}
}
