# kit - 结构化日志库

基于 Go 标准库 `log/slog` 的日志封装，支持上下文提取、webhook 通知和堆栈跟踪。

## 特性

- **标准库兼容**: 基于 `log/slog`，与 Go 生态无缝集成
- **上下文感知**: 自动从 context 提取 trace_id, request_id, user_id
- **Webhook 通知**: 支持 Error 级别自动触发 webhook（企业微信、飞书等）
- **堆栈跟踪**: 可配置的调用者信息和堆栈跟踪
- **灵活配置**: 全局默认 + 实例级覆盖的 context key 配置

## 快速开始

```go
package main

import (
    "context"
    "github.com/tsopia/go-kit/kit"
)

func main() {
    // 初始化
    kit.Init(kit.Options{
        Level:  "info",
        Format: kit.FormatJSON,
    })

    // 使用
    ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
    kit.Info(ctx, "服务启动", "port", 8080)
    // 输出: {"time":"...","level":"INFO","msg":"服务启动","trace_id":"abc-123","port":8080}
}
```

## API 概览

### 日志级别

```go
kit.Debug(ctx, "调试信息", "key", "value")
kit.Info(ctx, "普通信息")
kit.Warn(ctx, "警告信息")
kit.Error(ctx, "错误信息")
kit.Fatal(ctx, "致命错误")  // 输出后 os.Exit(1)
kit.Panic(ctx, "严重错误")  // 输出后 panic()
```

### 格式化日志

```go
kit.Infof(ctx, "用户 %d 登录", userID)
kit.Errorf(ctx, "操作失败: %v", err)
```

### 链式调用

```go
logger := kit.WithCtx(ctx)
logger.Info("步骤1")
logger.Info("步骤2")
```

## 配置选项

### 基本配置

```go
kit.Init(kit.Options{
    Level:      "info",             // 日志级别: debug, info, warn, error, fatal, panic
    Format:     kit.FormatJSON,     // 格式: JSON 或 Text
    Output:     os.Stdout,          // 输出目标
    AddCaller:  true,               // 是否添加调用者信息
    TimeFormat: time.RFC3339,       // 时间格式
})
```

### Context Key 配置

```go
// 方式1: 全局设置（推荐）
kit.SetDefaultContextKeys(kit.ContextKeys{
    Trace:   []string{"trace_id", "x-trace-id", "X-Trace-Id"},
    Request: []string{"request_id", "x-request-id"},
    User:    []string{"user_id", "userId"},
})

// 方式2: 实例级设置
logger := kit.New(kit.Options{
    ContextKeys: &kit.ContextKeys{
        Trace: []string{"uber-trace-id"},
        Custom: map[string][]string{
            "tenant_id": {"tenant_id", "x-tenant-id"},
        },
    },
})
```

### Webhook 配置

```go
// 添加 webhook 通知
kit.AddWebhook(&kit.WebhookConfig{
    Name: "feishu-alert",
    URL:  "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    BuildPayload: func(ctx context.Context, r kit.LogRecord) map[string]interface{} {
        return map[string]interface{}{
            "msg_type": "text",
            "content": map[string]string{
                "text": fmt.Sprintf("@oncall %s: %s", r.Caller, r.Message),
            },
        }
    },
    // 可选：自定义过滤
    Filter: func(ctx context.Context, r kit.LogRecord) bool {
        return r.Level >= kit.ErrorLevel  // ErrorLevel, FatalLevel, PanicLevel
    },
})
```

### 堆栈跟踪配置

```go
kit.Init(kit.Options{
    StackTrace: kit.StackTraceConfig{
        Enabled:     true,              // 启用堆栈跟踪
        Level:       kit.ErrorLevel,    // Level 类型: ErrorLevel, FatalLevel, PanicLevel
        Depth:       32,                // 堆栈深度
        SkipRuntime: true,              // 跳过 runtime 帧
    },
})
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/tsopia/go-kit/kit"
)

func main() {
    // 1. 初始化
    kit.Init(kit.Options{
        Level:      "info",              // 字符串级别: debug, info, warn, error, fatal, panic
        Format:     kit.FormatJSON,
        AddCaller:  true,
        StackTrace: kit.StackTraceConfig{
            Enabled: true,
            Level:   kit.ErrorLevel,      // StackTraceConfig.Level 仍是 Level 类型
        },
        ContextKeys: &kit.ContextKeys{
            Trace:   []string{"trace_id", "x-trace-id"},
            Request: []string{"request_id"},
        },
    })

    // 2. 添加告警 webhook
    kit.AddWebhook(&kit.WebhookConfig{
        Name: "alerts",
        URL:  os.Getenv("WEBHOOK_URL"),
        BuildPayload: func(ctx context.Context, r kit.LogRecord) map[string]interface{} {
            return map[string]interface{}{
                "text": fmt.Sprintf("@oncall [%s] %s", r.Caller, r.Message),
            }
        },
    })

    // 3. 使用
    ctx := context.Background()
    ctx = context.WithValue(ctx, "trace_id", "abc-123")
    ctx = context.WithValue(ctx, "request_id", "req-456")

    kit.Info(ctx, "服务启动", "port", 8080)

    // Error 会自动触发 webhook
    kit.Error(ctx, "数据库连接失败", "error", "timeout")
}
```

## 日志格式示例

### JSON 格式

```json
{
  "time": "2024-01-15T10:30:00+08:00",
  "level": "ERROR",
  "msg": "数据库连接失败",
  "trace_id": "abc-123",
  "request_id": "req-456",
  "error": "timeout",
  "caller": "db/conn.go:45",
  "stack_trace": "db/conn.go:45\nservice/user.go:88"
}
```

### Text 格式

```
time=2024-01-15T10:30:00+08:00 level=ERROR msg="数据库连接失败" trace_id=abc-123 request_id=req-456 caller=db/conn.go:45
```

## 与标准库 slog 的关系

本包基于 `log/slog` 实现，你可以获取底层的 `slog.Logger`：

```go
logger := kit.New(kit.Options{})
slogLogger := logger.GetSlog()
```

也可以与标准库 slog 混合使用：

```go
// 使用本包记录业务日志
kit.Info(ctx, "业务日志")

// 使用标准库记录系统日志
slog.Info("系统日志")
```

## 最佳实践

1. **始终传递 context**: `kit.Info(ctx, msg, fields...)`
2. **结构化优于格式化**: 优先使用 `kit.Info(ctx, msg, "key", value)` 而非 `kit.Infof`
3. **配置全局默认值**: 在 main 函数中配置一次 `kit.SetDefaultContextKeys`
4. **Error 级别用于可恢复错误**: 使用 `kit.Error` 记录错误但继续执行
5. **Fatal 仅用于不可恢复错误**: 如配置加载失败、数据库连接失败等
