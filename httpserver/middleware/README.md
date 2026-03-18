# HTTPServer Middleware 包

`httpserver/middleware` 提供可复用的 Gin 中间件，不依赖具体 metrics、tracing 或 auth 实现，适合作为 `httpserver` 的通用扩展层。

## 适用场景

- 需要 panic recovery
- 需要请求超时控制
- 需要结构化访问日志和可选 payload 调试日志
- 需要响应压缩
- 需要 Trace ID / Request ID 注入
- 需要基础 CORS 与安全响应头
- 需要限制请求体大小

## 快速开始

```go
srv := httpserver.NewServer(nil)
srv.Use(middleware.AccessLog())
srv.Use(middleware.Compression())
srv.Use(middleware.Recovery())
srv.Use(middleware.Timeout(2 * time.Second))
srv.Use(middleware.TraceID())
```

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
