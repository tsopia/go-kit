# HTTPServer Prometheus Observability 包

`httpserver/observability/prometheus` 提供请求计数与延迟统计能力，并通过显式注册的 `/metrics` 路由暴露指标。

## 设计约束

- 不向 `httpserver` core 自动注册 `/metrics`
- 中间件和 metrics 路由可以共享同一个 Collector
- 默认提供包级 Collector，适合快速接入

## 快速开始

```go
public := srv.Group("")
srv.Use(prometheus.Middleware())
prometheus.Register(public, prometheus.Config{
	Path: "/metrics",
})
```
