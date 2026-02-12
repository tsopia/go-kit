package kit

import (
	"context"

	"github.com/tsopia/go-kit/constants"
)

// contextExtractor 上下文信息提取器
type contextExtractor struct {
	keys ContextKeys
}

// newContextExtractor 创建上下文提取器
func newContextExtractor(keys ContextKeys) *contextExtractor {
	return &contextExtractor{keys: keys}
}

// extract 从 context 中提取信息
// 返回的 map 包含：trace_id, request_id, user_id 以及自定义字段
func (e *contextExtractor) extract(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	result := make(map[string]string)

	// 提取 trace ID（优先使用 constants 包）
	if traceID := constants.TraceIDFromContext(ctx); traceID != "" {
		result["trace_id"] = traceID
	} else if traceID := e.findValue(ctx, e.keys.Trace); traceID != "" {
		result["trace_id"] = traceID
	}

	// 提取 request ID（优先使用 constants 包）
	if requestID := constants.RequestIDFromContext(ctx); requestID != "" {
		result["request_id"] = requestID
	} else if requestID := e.findValue(ctx, e.keys.Request); requestID != "" {
		result["request_id"] = requestID
	}

	// 提取 user ID
	if userID := e.findValue(ctx, e.keys.User); userID != "" {
		result["user_id"] = userID
	}

	// 提取自定义字段
	for fieldName, keyList := range e.keys.Custom {
		if value := e.findValue(ctx, keyList); value != "" {
			result[fieldName] = value
		}
	}

	return result
}

// findValue 从 context 中按优先级查找值
// 遍历 keyList，找到第一个非空值即返回
func (e *contextExtractor) findValue(ctx context.Context, keyList []string) string {
	for _, key := range keyList {
		if val := ctx.Value(key); val != nil {
			if s, ok := val.(string); ok {
				return s
			}
		}
	}
	return ""
}
