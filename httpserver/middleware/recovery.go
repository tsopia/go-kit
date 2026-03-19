package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryConfig 描述 panic 恢复行为。
type RecoveryConfig struct {
	// Logger 输出结构化恢复日志。
	// 复用 AccessLog 已有的 LoggerFunc 签名。
	Logger LoggerFunc

	// OnPanic 允许业务自定义 panic 后处理（如上报 Sentry）。
	// 在日志输出之后调用。
	OnPanic func(c *gin.Context, recovered any, stack []byte)
}

// Recovery 在请求处理 panic 时返回 500。
func Recovery() gin.HandlerFunc {
	return RecoveryWithConfig(RecoveryConfig{})
}

// RecoveryWithConfig 使用自定义配置创建 Recovery 中间件。
func RecoveryWithConfig(config RecoveryConfig) gin.HandlerFunc {
	logger := config.Logger
	if logger == nil {
		logger = defaultAccessLogger
	}

	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			stack := debug.Stack()

			logger(c.Request.Context(), "error", "recovery", map[string]any{
				"panic":      recovered,
				"stack":      string(stack),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"request_id": requestIDFromContext(c),
				"trace_id":   traceIDFromContext(c),
				"client_ip":  clientIPFromContext(c),
			})

			if config.OnPanic != nil {
				config.OnPanic(c, recovered, stack)
			}

			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}


