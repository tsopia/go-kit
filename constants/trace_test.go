package constants

import (
	"context"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	// 测试常量值
	if TraceIDKey != "trace_id" {
		t.Errorf("Expected TraceIDKey to be 'trace_id', got '%s'", TraceIDKey)
	}

	if RequestIDKey != "request_id" {
		t.Errorf("Expected RequestIDKey to be 'request_id', got '%s'", RequestIDKey)
	}

	if TraceIDHeader != "X-Trace-ID" {
		t.Errorf("Expected TraceIDHeader to be 'X-Trace-ID', got '%s'", TraceIDHeader)
	}

	if RequestIDHeader != "X-Request-ID" {
		t.Errorf("Expected RequestIDHeader to be 'X-Request-ID', got '%s'", RequestIDHeader)
	}
}

func TestGenerateID(t *testing.T) {
	// 测试ID生成
	id1 := GenerateID()
	id2 := GenerateID()

	// ID应该不为空
	if id1 == "" {
		t.Error("GenerateID() should not return empty string")
	}

	if id2 == "" {
		t.Error("GenerateID() should not return empty string")
	}

	// 两次生成的ID应该不同
	if id1 == id2 {
		t.Error("GenerateID() should generate unique IDs")
	}

	// ID应该是32个字符的十六进制字符串（16字节 * 2）
	if len(id1) != 32 {
		t.Errorf("Expected ID length to be 32, got %d", len(id1))
	}

	// 验证是否为有效的十六进制字符串
	for _, char := range id1 {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("ID should only contain hex characters, found '%c' in '%s'", char, id1)
		}
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-123"

	// 添加 trace ID 到 context
	newCtx := WithTraceID(ctx, traceID)

	// 验证 trace ID 被正确设置
	if value := newCtx.Value(TraceIDKey); value != traceID {
		t.Errorf("Expected trace ID '%s', got '%v'", traceID, value)
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-456"

	// 添加 request ID 到 context
	newCtx := WithRequestID(ctx, requestID)

	// 验证 request ID 被正确设置
	if value := newCtx.Value(RequestIDKey); value != requestID {
		t.Errorf("Expected request ID '%s', got '%v'", requestID, value)
	}
}

func TestWithTraceAndRequestID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-789"
	requestID := "test-request-abc"

	// 同时添加 trace ID 和 request ID
	newCtx := WithTraceAndRequestID(ctx, traceID, requestID)

	// 验证两个ID都被正确设置
	if value := newCtx.Value(TraceIDKey); value != traceID {
		t.Errorf("Expected trace ID '%s', got '%v'", traceID, value)
	}

	if value := newCtx.Value(RequestIDKey); value != requestID {
		t.Errorf("Expected request ID '%s', got '%v'", requestID, value)
	}
}

func TestTraceIDFromContext(t *testing.T) {
	ctx := context.Background()

	// 测试空的 context
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		t.Errorf("Expected empty trace ID from empty context, got '%s'", traceID)
	}

	// 测试有 trace ID 的 context
	expectedTraceID := "test-trace-def"
	ctxWithTrace := WithTraceID(ctx, expectedTraceID)

	if traceID := TraceIDFromContext(ctxWithTrace); traceID != expectedTraceID {
		t.Errorf("Expected trace ID '%s', got '%s'", expectedTraceID, traceID)
	}

	// 测试错误类型的值
	ctxWithWrongType := context.WithValue(ctx, TraceIDKey, 123)
	if traceID := TraceIDFromContext(ctxWithWrongType); traceID != "" {
		t.Errorf("Expected empty trace ID for wrong type, got '%s'", traceID)
	}
}

func TestRequestIDFromContext(t *testing.T) {
	ctx := context.Background()

	// 测试空的 context
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		t.Errorf("Expected empty request ID from empty context, got '%s'", requestID)
	}

	// 测试有 request ID 的 context
	expectedRequestID := "test-request-ghi"
	ctxWithRequest := WithRequestID(ctx, expectedRequestID)

	if requestID := RequestIDFromContext(ctxWithRequest); requestID != expectedRequestID {
		t.Errorf("Expected request ID '%s', got '%s'", expectedRequestID, requestID)
	}

	// 测试错误类型的值
	ctxWithWrongType := context.WithValue(ctx, RequestIDKey, 456)
	if requestID := RequestIDFromContext(ctxWithWrongType); requestID != "" {
		t.Errorf("Expected empty request ID for wrong type, got '%s'", requestID)
	}
}

func TestIDFormat(t *testing.T) {
	// 生成多个ID并验证格式一致性
	for i := 0; i < 10; i++ {
		id := GenerateID()

		// 验证长度
		if len(id) != 32 {
			t.Errorf("ID %d: expected length 32, got %d", i, len(id))
		}

		// 验证只包含小写十六进制字符
		if strings.ToLower(id) != id {
			t.Errorf("ID %d: should be lowercase, got '%s'", i, id)
		}

		// 验证是有效的十六进制
		for _, char := range id {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				t.Errorf("ID %d: invalid hex character '%c' in '%s'", i, char, id)
				break
			}
		}
	}
}

func TestIDUniqueness(t *testing.T) {
	// 生成大量ID并验证唯一性
	ids := make(map[string]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id := GenerateID()
		if ids[id] {
			t.Errorf("Duplicate ID generated: '%s'", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestContextChaining(t *testing.T) {
	// 测试链式操作
	ctx := context.Background()
	traceID := "chain-trace"
	requestID := "chain-request"

	// 分步添加
	ctx = WithTraceID(ctx, traceID)
	ctx = WithRequestID(ctx, requestID)

	// 验证两个值都存在
	if extractedTrace := TraceIDFromContext(ctx); extractedTrace != traceID {
		t.Errorf("Expected trace ID '%s', got '%s'", traceID, extractedTrace)
	}

	if extractedRequest := RequestIDFromContext(ctx); extractedRequest != requestID {
		t.Errorf("Expected request ID '%s', got '%s'", requestID, extractedRequest)
	}
}

func TestContextOverwrite(t *testing.T) {
	// 测试覆盖值
	ctx := context.Background()
	originalTrace := "original-trace"
	newTrace := "new-trace"

	// 添加原始值
	ctx = WithTraceID(ctx, originalTrace)
	if extractedTrace := TraceIDFromContext(ctx); extractedTrace != originalTrace {
		t.Errorf("Expected original trace ID '%s', got '%s'", originalTrace, extractedTrace)
	}

	// 覆盖值
	ctx = WithTraceID(ctx, newTrace)
	if extractedTrace := TraceIDFromContext(ctx); extractedTrace != newTrace {
		t.Errorf("Expected new trace ID '%s', got '%s'", newTrace, extractedTrace)
	}
}
