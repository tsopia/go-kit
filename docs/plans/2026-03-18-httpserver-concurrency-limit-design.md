# HTTPServer Concurrency Limit Middleware Design

## 背景

当前 `httpserver/middleware` 已提供：

- `Recovery`
- `Timeout`
- `TraceID`
- `RequestID`
- `AccessLog`
- `Compression`
- `CORS`
- `SecurityHeaders`
- `MaxBodySize`

这些能力里，`Timeout` 负责单请求耗时边界，`Compression` 负责传输优化，`AccessLog` 负责请求观测，但还缺少一个“单进程承载保护”能力。

当服务遇到慢请求、下游阻塞或短时流量突刺时，即使 QPS 不高，也可能因为在途请求过多而把进程拖垮。这个问题更适合用并发闸门解决，而不是直接上带分布式语义的 `RateLimit`。

## 目标

为 `httpserver/middleware` 增加一个官方 `ConcurrencyLimit` 中间件，满足以下要求：

- 第一版只做单进程、全局并发闸门
- 超出上限立即拒绝，不排队
- 默认返回 `503 Service Unavailable`
- 不强绑定 JSON 响应体语义
- 与 `Recovery`、`Timeout`、`AccessLog` 稳定协作

## 非目标

- 不做按 IP / 用户 / API Key / 路由维度分桶
- 不做分布式并发控制
- 不做等待队列、公平调度或超时等待
- 不在第一版提供 metrics/tracing 集成
- 不改变 `httpserver.NewServer(...)` 的默认行为

## 设计

### 1. 能力边界

`ConcurrencyLimit` 放在 `httpserver/middleware`，而不是 `httpserver` core。

推荐默认策略：

- `httpserver` core：默认不开启
- `middleware.ConcurrencyLimit(...)`：显式挂载
- `preset.NewProductionServer(...)`：后续可作为推荐默认值
- `preset.NewDevelopmentServer(...)`：默认不挂载

这样可以保持 core 轻量，同时把“服务承载保护”作为一类明确的 middleware 能力暴露出来。

### 2. API 设计

第一版建议：

```go
type ConcurrencyLimitConfig struct {
    Limit      int
    OnRejected gin.HandlerFunc
}

func ConcurrencyLimit(limit int) gin.HandlerFunc
func ConcurrencyLimitWithConfig(config ConcurrencyLimitConfig) gin.HandlerFunc
```

默认行为：

- `Limit <= 0` 直接 `panic`，避免静默错误配置
- `OnRejected == nil` 时只返回状态码 `503`
- 如果设置了 `OnRejected`，由调用方负责写入响应

### 3. 语义模型

第一版语义是“单进程全局在途请求数上限”：

1. 请求进入中间件时尝试获取一个并发槽位
2. 获取成功则继续执行后续 handler
3. 获取失败则立即拒绝
4. 请求结束时释放槽位

这里的“结束”包括：

- 正常返回
- `Abort`
- `panic`
- `Timeout` 返回

最重要的约束是：**绝不能泄漏槽位**。

### 4. 实现方式

第一版建议用 `chan struct{}` 作为固定容量信号量：

- `make(chan struct{}, limit)` 表示最大并发数
- 非阻塞写入成功表示拿到槽位
- 非阻塞写入失败表示当前已满
- `defer` 释放槽位

选择这个方案的原因：

- Go 里实现最简单、最稳
- 并发边界比原子计数器更不容易出错
- 与“超限立即拒绝”的语义天然一致

不选原子计数器的原因：

- 容易在 `panic`、`abort`、提前返回等边界上漏回收
- 第一版没有必要为了极小的性能收益引入更脆弱的实现

### 5. 默认拒绝行为

超限时默认：

- 直接 `AbortWithStatus(http.StatusServiceUnavailable)`
- 不写 JSON body

这样做的原因：

- `middleware` 不应假设服务一定是 JSON API
- 纯状态码对 REST、静态资源、内部服务都成立
- 业务若有统一错误模型，可通过 `OnRejected` 自定义

例如：

```go
srv.Use(middleware.ConcurrencyLimitWithConfig(middleware.ConcurrencyLimitConfig{
    Limit: 100,
    OnRejected: func(c *gin.Context) {
        c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
            "code": "service_busy",
            "message": "service is busy",
        })
    },
}))
```

### 6. 推荐中间件顺序

推荐：

```go
srv.Use(middleware.AccessLog())
srv.Use(middleware.Recovery())
srv.Use(middleware.ConcurrencyLimit(100))
srv.Use(middleware.Timeout(2 * time.Second))
```

原因：

- `AccessLog` 放最外层，可以记录被拒绝的 `503`
- `Recovery` 放在 `ConcurrencyLimit` 外层，保证 panic 场景也能走完整恢复链
- `ConcurrencyLimit` 放在 `Timeout` 外层，表示“进入处理链就占一个槽位”，直到请求完整结束才释放
- `Timeout` 继续只负责单请求执行时长

### 7. 测试策略

重点覆盖以下行为：

- 低于上限时正常放行
- 超过上限时立即返回 `503`
- 已完成请求会释放槽位，后续请求可继续进入
- `OnRejected` 可覆盖默认响应
- handler `panic` 后槽位仍会释放
- 与 `Timeout` 叠加时，超时结束后槽位会释放

测试实现建议：

- 使用 channel 控制第一个请求阻塞
- 在其占住槽位时发起第二个请求，验证立即拒绝
- 释放第一个请求后，再发起第三个请求，验证可再次进入

## 结论

第一版 `ConcurrencyLimit` 应坚持：

- 单进程、全局闸门
- 超限立即 `503`
- 不排队、不分桶
- 用 `chan struct{}` 实现固定容量信号量
- 以“不泄漏槽位”为核心正确性约束

这样最符合当前 `httpserver/middleware` 的定位，也比直接做 `RateLimit` 更适合作为下一批官方中间件能力。
