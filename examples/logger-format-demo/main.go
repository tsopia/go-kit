package main

import (
	"errors"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("=== Go-Kit Logger 格式化功能演示 ===")
	fmt.Println()
	fmt.Println("注意: logger 包已更新为 kit 包，演示使用标准 log 包")
	fmt.Println()

	// 基本格式化方法
	log.Printf("[DEBUG] 这是一个调试消息: %s", "debug info")
	log.Printf("[INFO] 用户 %s (ID: %d) 登录成功", "alice", 12345)
	log.Printf("[WARN] 内存使用率达到 %.2f%%", 85.5)
	log.Printf("[ERROR] 数据库连接失败: %v", errors.New("connection timeout"))

	fmt.Println("\n2. 结构化日志方法:")
	fmt.Println("-------------------------------")

	// 结构化日志方法
	log.Printf("[INFO] user_login user=%s user_id=%d ip=%s timestamp=%s",
		"alice", 12345, "192.168.1.1", time.Now().Format(time.RFC3339))

	fmt.Println("\n3. 性能优化 - 级别检查:")
	fmt.Println("-------------------------------")

	// 模拟日志级别控制
	currentLevel := "INFO"
	if currentLevel == "DEBUG" {
		log.Printf("[DEBUG] 这个消息会被处理: %s %d", "expensive operation", 999999)
	} else {
		fmt.Println("(DEBUG消息被跳过，因为当前级别是INFO)")
	}

	fmt.Printf("当前日志级别: %s\n", currentLevel)

	fmt.Println("\n4. 错误处理演示:")
	fmt.Println("-------------------------------")

	// 模拟各种错误情况
	err := simulateError()
	if err != nil {
		log.Printf("[ERROR] 操作失败: %v", err)
	}

	// 带上下文的错误
	userID := 12345
	operation := "update_profile"
	log.Printf("[ERROR] 用户 %d 执行 %s 操作失败: %v", userID, operation, err)

	fmt.Println("\n5. 生产环境日志 (JSON格式):")
	fmt.Println("-------------------------------")

	// 生产环境日志示例
	log.Printf(`{"level":"INFO","msg":"生产环境日志","order_id":"%s","amount":%.2f}`, "ORD-001", 99.99)
	log.Printf(`{"level":"ERROR","msg":"支付失败","order_id":"%s","error":"%v"}`, "ORD-002", errors.New("insufficient funds"))

	fmt.Println("\n6. 带字段的日志:")
	fmt.Println("-------------------------------")

	// 带字段的日志
	log.Printf("[INFO] [user_id=%d] [module=%s] 用户 %s 执行了 %s 操作", 12345, "auth", "alice", "login")
	log.Printf("[ERROR] [user_id=%d] [module=%s] 权限检查失败: %s", 12345, "auth", "access denied")

	fmt.Println("\n7. 复杂格式化示例:")
	fmt.Println("-------------------------------")

	// 复杂的格式化示例
	stats := map[string]interface{}{
		"requests": 1000,
		"errors":   5,
		"latency":  "25ms",
	}

	log.Printf("[INFO] 系统统计: 请求数=%d, 错误数=%d, 延迟=%s",
		stats["requests"], stats["errors"], stats["latency"])

	// 格式化时间
	now := time.Now()
	log.Printf("[INFO] 当前时间: %s (Unix: %d)", now.Format(time.RFC3339), now.Unix())

	fmt.Println("\n演示完成!")
}

func simulateError() error {
	return fmt.Errorf("模拟的业务错误: %w", errors.New("database connection failed"))
}
