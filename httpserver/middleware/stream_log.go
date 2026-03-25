package middleware

import (
	"context"
	"net/http"
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
