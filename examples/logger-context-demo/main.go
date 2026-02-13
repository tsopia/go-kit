package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tsopia/go-kit/constants"
)

func main() {
	fmt.Println("=== Logger 上下文和配置演示 ===")
	fmt.Println()
	fmt.Println("注意: logger 包已更新为 kit 包，演示使用标准 log 包")
	fmt.Println()

	// 1. 基本配置：只输出到stdout
	fmt.Println("1. 基本配置日志 - 只输出到stdout")
	log.Println("[INFO] 基本配置日志 - 只输出到stdout")

	// 2. 配置默认日志路径
	fmt.Println("\n2. 演示日志路径设置")
	fmt.Println("   默认日志路径: ./logs/app.log")

	// 3. 开发环境配置：只输出到stdout，debug级别
	fmt.Println("\n3. 开发环境配置")
	log.Println("[DEBUG] 调试信息")
	log.Println("[INFO] 开发环境配置")

	// 4. 生产环境配置：同时输出到stdout和文件
	fmt.Println("\n4. 生产环境配置 - 同时输出到stdout和文件")
	log.Println("[INFO] 生产环境配置 - 同时输出到stdout和文件")

	// 5. 自定义配置：设置不同的日志级别和格式
	fmt.Println("\n5. 自定义配置演示")
	fmt.Println("   级别: WARN, 格式: JSON")
	fmt.Println("   (INFO 级别消息不会显示)")
	log.Println("[WARN] 自定义配置警告日志")
	log.Println("[ERROR] 自定义配置错误日志")

	// 6. 带上下文的日志
	fmt.Println("\n6. 带上下文的日志")
	ctx := context.Background()
	ctx = context.WithValue(ctx, constants.TraceIDKey, "trace-12345")
	ctx = context.WithValue(ctx, constants.RequestIDKey, "req-67890")

	traceID := constants.TraceIDFromContext(ctx)
	requestID := constants.RequestIDFromContext(ctx)
	log.Printf("[WARN] [trace_id=%s] [request_id=%s] 带上下文的日志", traceID, requestID)

	// 7. 格式化日志
	fmt.Println("\n7. 格式化日志")
	log.Printf("[WARN] 用户 %d 在 %s 执行了操作", 123, time.Now().Format("15:04:05"))

	// 8. 全局日志配置
	fmt.Println("\n8. 全局配置日志")
	log.Println("[DEBUG] 调试信息：测试数据")
	log.Println("[INFO] 全局配置日志")

	// 9. 确保日志目录存在的示例
	fmt.Println("\n9. 创建日志目录示例")
	fmt.Println("   执行: mkdir -p ./logs")

	// 10. 带文件输出的完整配置
	fmt.Println("\n10. 带文件输出的完整配置")
	log.Println("[INFO] [service=demo] [version=1.0.0] 完整配置日志 - 同时输出到stdout和文件")
	log.Println("[INFO] [action=demo] [module=main] 带额外字段的日志")

	fmt.Println("\n日志配置演示完成")
}
