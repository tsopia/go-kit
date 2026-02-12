# slog 包重构计划

**状态**: ✅ 已完成

## 实施结果

已创建新的 `kit` 包替换旧的 `logger` 和 `slog` 包。

### 完成的文件

| 文件 | 说明 |
|------|------|
| `kit/options.go` | 配置选项定义（Level, Format, ContextKeys, WebhookConfig, StackTraceConfig） |
| `kit/context.go` | 上下文信息提取器 |
| `kit/stack.go` | 堆栈跟踪功能 |
| `kit/webhook.go` | Webhook 通知管理 |
| `kit/logger.go` | 核心 Logger 实现 |
| `kit/global.go` | 包级便捷函数 |
| `kit/logger_test.go` | 单元测试（table-driven） |
| `kit/README.md` | 使用文档 |

### 删除的文件
- `slog/` - 旧 slog 包
- `logger/` - 旧 zap 包

## 目标
简化并强化 slog 包，保留标准库 slog 作为基础，添加以下增强功能。

## 需求分析与实现方案

### 1. 可配置的 Context Key 提取

**设计：混合方案（全局默认 + 可覆盖）**

```go
package slog

// ContextKeys 定义要从 context 中提取的 keys
 type ContextKeys struct {
    Trace   []string // 默认: ["trace_id", "x-trace-id", "X-Trace-Id"]
    Request []string // 默认: ["request_id", "x-request-id", "X-Request-Id"]
    User    []string // 可选: ["user_id", "userId"]
    // 支持自定义扩展
    Custom  map[string][]string // {"session_id": ["session_id", "sid"]}
}

// 全局默认配置
var defaultContextKeys = ContextKeys{
    Trace:   []string{"trace_id", "x-trace-id", "X-Trace-Id", "traceId"},
    Request: []string{"request_id", "x-request-id", "X-Request-Id", "requestId"},
}

// 全局设置
func SetDefaultContextKeys(keys ContextKeys)

// Options 中支持覆盖
type Options struct {
    ContextKeys ContextKeys  // 如果不设置，使用全局默认值
    // ...
}
```

**优点：**
- 开箱即用：零配置也能正常工作
- 灵活：支持不同服务使用不同 key（如某些服务用 `X-Request-ID`）
- 可扩展：支持业务自定义字段（如 `tenant_id`, `device_id`）

**示例：**
```go
// 场景1：使用默认值
log.Info(ctx, "用户登录")  // 自动提取 trace_id, request_id

// 场景2：公司统一使用 X-Request-ID
log.SetDefaultContextKeys(log.ContextKeys{
    Trace:   []string{"X-Trace-ID"},
    Request: []string{"X-Request-ID"},
})

// 场景3：某个服务需要特殊配置
serviceLogger := log.New(log.Options{
    ContextKeys: log.ContextKeys{
        Trace: []string{"uber-trace-id"},  // Jaeger 格式
    },
})
```

---

### 2. Error 级别 Webhook 通知

**设计：**

```go
package slog

// WebhookConfig Webhook 配置
type WebhookConfig struct {
    // 必填：接收地址
    URL string

    // 可选：请求方法，默认 POST
    Method string

    // 可选：额外 headers
    Headers map[string]string

    // 可选：自定义 payload 构建函数
    // 返回的 map 会与错误信息合并
    BuildPayload func(ctx context.Context, record LogRecord) map[string]interface{}

    // 可选：超时设置，默认 5s
    Timeout time.Duration

    // 可选：过滤函数，只有返回 true 才发送
    Filter func(ctx context.Context, record LogRecord) bool
}

// LogRecord 日志记录信息
type LogRecord struct {
    Level      string
    Message    string
    Time       time.Time
    TraceID    string
    RequestID  string
    Caller     string           // 调用位置
    StackTrace string           // 堆栈信息
    Fields     map[string]interface{} // 其他字段
}
```

**默认 Payload 结构：**
```json
{
  "level": "ERROR",
  "message": "数据库连接失败",
  "time": "2024-01-15T10:30:00Z",
  "trace_id": "abc-123",
  "request_id": "req-456",
  "caller": "service/user.go:42",
  "stack_trace": "...",
  // 用户自定义内容
  "mention": "@oncall",
  "channel": "#alerts"
}
```

