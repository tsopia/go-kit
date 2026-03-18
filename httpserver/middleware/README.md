# HTTPServer Middleware 包

`httpserver/middleware` 提供可复用的 Gin 中间件，不依赖具体 metrics、tracing 或 auth 实现，适合作为 `httpserver` 的通用扩展层。

## 适用场景

- 需要 panic recovery
- 需要请求超时控制
- 需要结构化访问日志和可选 payload 调试日志
- 需要响应压缩
- 需要单进程全局并发闸门，超限立即返回 503，不排队
- 需要 Trace ID / Request ID 注入
- 需要基础 CORS 与安全响应头
- 需要限制请求体大小

## 快速开始

```go
srv := httpserver.NewServer(nil)
srv.Use(middleware.AccessLog())
srv.Use(middleware.Compression())
srv.Use(middleware.Recovery())
srv.Use(middleware.ConcurrencyLimit(100))
srv.Use(middleware.Timeout(2 * time.Second))
srv.Use(middleware.TraceID())
```

## ConcurrencyLimit

`ConcurrencyLimit` 用于控制单进程内的全局并发上限。当前请求数达到阈值时，新请求会直接返回 `503 Service Unavailable`，不会进入队列等待。

适合这些场景：

- 防止单实例被突发流量打满
- 保护下游数据库、RPC 或第三方接口
- 希望用简单的闸门而不是排队来削峰

## Timeout

`Timeout` 用于给单次请求链路注入协作式执行预算。它的职责是向 `context.Context` 传播 deadline，而不是强制终止正在运行的 goroutine。

这意味着：

- handler、service、repository 和下游 client 需要协作感知 `ctx.Done()`
- `Timeout` 不会主动杀掉 goroutine
- 当 `Timeout` 与 `ConcurrencyLimit` 组合使用时，槽位释放以“请求真正执行结束”为准，而不是以“客户端已经收到 `504`”为准

适合这些场景：

- 给入口请求设置统一执行预算
- 让下游数据库、RPC 或 HTTP client 复用同一个 deadline
- 与 `ConcurrencyLimit` 配合时，明确“执行结束”才释放并发槽位

## AccessLog

`AccessLog` 默认输出一条摘要访问日志，包含：

- `method`
- `path`
- `route`
- `status`
- `latency_ms`
- `client_ip`
- `request_id`
- `trace_id`
- `bytes_in`
- `bytes_out`

如果需要调试请求体和响应体，可以显式开启 payload capture：

```go
srv.Use(middleware.AccessLog(middleware.AccessLogConfig{
    CapturePayload: true,
    MaxBodyBytes:   8 << 10,
    Multipart: middleware.MultipartConfig{
        Mode: middleware.MultipartFormFieldsOnly,
    },
}))
```

默认行为：

- access log 始终输出摘要
- payload log 默认关闭
- JSON / form 会按字段脱敏
- multipart 默认按结构化 part 处理，不记录文件内容

## Compression

`Compression` 默认按 `Accept-Encoding` 协商响应压缩。第一版只支持 `gzip`，适合 JSON API、文本响应和较大的列表接口。

默认行为：

- 不会自动挂载到 `httpserver` core
- 在 `middleware` 层需要显式启用
- 在 `preset.NewProductionServer` 中后续可作为推荐默认值
- 小响应、`SSE`、下载和已压缩响应默认跳过
