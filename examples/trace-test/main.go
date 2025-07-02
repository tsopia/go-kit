package main

import (
	"context"
	"fmt"

	"go-kit/pkg/constants"
	"go-kit/pkg/logger"
)

func main() {
	fmt.Println("=== Trace ID 和 Logger 联动功能测试 ===")

	// 1. 创建一个基础的 context
	ctx := context.Background()
	fmt.Println("1. 基础 context，无追踪信息")

	// 用基础 context 创建 logger
	logger1 := logger.FromContext(ctx)
	logger1.Info("这条日志没有追踪信息")

	// 2. 添加 trace ID 和 request ID
	traceID := constants.GenerateID()
	requestID := constants.GenerateID()

	fmt.Printf("\n2. 生成的 IDs:\n")
	fmt.Printf("   Trace ID: %s\n", traceID)
	fmt.Printf("   Request ID: %s\n", requestID)

	// 3. 将 IDs 添加到 context 中
	ctx = constants.WithTraceAndRequestID(ctx, traceID, requestID)

	// 用带有追踪信息的 context 创建 logger
	logger2 := logger.FromContext(ctx)
	fmt.Println("\n3. 带有追踪信息的日志输出:")
	logger2.Info("这条日志包含 trace_id 和 request_id")
	logger2.Warn("警告日志也会包含追踪信息", "action", "test")
	logger2.Error("错误日志同样包含追踪信息", "error", "这只是测试")

	// 4. 验证从 context 中提取 IDs
	fmt.Println("\n4. 从 context 中提取 IDs:")
	extractedTraceID := constants.TraceIDFromContext(ctx)
	extractedRequestID := constants.RequestIDFromContext(ctx)
	fmt.Printf("   提取的 Trace ID: %s\n", extractedTraceID)
	fmt.Printf("   提取的 Request ID: %s\n", extractedRequestID)

	// 5. 验证 ID 匹配
	fmt.Printf("\n5. ID 匹配验证:\n")
	fmt.Printf("   Trace ID 匹配: %v\n", traceID == extractedTraceID)
	fmt.Printf("   Request ID 匹配: %v\n", requestID == extractedRequestID)

	// 6. 演示常量的使用
	fmt.Printf("\n6. 常量值演示:\n")
	fmt.Printf("   TraceIDKey: %s\n", constants.TraceIDKey)
	fmt.Printf("   RequestIDKey: %s\n", constants.RequestIDKey)
	fmt.Printf("   TraceIDHeader: %s\n", constants.TraceIDHeader)
	fmt.Printf("   RequestIDHeader: %s\n", constants.RequestIDHeader)

	fmt.Println("\n=== 测试完成 ===")
}
