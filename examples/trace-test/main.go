package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tsopia/go-kit/utils"
)

func main() {
	fmt.Println("=== Trace ID 和 Logger 联动功能测试 ===")

	// 1. 创建一个基础的 context
	ctx := context.Background()
	fmt.Println("1. 基础 context，无追踪信息")
	log.Println("这条日志没有追踪信息")

	// 2. 添加 trace ID 和 request ID
	traceID := utils.GenerateID()
	requestID := utils.GenerateID()

	fmt.Printf("\n2. 生成的 IDs:\n")
	fmt.Printf("   Trace ID: %s\n", traceID)
	fmt.Printf("   Request ID: %s\n", requestID)

	// 3. 将 IDs 添加到 context 中
	ctx = utils.WithTraceAndRequestID(ctx, traceID, requestID)

	// 4. 使用带有追踪信息的 context
	fmt.Println("\n3. 带有追踪信息的日志输出:")
	log.Printf("[trace_id=%s] 这条日志包含 trace_id\n", utils.TraceIDFromContext(ctx))
	log.Printf("[request_id=%s] 这条日志包含 request_id\n", utils.RequestIDFromContext(ctx))

	// 4. 验证从 context 中提取 IDs
	fmt.Println("\n4. 从 context 中提取 IDs:")
	extractedTraceID := utils.TraceIDFromContext(ctx)
	extractedRequestID := utils.RequestIDFromContext(ctx)
	fmt.Printf("   提取的 Trace ID: %s\n", extractedTraceID)
	fmt.Printf("   提取的 Request ID: %s\n", extractedRequestID)

	// 5. 验证 ID 匹配
	fmt.Printf("\n5. ID 匹配验证:\n")
	fmt.Printf("   Trace ID 匹配: %v\n", traceID == extractedTraceID)
	fmt.Printf("   Request ID 匹配: %v\n", requestID == extractedRequestID)

	// 6. 演示常量的使用
	fmt.Printf("\n6. 常量值演示:\n")
	fmt.Printf("   TraceIDKey: %s\n", utils.TraceIDKey)
	fmt.Printf("   RequestIDKey: %s\n", utils.RequestIDKey)
	fmt.Printf("   TraceIDHeader: %s\n", utils.TraceIDHeader)
	fmt.Printf("   RequestIDHeader: %s\n", utils.RequestIDHeader)

	fmt.Println("\n=== 测试完成 ===")
}
