# HTTPServer Middleware 包

`httpserver/middleware` 提供可复用的 Gin 中间件，不依赖具体 metrics、tracing 或 auth 实现，适合作为 `httpserver` 的通用扩展层。

## 适用场景

- 需要 panic recovery
- 需要请求超时控制
- 需要 Trace ID / Request ID 注入
- 需要基础 CORS 与安全响应头
- 需要限制请求体大小

## 快速开始

```go
srv := httpserver.NewServer(nil)
srv.Use(middleware.Recovery())
srv.Use(middleware.Timeout(2 * time.Second))
srv.Use(middleware.TraceID())
```
