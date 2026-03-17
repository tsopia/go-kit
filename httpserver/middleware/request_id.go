package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// RequestID 为请求注入 request id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := utils.GenerateID()

		c.Header(utils.RequestIDHeader, requestID)
		c.Set(utils.RequestIDKey, requestID)

		ctx := utils.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
