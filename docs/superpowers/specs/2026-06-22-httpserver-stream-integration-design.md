# httpserver SSE/WS 融入 Gin 封装 — 架构设计

- 日期：2026-06-22
- 分支：`worktree-feat+httpserver-stream-integration`
- 范围：A 类（流式连接与 Gin 中间件链的语义对齐）

## 1. 背景与目标

业务侧无论 SSE 还是 WebSocket 都依托于同一个 Gin 服务，期望与普通 HTTP
请求**共享同一套 ctx 与中间件链**（accesslog、鉴权、trace 等），并保持
**统一的可观测性**：连接建立、断开、错误都要有结构化记录。

经过架构 review，确认 httpserver 的分层与扩展机制是健康的，**中间件共享链路
已经打通**（见 §2）。本设计不重构骨架，只填补一个贯穿性缺陷：
**"在 `c.Next()` 之后做汇总"的观测型中间件，对流式连接的语义是错配的**，
而当前只有 Timeout 被正确豁免，观测层（AccessLog / prometheus / otel）没有
对应机制。

## 2. 现状分析（置信度标注）

### 2.1 中间件共享链路已打通（置信度：确定）

- `preset.NewProductionServer` 先 `srv.Use(Recovery/RequestID/TraceID/SecurityHeaders)`，
  再创建 `regularGroup` / `streamingGroup` 并 `SetGroups`（`preset/production.go:18-33`）。
- `SetGroups` 把两个 group 转成 `routeGroupSpec`（`server.go:445-450`）。当 group
  的 handler 前缀与 `engine.Handlers` 完全一致时，只保存 `basePath` + 本地 handler，
  注册路由时用 `s.engine.Group(basePath)` **延迟重建**，从而继承后续
  `srv.Use(auth)` 添加的 engine 级中间件（`server.go:62-96`）。
- 结论：只要用户在注册路由前 `srv.Use(authMiddleware)`，**auth 已经作用于
  SSE/WS**。`c.Set("user", u)` 写入的数据对 SSE/WS handler 可见（前提是
  SSE/WS 能读到 gin keys，见 §2.3 gap）。

### 2.2 连接级可观测已存在（置信度：确定）

- `logStreamEvent`（`logging.go:13-48`）在 SSE 注册（`server.go:231,243`）和
  WS 注册（`server.go:276,435`）处自动产生 `stream_connect` / `stream_disconnect`
  日志，断开日志含 `duration_ms` 与 `error`（`ctx.Err()`）。
- AccessLog 通过 `WithStreamLogConfig` 把自定义 `LoggerFunc` 注入
  `request.Context()`（`access_log.go:124-128`），`logStreamEvent` 从中取出
  同一个 logger（`logging.go:18-22`），实现日志器统一。

### 2.3 三个真实 Gap（置信度：确定）

**G1 — 观测型中间件对流式连接语义错配**

SSE/WS 的 handler 在 gin handler 内**同步阻塞**执行（`server.go:237` 的
`handler(stream)`、`server.go:431` 的 `handler(session)` 均为同步调用），因此
所有挂在 engine 上、在 `c.Next()` 之后做汇总的中间件，其 `c.Next()` 会一直
阻塞到连接断开：

| 中间件 | 普通请求 | 流式连接的错配 | 后果 |
|--------|---------|---------------|------|
| AccessLog | latency=请求耗时 | latency=整个连接时长，status/bytes_out 不准（`access_log.go:135-156`） | 多一条噪音 access_log（与 stream_disconnect 重复） |
| prometheus | duration 计入 sum | 一个 1 小时连接给 `http_request_duration_seconds_sum` 贡献 +3600s（`middleware.go:47-74`，导出见 `middleware.go:128-137`） | `sum/count` 平均延迟被严重拉高 |
| otel | span 覆盖请求，status 准确 | span 覆盖整个连接，`status>=500` 判定永远不触发（SSE 200/WS 101），无法反映断开原因（`otel/middleware.go:49-60`） | trace 失真 |
| Timeout | 正常 | 已通过 streamingGroup 不挂 Timeout 豁免（`preset/production.go:28-30`） | 无（正面范例） |

> 注：prometheus 当前导出的是 `http_requests_total`(counter) +
> `http_request_duration_seconds_sum`(累积 sum)，**不是直方图**
> （`middleware.go:117-137`）。流式污染的准确机制是"平均延迟被拉高"，
> 而非"破坏 p99"。

**G2 — WSSession 缺少 Get/GetString（API 不对称）**

`SSEStream` 有 `Get`/`GetString`（`sse.go:47-54, 88-102`），`WSSession` 没有
（`ws.go:76-85`）。鉴权用 `c.Set("user", u)` 时，WS handler 无法读取。