**示例：**
```go
// 企业微信机器人
log.SetWebhook(&log.WebhookConfig{
    URL: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx",
    BuildPayload: func(ctx context.Context, r log.LogRecord) map[string]interface{} {
        return map[string]interface{}{
            "msgtype": "markdown",
            "markdown": map[string]string{
                "content": fmt.Sprintf("**错误报警** @oncall\n> 消息: %s\n> 位置: %s",
                    r.Message, r.Caller),
            },
        }
    },
})

// 飞书机器人
log.SetWebhook(&log.WebhookConfig{
    URL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
    BuildPayload: func(ctx context.Context, r log.LogRecord) map[string]interface{} {
        return map[string]interface{}{
            "msg_type": "text",
            "content": map[string]string{
                "text": fmt.Sprintf("@user-id 错误: %s", r.Message),
            },
        }
    },
})

// 使用
log.Error(ctx, "数据库连接失败", "retry", 3)
// 自动发送 webhook
```

---

### 3. 默认初始化 + 包级函数

**设计：**

```go
package slog

// 全局默认 logger
var std *Logger

// Init 初始化全局 logger
func Init(opts Options) {
    std = New(opts)
}

// 包级函数 - 自动使用全局 std
func Debug(ctx context.Context, msg string, fields ...any)
func Info(ctx context.Context, msg string, fields ...any)
func Warn(ctx context.Context, msg string, fields ...any)
func Error(ctx context.Context, msg string, fields ...any)
func Fatal(ctx context.Context, msg string, fields ...any)

// 支持纯字符串（无 context，向后兼容）
func Debug(msg string, fields ...any)
func Info(msg string, fields ...any)
// ...
```

**关键问题：ctx 参数的位置**

有两种 API 设计，请确认你倾向哪种：

**设计 A：ctx 作为第一个参数（Go 标准做法）**
```go
log.Info(ctx, "用户登录", "user_id", 123)
```

**设计 B：无 ctx 时自动使用 context.Background()**
```go
// 有 ctx
log.InfoCtx(ctx, "用户登录", "user_id", 123)

// 无 ctx
log.Info("用户登录", "user_id", 123)
```

**设计 C：ctx 放入 fields（类似 zap.With）**
```go
log.Info("用户登录", log.Ctx(ctx), "user_id", 123)
// 或
log.Info("用户登录", "user_id", 123, log.Ctx(ctx))
```

**我的建议：设计 A**，理由：
1. 符合 Go 1.21+ slog 标准：`slog.InfoContext(ctx, msg)`
2. 强制开发者考虑 context 传递
3. API 清晰无歧义

---

### 4. 支持字符串拼接或 Key-Value

**设计：**

```go
// Key-Value 风格（推荐，结构化）
log.Info(ctx, "用户登录", "user_id", 123, "ip", "1.2.3.4")

// 字符串拼接（便捷但不推荐）
log.Infof(ctx, "用户 %d 从 %s 登录", 123, "1.2.3.4")

// 混合：Info 支持单个字符串作为消息
log.Info(ctx, fmt.Sprintf("用户 %d 登录", 123))  // fields 为空
```

**内部实现：**
```go
func Info(ctx context.Context, msg string, fields ...any) {
    if len(fields) == 0 {
        std.log(ctx, LevelInfo, msg)
        return
    }
    // 解析 fields 为 slog.Attr
    attrs := parseFields(fields)
    std.logAttrs(ctx, LevelInfo, msg, attrs)
}
```

---

### 5. 移除文件管理

**当前功能删除清单：**
- [ ] `DefaultLogFile` / `DefaultLogDir` 全局变量
- [ ] `SetDefaultLogFile` / `SetDefaultLogDir` 函数
- [ ] `GetDefaultLogPath` 函数
- [ ] `CleanupLogFiles` / `CleanupLogFile` 函数
- [ ] `EnsureLogDir` / `EnsureLogDirForPath` 函数
- [ ] `isDirectoryNotEmpty` 函数
- [ ] `EnableFileOutput` 选项
- [ ] `Rotate` 配置

