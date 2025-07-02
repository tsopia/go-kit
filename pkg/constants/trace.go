package constants

import (
	"context"
	"crypto/rand"
	"fmt"
)

// Trace and Request ID 相关常量
const (
	// TraceIDKey trace ID 在 context 中的 key
	TraceIDKey = "trace_id"
	// RequestIDKey request ID 在 context 中的 key
	RequestIDKey = "request_id"
)

// HTTP headers for trace and request IDs
const (
	// TraceIDHeader trace ID 的 HTTP 头名称
	TraceIDHeader = "X-Trace-ID"
	// RequestIDHeader request ID 的 HTTP 头名称
	RequestIDHeader = "X-Request-ID"
)

// GenerateID 生成随机的 ID（用于 trace id 和 request id）
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// TraceIDFromContext 从 context 中提取 trace ID
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// RequestIDFromContext 从 context 中提取 request ID
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithTraceID 将 trace ID 添加到 context 中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithRequestID 将 request ID 添加到 context 中
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceAndRequestID 同时将 trace ID 和 request ID 添加到 context 中
func WithTraceAndRequestID(ctx context.Context, traceID, requestID string) context.Context {
	ctx = WithTraceID(ctx, traceID)
	ctx = WithRequestID(ctx, requestID)
	return ctx
}
