# HTTPServer OTel Observability 包

`httpserver/observability/otel` 提供 OpenTelemetry tracing 中间件，负责请求上下文中的 span 创建与传播。

## 设计约束

- 不在包内初始化 exporter
- 不修改 `httpserver` core 的默认行为
- 优先从请求上下文和 Header 中传播 trace 上下文

## 快速开始

```go
srv.Use(otel.Middleware(otel.Config{
	TracerName: "user-service",
}))
```
