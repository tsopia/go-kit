package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize 限制请求体大小。
func MaxBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
