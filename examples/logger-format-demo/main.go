package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/tsopia/go-kit/pkg/logger"
)

func main() {
	fmt.Println("=== Go-Kit Logger 格式化功能演示 ===")
	fmt.Println()

	// 创建不同配置的日志记录器
	devLogger := logger.NewDevelopment()
	prodLogger := logger.NewProduction()

	fmt.Println("1. 基本格式化日志方法:")
	fmt.Println("-------------------------------")

	// 基本格式化方法
	devLogger.Debugf("这是一个调试消息: %s", "debug info")
	devLogger.Infof("用户 %s (ID: %d) 登录成功", "alice", 12345)
	devLogger.Warnf("内存使用率达到 %.2f%%", 85.5)
	devLogger.Errorf("数据库连接失败: %v", errors.New("connection timeout"))

	fmt.Println("\n2. 对比结构化日志方法:")
	fmt.Println("-------------------------------")

	// 对比：结构化日志方法
	devLogger.Info("用户登录成功",
		"user", "alice",
		"user_id", 12345,
		"ip", "192.168.1.1",
		"timestamp", time.Now(),
	)

	// 对比：格式化日志方法
	devLogger.Infof("用户 %s (ID: %d) 从 %s 在 %s 登录成功",
		"alice", 12345, "192.168.1.1", time.Now().Format("2006-01-02 15:04:05"))

	fmt.Println("\n3. 全局格式化函数:")
	fmt.Println("-------------------------------")

	// 设置全局日志记录器
	logger.SetDefaultLogger(devLogger)

	// 使用全局格式化函数
	logger.Debugf("全局调试: %s", "system starting")
	logger.Infof("服务启动在端口 %d", 8080)
	logger.Warnf("配置项 %s 使用默认值: %v", "timeout", 30*time.Second)
	logger.Errorf("初始化失败: %v", errors.New("config not found"))

	fmt.Println("\n4. 性能优化 - 级别检查:")
	fmt.Println("-------------------------------")

	// 设置为Info级别，Debug不会输出
	devLogger.SetLevel(logger.InfoLevel)

	// 这个Debug消息不会被处理，因为级别不够
	devLogger.Debugf("这个消息不会被处理: %s %d %v",
		"expensive operation", 999999, map[string]interface{}{"key": "value"})

	fmt.Printf("当前日志级别: %s\n", devLogger.GetLevel().String())
	fmt.Printf("Debug级别是否启用: %v\n", devLogger.IsEnabled(logger.DebugLevel))

	fmt.Println("\n5. 错误处理演示:")
	fmt.Println("-------------------------------")

	// 模拟各种错误情况
	err := simulateError()
	if err != nil {
		devLogger.Errorf("操作失败: %v", err)
	}

	// 带上下文的错误
	userID := 12345
	operation := "update_profile"
	devLogger.Errorf("用户 %d 执行 %s 操作失败: %v", userID, operation, err)

	fmt.Println("\n6. 生产环境日志 (JSON格式):")
	fmt.Println("-------------------------------")

	// 生产环境日志会输出到文件，这里只是演示
	prodLogger.Infof("生产环境日志: 订单 %s 处理完成，金额: %.2f", "ORD-001", 99.99)
	prodLogger.Errorf("支付失败: 订单 %s, 错误: %v", "ORD-002", errors.New("insufficient funds"))

	fmt.Println("\n7. 带字段的格式化日志:")
	fmt.Println("-------------------------------")

	// 创建带字段的日志记录器
	userLogger := devLogger.WithFields(map[string]interface{}{
		"user_id": 12345,
		"module":  "auth",
	})

	userLogger.Infof("用户 %s 执行了 %s 操作", "alice", "login")
	userLogger.Errorf("权限检查失败: %s", "access denied")

	fmt.Println("\n8. 复杂格式化示例:")
	fmt.Println("-------------------------------")

	// 复杂的格式化示例
	stats := map[string]interface{}{
		"requests": 1000,
		"errors":   5,
		"latency":  "25ms",
	}

	devLogger.Infof("系统统计: 请求数=%d, 错误数=%d, 延迟=%s",
		stats["requests"], stats["errors"], stats["latency"])

	// 格式化时间
	now := time.Now()
	devLogger.Infof("当前时间: %s (Unix: %d)", now.Format(time.RFC3339), now.Unix())

	fmt.Println("\n演示完成!")
}

func simulateError() error {
	return fmt.Errorf("模拟的业务错误: %w", errors.New("database connection failed"))
}
