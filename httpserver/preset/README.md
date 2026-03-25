# HTTPServer Preset 包

`httpserver/preset` 提供官方推荐的 HTTP server 装配方式，适合希望快速得到一致默认链路的项目。

它的定位是 **transport baseline**，不是完整的生产框架。

## 设计约束

- 只做装配，不重复实现中间件
- 不自动注册 metrics、swagger、pprof
- 仍保留 `srv.Use(...)` 和 `srv.Group(...)` 的扩展空间

补充约束：

- `preset` 负责 regular/streaming helper 路由分流：普通 helper 路由可挂 `HandlerTimeout`，SSE/WS 这类流式 helper 路由不会自动挂该超时
- 在 `preset.NewProductionServer(...)` 返回后继续调用 `srv.Use(...)`，未来通过 `srv.GET(...)` / `srv.Group(...)` / `srv.SSE(...)` / `srv.WS(...)` 注册的 helper 路由会继承新增 middleware
- 已经提前创建并持有的原生 `*gin.RouterGroup` 仍遵循 Gin 的快照语义，不会被后续 `srv.Use(...)` 追溯补链

## 快速开始

```go
srv := preset.NewProductionServer(cfg)
```

默认链路如下：

- 共享 middleware：`Recovery`、`RequestID`、`TraceID`、`SecurityHeaders`
- regular helper 路由：`srv.GET(...)`、`srv.POST(...)`、`srv.Group(...)` 等；当 `cfg.HandlerTimeout > 0` 时，会额外挂 `Timeout(cfg.HandlerTimeout)`
- streaming helper 路由：`srv.SSE(...)`、`srv.WS(...)`、`srv.StreamingGroup(...)`；不会自动挂 `HandlerTimeout`

如果你需要 transport timeout 与协作式 handler timeout 同时生效，建议同时配置：

```go
cfg := &httpserver.Config{
	WriteTimeout:   30 * time.Second,
	HandlerTimeout: 5 * time.Second,
}

srv := preset.NewProductionServer(cfg)
```

约束：

- `HandlerTimeout` 只影响 `preset` 下的 regular helper 路由
- 当 `WriteTimeout > 0` 且 `HandlerTimeout > 0` 时，必须满足 `HandlerTimeout < WriteTimeout`
- 如果你直接使用 `srv.Engine()` 或原生 `*gin.RouterGroup` 注册路由，是否挂超时 middleware 由你自己决定

`preset` 不会自动帮你配置这些生产边界：

- `AccessLog`
- `RealIP`
- `CORS`
- 浏览器 `WebSocket Origin` allowlist
- 认证鉴权
- 限流