**G3 — 三层语义未文档化**

engine 中间件作用于全部路由、Timeout 仅 regularGroup、流式连接的可观测
模型，这些都未在 README/doc.go 说明，用户不知道现有机制已可用。

## 3. 核心架构方案：统一的"流式标记 + 观测层感知"

引入一个**贯穿性识别机制**，用一个标记统一治理三个观测型中间件，而非给
每个中间件单独打补丁。

### 3.1 流式标记（StreamingKey）

- 在 `utils` 包新增常量 `StreamingKey = "stream"`（与 `TraceIDKey` 同样的
  集中管理方式，`utils/trace.go` 已有先例）。
- 打标点（流式 handler 入口，在 `c.Next()` 链**内部**写入）：
  - SSE 的 ginHandler 第一行：`c.Set(utils.StreamingKey, "sse")`
  - WS 的 ginHandler 第一行：`c.Set(utils.StreamingKey, "ws")`
  - 导出 `middleware.MarkStreaming(transport string)`，供用户通过
    `srv.StreamingGroup()` 自定义的流式路由使用。
- 关键时序：标记在 `c.Next()` 链内写入，观测型中间件在 `c.Next()` 返回后
  读取 `c.GetString(utils.StreamingKey)` —— 天然契合，无竞态。
- 值用字符串 transport（`"sse"`/`"ws"`），既能判定"是否流式"，又能作为
  指标/日志的维度，避免再引入第二个布尔标记。

### 3.2 AccessLog 感知

`c.Next()` 之后若 `c.GetString(utils.StreamingKey) != ""`：跳过 `access_log`
与 `payload_log` 汇总条目（`access_log.go:133-161` 之后增加判断）。
`WithStreamLogConfig` 注入在 `c.Next()` **之前**（`access_log.go:124-128`），
不受影响 —— connect/disconnect 仍用统一 logger。

结果：流式连接稳定为 **2 条结构化日志**（connect + disconnect），消除噪音。

### 3.3 prometheus 感知 + 活跃连接 gauge

**(a) 不污染请求指标**：`c.Next()` 后若流式，**跳过** `observe(...)`
（`middleware.go:70-74` 之前增加判断），不计入 `http_requests_total` /
`http_request_duration_seconds_sum`。

**(b) 新增活跃连接 gauge**：`streaming_active_connections{transport="sse|ws"}`，
连接建立 +1、断开 -1。

依赖方向问题：gauge 埋点在 httpserver 核心包的 SSE/WS handler，但核心包
**不能反向依赖** `observability/prometheus` 子包。解决方案 —— 复用 §2.2 的
context 传播模式：

```
httpserver 定义抽象（不依赖任何 observability 实现）:
    type StreamObserver interface {
        OnConnect(transport string)
        OnDisconnect(transport string)
    }
  通过 request.Context() 传播（与 StreamLogConfig 同一注入点）。

prometheus.Middleware() 在 c.Next() 前把 collector 的 inc/dec 包装成
    StreamObserver 注入 ctx。

SSE/WS handler 在 connect 时 observer.OnConnect(transport)，
    disconnect 时 observer.OnDisconnect(transport)。
```

依赖方向保持 `子包 → 核心包`，不破坏分层。Collector 增加
`streamGauge map[string]int64` + mutex，`render()` 追加
`streaming_active_connections` 输出。

> 抽象归属：`StreamObserver` 与其 context 注入/读取放在 `middleware` 子包
> （与 `StreamLogConfig` 同包 `stream_log.go`），httpserver 核心包通过
> `logging.go` 已有的 `httpmiddleware` 依赖访问，方向与现状一致。

### 3.4 otel 感知

`c.Next()` 后若流式：给 span 加 attribute `stream.transport=sse|ws`，并
**不**用 `status>=500` 判定 error（`otel/middleware.go:58-60` 增加流式分支）。
`c.Errors` 非空时仍记录 error（保留 `otel/middleware.go:51-56` 行为）。

### 3.5 WSSession.Get / GetString（G2）

WS pump 在 goroutine 中运行，不能跨 goroutine 读 live `gin.Context.Keys`。
在 `WS()` 注册时（`server.go:262` 进入 ginHandler 后、启动 pump 前）**快照**
`c.Keys` 传入 `wsSession`：

```
// ws_session.go: wsSession 增加 keys map[string]any
func (s *wsSession) Get(key string) (any, bool) { v, ok := s.keys[key]; return v, ok }
func (s *wsSession) GetString(key string) (string, bool) { ... }
```

