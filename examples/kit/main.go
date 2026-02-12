package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tsopia/go-kit/kit"
)

func main() {
	// 示例1: 基本使用
	fmt.Println("=== 示例1: 基本使用 ===")
	basicUsage()

	// 示例2: 上下文提取
	fmt.Println("\n=== 示例2: 上下文提取 ===")
	contextExtraction()

	// 示例3: 格式化日志
	fmt.Println("\n=== 示例3: 格式化日志 ===")
	formattedLogging()

	// 示例4: 链式调用
	fmt.Println("\n=== 示例4: 链式调用 ===")
	chainedLogging()

	// 示例5: 堆栈跟踪
	fmt.Println("\n=== 示例5: 堆栈跟踪 ===")
	stackTraceDemo()

	// 示例6: 自定义 Context Keys
	fmt.Println("\n=== 示例6: 自定义 Context Keys ===")
	customContextKeys()

	fmt.Println("\n=== 所有示例完成 ===")
}

// basicUsage 基本使用示例
func basicUsage() {
	// 初始化 kit
	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatText,
	})

	ctx := context.Background()

	// 不同级别的日志
	kit.Debug(ctx, "调试信息", "detail", "something") // 不会输出（级别为 Info）
	kit.Info(ctx, "服务启动成功")
	kit.Warn(ctx, "配置项缺失，使用默认值", "config", "timeout")
	kit.Error(ctx, "数据库连接失败", "retry", 3)

	// 注意: Fatal 和 Panic 会终止程序，这里不演示
	// kit.Fatal(ctx, "致命错误") // 会 os.Exit(1)
	// kit.Panic(ctx, "严重错误") // 会 panic()
}

// contextExtraction 上下文提取示例
func contextExtraction() {
	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatJSON,
		ContextKeys: &kit.ContextKeys{
			Trace:   []string{"trace_id", "x-trace-id"},
			Request: []string{"request_id", "x-request-id"},
			User:    []string{"user_id"},
		},
	})

	// 创建带追踪信息的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "trace-abc-123")
	ctx = context.WithValue(ctx, "request_id", "req-xyz-456")
	ctx = context.WithValue(ctx, "user_id", "user-789")

	kit.Info(ctx, "处理用户请求", "action", "login")
	// 输出会自动包含 trace_id, request_id, user_id
}

// formattedLogging 格式化日志示例
func formattedLogging() {
	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatText,
	})

	ctx := context.Background()
	userID := 12345
	username := "zhangsan"
	ip := "192.168.1.1"

	// 格式化日志（便捷但不推荐）
	kit.Infof(ctx, "用户 %s (ID:%d) 从 %s 登录", username, userID, ip)

	// 结构化日志（推荐，便于检索和分析）
	kit.Info(ctx, "用户登录",
		"username", username,
		"user_id", userID,
		"ip", ip,
	)
}

// chainedLogging 链式调用示例
func chainedLogging() {
	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatText,
	})

	// 创建带追踪信息的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "chained-trace-001")

	// 使用 WithCtx 创建链式 logger
	logger := kit.WithCtx(ctx)

	// 多次记录使用相同的上下文
	logger.Info("开始处理订单")
	logger.Info("验证订单信息")
	logger.Info("扣减库存")
	logger.Info("创建支付记录")
	logger.Info("订单处理完成")
}

// stackTraceDemo 堆栈跟踪示例
func stackTraceDemo() {
	kit.Init(kit.Options{
		Level:     "info",
		Format:    kit.FormatJSON,
		AddCaller: true,
		StackTrace: kit.StackTraceConfig{
			Enabled:     true,
			Level:       kit.ErrorLevel,
			Depth:       10,
			SkipRuntime: true,
		},
	})

	ctx := context.Background()

	// Info 级别不会显示堆栈
	kit.Info(ctx, "普通信息")

	// Error 级别会显示调用者和堆栈
	kit.Error(ctx, "业务错误", "code", 500)
}

// customContextKeys 自定义 Context Keys 示例
func customContextKeys() {
	// 设置全局默认的 context keys
	kit.SetDefaultContextKeys(kit.ContextKeys{
		Trace:   []string{"uber-trace-id"},
		Request: []string{"x-request-id"},
		User:    []string{"user_id"},
		// 自定义字段
		Custom: map[string][]string{
			"session_id": {"session_id", "sid", "x-session-id"},
			"tenant_id":  {"tenant_id", "x-tenant-id"},
		},
	})

	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatJSON,
	})

	// 创建上下文（使用自定义的 key）
	ctx := context.Background()
	ctx = context.WithValue(ctx, "uber-trace-id", "uber-trace-001")
	ctx = context.WithValue(ctx, "x-request-id", "req-002")
	ctx = context.WithValue(ctx, "x-session-id", "sess-003")
	ctx = context.WithValue(ctx, "tenant_id", "tenant-004")

	kit.Info(ctx, "多租户请求", "action", "query")
	// 输出包含: uber-trace-id, x-request-id, session_id, tenant_id
}

// webhookDemo Webhook 通知示例（注释掉，因为需要真实的 webhook URL）
func webhookDemo() {
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		fmt.Println("跳过 webhook 演示（未设置 WEBHOOK_URL）")
		return
	}

	kit.Init(kit.Options{
		Level:  "info",
		Format: kit.FormatJSON,
	})

	// 添加飞书 webhook
	kit.AddWebhook(&kit.WebhookConfig{
		Name:   "feishu-alerts",
		URL:    webhookURL,
		Method: "POST",
		BuildPayload: func(ctx context.Context, r kit.LogRecord) map[string]interface{} {
			return map[string]interface{}{
				"msg_type": "text",
				"content": map[string]string{
					"text": fmt.Sprintf("@oncall [%s] %s", r.Caller, r.Message),
				},
			}
		},
		// 只发送 Error 及以上级别
		Filter: func(ctx context.Context, r kit.LogRecord) bool {
			return r.Level >= kit.ErrorLevel
		},
	})

	ctx := context.Background()

	// 这条不会触发 webhook
	kit.Info(ctx, "普通信息")

	// 这条会触发 webhook
	kit.Error(ctx, "服务异常", "error", "connection timeout")
}
