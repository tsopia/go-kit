package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

// StreamLogConfig 描述流式连接日志行为。
type StreamLogConfig struct {
	Logger                LoggerFunc
	AllowedRequestHeaders []string
}

type streamLogConfigKey struct{}

// WithStreamLogConfig 将流式日志配置写入 context。
func WithStreamLogConfig(ctx context.Context, config StreamLogConfig) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	cloned := config
	cloned.AllowedRequestHeaders = append([]string(nil), config.AllowedRequestHeaders...)
	return context.WithValue(ctx, streamLogConfigKey{}, cloned)
}

// StreamLogConfigFromContext 从 context 读取流式日志配置。
func StreamLogConfigFromContext(ctx context.Context) (StreamLogConfig, bool) {
	if ctx == nil {
		return StreamLogConfig{}, false
	}
	config, ok := ctx.Value(streamLogConfigKey{}).(StreamLogConfig)
	return config, ok
}

// DefaultLogger 使用默认结构化 logger 输出日志。
func DefaultLogger(ctx context.Context, level string, event string, fields map[string]any) {
	defaultAccessLogger(ctx, level, event, fields)
}

// CaptureAllowedRequestHeaders 捕获 allowlist 中的请求头。
func CaptureAllowedRequestHeaders(headers http.Header, allowlist []string) map[string]any {
	if len(allowlist) == 0 {
		return nil
	}

	result := make(map[string]any, len(allowlist))
	for _, headerName := range allowlist {
		values := headers.Values(headerName)
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			result[headerName] = values[0]
			continue
		}
		cloned := append([]string(nil), values...)
		result[headerName] = cloned
	}
	return result
}

// StreamObserver 观测流式连接的建立与断开。
// 实现方（如 observability/prometheus）通过 WithStreamObserver 注入 context，
// SSE/WS handler 在连接建立/断开时回调，从而在不反向依赖 observability 子包的
// 前提下维护活跃连接指标。
type StreamObserver interface {
	OnConnect(transport string)
	OnDisconnect(transport string)
}

type streamObserverKey struct{}

// WithStreamObserver 将 StreamObserver 写入 context。
func WithStreamObserver(ctx context.Context, obs StreamObserver) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, streamObserverKey{}, obs)
}

// StreamObserverFromContext 从 context 读取 StreamObserver。
func StreamObserverFromContext(ctx context.Context) (StreamObserver, bool) {
	if ctx == nil {
		return nil, false
	}
	obs, ok := ctx.Value(streamObserverKey{}).(StreamObserver)
	return obs, ok
}

// MarkStreaming 返回一个把流式标记写入 gin.Context 的中间件，
// 供通过 srv.StreamingGroup() 自定义注册的流式路由使用。
// SSE/SSEPost/WS 便利方法已自动打标，无需再用此中间件。
func MarkStreaming(transport string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(utils.StreamingKey, transport)
		c.Next()
	}
}
