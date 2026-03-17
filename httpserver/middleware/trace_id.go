package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// TraceID 为请求注入 trace id。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(utils.TraceIDHeader)
		if traceID == "" {
			traceID = utils.GenerateID()
		}

		c.Header(utils.TraceIDHeader, traceID)
		c.Set(utils.TraceIDKey, traceID)

		ctx := utils.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
