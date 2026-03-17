package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery 在请求处理 panic 时返回 500。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() == nil {
				return
			}

			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}