`WSSession` 接口（`ws.go:76-85`）增加 `Get`/`GetString`，与 `SSEStream` 对齐。
SSE 因同步持有 ginCtx，继续直读（`sse.go:88-102` 不变）。

### 3.6 文档（G3）

- `httpserver/README.md` + `doc.go`：三层模型 + 标准范式
  （`Use(auth)` → SSE/WS handler 内 `stream.Get("user")` / `session.Get("user")`）。
- 流式可观测说明：每个流式连接产生 2 条日志 + gauge ±1，不产生 access_log /
  请求延迟指标。
- 中间件顺序提示（RealIP/RequestID/TraceID 必须在 AccessLog/Recovery 之前）
  归入 B 类，本次仅在范式示例中隐式体现，不展开。

## 4. 文件变更清单

| 文件 | 操作 | 责任 |
|------|------|------|
| `utils/...`（trace.go 或新 const） | 修改 | 新增 `StreamingKey` 常量 |
| `httpserver/middleware/stream_log.go` | 修改 | 新增 `StreamObserver` 接口 + context 注入/读取 + `MarkStreaming` |
| `httpserver/middleware/access_log.go` | 修改 | `c.Next()` 后流式判断，跳过汇总条目 |
| `httpserver/server.go` | 修改 | SSE/WS ginHandler 打标 + connect/disconnect 调 observer |
| `httpserver/ws.go` | 修改 | `WSSession` 接口加 `Get`/`GetString` |
| `httpserver/ws_session.go` | 修改 | `wsSession` 加 `keys` 快照 + `Get`/`GetString` 实现 |
| `httpserver/observability/prometheus/middleware.go` | 修改 | 流式跳过 observe + Collector gauge + render |
| `httpserver/observability/prometheus/register.go` | 修改 | gauge 渲染辅助（如需要） |
| `httpserver/observability/otel/middleware.go` | 修改 | 流式 span attribute + 不误判 error |
| `httpserver/README.md` / `doc.go` | 修改 | 三层模型 + 范式文档 |
| 各对应 `*_test.go` | 创建/修改 | 覆盖每个改动点 |

每个 Task 控制在 ≤ 50 行实现改动，测试 : 实现 ≥ 1.5:1。

## 5. 测试策略

- **流式标记**：单测 `MarkStreaming` 写入正确 transport；SSE/WS 集成测试断言
  `c.GetString(StreamingKey)` 在中间件层可见。
- **AccessLog 跳过**：用 fake LoggerFunc 断言流式连接**不产生** `access_log`
  事件，但 `stream_connect`/`stream_disconnect` 仍产生且用同一 logger。
- **prometheus**：断言流式连接后 `http_requests_total` 不增长；
  `streaming_active_connections` 在连接期间为 1、断开后为 0。
- **otel**：用 in-memory exporter 断言 span 含 `stream.transport` attribute，
  且 SSE/WS 200/101 不被标记为 error。
- **WSSession.Get**：集成测试 —— 中间件 `c.Set("user", u)` 后，WS handler
  `session.Get("user")` 能取回同一对象。
- 回归：现有 `sse_test.go` / `ws_test.go` / `ws_integration_test.go` /
  observability 测试全绿。

## 6. 风险与边界

- **R1 函数指针比较**：`routeGroupSpec` 既有机制（`server.go:62-96`）不在本次
  改动范围，不引入新风险。
- **R2 StreamObserver 注入缺失**：若用户未挂 prometheus 中间件，observer 为
  nil，SSE/WS handler 需 nil-safe（无 observer 时跳过 ±1）。
- **R3 gauge 泄漏**：连接异常断开（panic）时 `OnDisconnect` 必须经 `defer`
  保证调用，避免 gauge 只增不减。WS 已有 pump panic recovery
  （`ws.go:50-60`），需确保 disconnect 埋点在 defer 链中。
- **R4 现网兼容**：流式连接不再产生 access_log 条目属于行为变更；用户已确认
  现网未消费该（不准的）条目，按"默认跳过、无开关"实现。
- **R5 范围纪律**：B 类（中间件顺序文档化、HandlerTimeout 归属、健康检查配置
  认知负担）与 C 类不在本次范围，仅记录待排期。

## 7. 不做的事（YAGNI）

- 不引入 `StreamContext` 公共接口（SSE/WS handler 业务逻辑无法共享，假想需求）。
- 不给 AccessLog 加 `SkipStreaming` 开关（默认跳过即可）。
- 不改 `RegisterModules` 走 regularGroup（会破坏模块内流式路由，仅文档说明）。
- 不新增 prometheus 连接总计数 counter（用户选择只加 active gauge）。
