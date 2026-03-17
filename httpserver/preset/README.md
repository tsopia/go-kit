# HTTPServer Preset 包

`httpserver/preset` 提供官方推荐的 HTTP server 装配方式，适合希望快速得到一致默认链路的项目。

## 设计约束

- 只做装配，不重复实现中间件
- 不自动注册 metrics、swagger、pprof
- 仍保留 `srv.Use(...)` 和 `srv.Group(...)` 的扩展空间

## 快速开始

```go
srv := preset.NewProductionServer(cfg)
```