**保留：**
- `io.Writer` 接口支持，用户可自行传入文件

**输出目标简化：**
```go
type Options struct {
    // 输出目标，默认 os.Stdout
    // 可以设置为 io.MultiWriter(os.Stdout, file)
    Output io.Writer

    // 移除：EnableFileOutput, Rotate
}
```

---

### 6. 堆栈跟踪支持

**设计：**

```go
type Options struct {
    // 是否启用调用者信息（文件:行号）
    AddCaller bool  // 默认 true

    // 堆栈跟踪配置
    StackTrace StackTraceConfig
}

type StackTraceConfig struct {
    // 是否启用
    Enabled bool  // 默认 true for Error+

    // 最小级别触发堆栈
    Level Level  // 默认 Error

    // 堆栈深度
    Depth int  // 默认 32

    // 是否过滤 runtime 帧
    SkipRuntime bool  // 默认 true
}
```

**输出示例：**
```json
{
  "time": "2024-01-15T10:30:00+08:00",
  "level": "ERROR",
  "msg": "数据库查询失败",
  "caller": "service/user.go:42",
  "stack": [
    "service/user.go:42",
    "handler/auth.go:88",
    "router/router.go:156"
  ]
}
```

**实现要点：**
- 使用 `runtime.Caller` 获取调用位置
- 使用 `runtime.Stack` 或 `debug.Stack()` 获取堆栈
- 过滤掉 slog 包内部的帧
- 只保留项目根目录下的帧（通过 `runtime.Caller` 的 `ok` 判断）

---

## 新 API 完整示例

```go
package main

import (
    "context"
    "os"

    "github.com/tsopia/go-kit/slog"
)

func main() {
    // 1. 初始化（通常在 main 或 init 中）
    slog.Init(slog.Options{
        Level:     slog.InfoLevel,
        Format:    slog.FormatJSON,
        AddCaller: true,
        StackTrace: slog.StackTraceConfig{
            Enabled: true,
            Level:   slog.ErrorLevel,
        },
        ContextKeys: slog.ContextKeys{
            Trace:   []string{"trace_id", "x-trace-id"},
            Request: []string{"request_id"},
        },
        Webhook: &slog.WebhookConfig{
            URL: os.Getenv("ALERT_WEBHOOK"),
            BuildPayload: func(ctx context.Context, r slog.LogRecord) map[string]interface{} {
                return map[string]interface{}{
                    "text": fmt.Sprintf("@oncall %s: %s", r.Caller, r.Message),
                }
            },
        },
    })

    // 2. 使用
    ctx := context.WithValue(context.Background(), "trace_id", "abc-123")
    ctx = context.WithValue(ctx, "request_id", "req-456")

    // 结构化日志
    slog.Info(ctx, "服务启动", "port", 8080)
    // 输出: {"time":"...","level":"INFO","msg":"服务启动","trace_id":"abc-123","request_id":"req-456","port":8080,"caller":"main.go:32"}

    // 格式化日志
    slog.Infof(ctx, "用户 %d 登录", 12345)

    // 错误日志（自动触发 webhook，带堆栈）
    slog.Error(ctx, "数据库连接失败", "error", "timeout")
    // 输出: {"time":"...","level":"ERROR","msg":"数据库连接失败","trace_id":"abc-123","error":"timeout","caller":"db/conn.go:45","stack":["db/conn.go:45", "service/user.go:88"]}
    // 同时发送 webhook: @oncall db/conn.go:45: 数据库连接失败
}
```

---

## 待确认问题

1. **ctx 参数位置**：设计 A（ctx 第一参数）vs 设计 B（Ctx 后缀）vs 设计 C（放入 fields）？
2. **Context Key 配置**：混合方案（全局+可覆盖）是否符合预期？
3. **Webhook**：是否需要支持多个 webhook 地址（如同时发送到企业微信和飞书）？
4. **Fatal 处理**：是否保留 `os.Exit(1)` 行为？
5. **包名**：是否从 `slog` 改为 `log` 避免与标准库冲突？（建议保持 `slog`，用户使用时用别名 `gklog` 或 `kitlog`）

请确认以上问题，我将开始实施重构。
