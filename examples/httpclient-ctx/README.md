# HTTP Client Context 方法使用指南

本示例演示了 `httpclient` 包中 `Request.Context(ctx)` 的使用方式。

## 功能概述

`Context` 方法用于为 HTTP 请求设置 `context.Context`。它支持：

- ⏰ **超时控制** - 设置请求超时时间
- 🚫 **取消控制** - 主动取消正在进行的请求
- 📋 **值传递** - 在请求中传递trace ID、用户信息等上下文数据
- 🔗 **链式调用** - 与其他请求方法完美配合

## 方法签名

```go
func (r *Request) Context(ctx context.Context) *Request
```

## 使用场景

### 1. 设置请求超时

```go
// 创建5秒超时的context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// 使用 Context 设置超时
resp, err := client.NewRequest("GET", "https://api.example.com/data").
    Context(ctx).
    Do()
```

### 2. 主动取消请求

```go
ctx, cancel := context.WithCancel(context.Background())

// 在另一个goroutine中取消请求
go func() {
    time.Sleep(2 * time.Second)
    cancel() // 取消请求
}()

resp, err := client.NewRequest("GET", "https://api.example.com/slow").
    Context(ctx).
    Do()
```

### 3. 传递追踪信息

```go
// 创建带有追踪信息的context
ctx := context.WithValue(context.Background(), "trace_id", "abc123")
ctx = context.WithValue(ctx, "user_id", "user456")

resp, err := client.NewRequest("GET", "https://api.example.com/user").
    Context(ctx).
    Header("X-Trace-ID", "abc123").
    Do()

// 从context获取追踪信息
traceID := ctx.Value("trace_id")
userID := ctx.Value("user_id")
```

### 4. 链式调用

```go
resp, err := client.NewRequest("POST", "/api/users").
    Context(ctx).                          // 设置context
    Header("Content-Type", "application/json"). // 设置请求头
    JSON(userData).                        // 设置JSON请求体
    Timeout(10 * time.Second).            // 设置超时（会覆盖context的超时）
    Do()
```

## 最佳实践

### 1. 总是设置超时

```go
// ✅ 推荐：设置合理的超时时间
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := client.NewRequest("GET", url).Context(ctx).Do()
```

### 2. 正确处理取消

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

resp, err := client.NewRequest("GET", url).Context(ctx).Do()
if err != nil {
    if ctx.Err() == context.Canceled {
        fmt.Println("请求被主动取消")
    } else if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("请求超时")
    }
}
```

## 运行示例

```bash
cd examples/httpclient-ctx
go run main.go
```

`Context(ctx)` 让请求级超时、取消和上下文透传都保持在同一条链路里，和当前 `httpclient` 的 Context-first API 一致。
