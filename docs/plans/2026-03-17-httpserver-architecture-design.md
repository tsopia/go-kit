# httpserver 架构演进设计

## 背景

当前 `httpserver` 的定位已经比较清晰：

- 基于 Gin 的轻量 HTTP server 封装
- `Server` 负责生命周期、监听、优雅关闭和健康检查
- `Engine()` 直接暴露 Gin 能力，不试图隐藏框架
- `Handle` / `HandleJSON` 提供可选的 typed handler 契约

从实现和文档看，这个包的核心价值不是“提供一套完整 Web 框架”，而是“提供稳定、可复用、易于 AI 理解的 HTTP 传输层底座”。继续把 metrics、tracing、auth、限流、调试、文档、默认日志等能力直接堆进 `Server`，会让这个边界迅速失控。

## 目标

- 保持 `httpserver` 的 Gin-first、轻封装定位
- 把 `Server core` 定义为生命周期内核，而不是功能集合
- 为团队提供一套官方推荐装配方式，但不把这些能力强塞进 core
- 为后续的 observability、middleware、preset 留出稳定扩展边界
- 让架构可以渐进演进，而不是一次性重写

## 非目标

- 不把 `httpserver` 扩展成强约束的工程框架
- 不在 core 中内置具体日志、metrics、tracing、auth 实现
- 不在 `Server` 中引入新的业务路由 DSL
- 不把统一业务响应 envelope 绑定到 `Server core`
- 不优先解决自动 TLS、热更新、零停机重启这类部署侧问题

## 方案比较

### 方案 1：纯轻量 core

只保留 `Server` 生命周期、健康检查、typed handler、路由注册，其余一概由业务项目自行实现。

优点：

- 边界非常清晰
- 不容易演变成框架
- 对现有实现侵入最小

缺点：

- 每个项目都要重复拼装 recovery、timeout、access log、metrics
- 团队实践容易漂移
- AI 生成代码时缺少官方默认路径

### 方案 2：厚 core

把 recovery、metrics、tracing、access log、安全头、限流、调试能力都塞进 `httpserver` 主包。

优点：

- 开箱即用
- 业务项目接入成本低

缺点：

- 与当前“轻量 Gin 封装”的定位冲突
- 很快形成难以裁剪的能力集合
- 具体实现会被日志、监控、追踪生态绑定

### 方案 3：分层 core + 官方装配

保留一个薄的 `Server core`，再提供独立的 middleware、observability 和 preset 层。

优点：

- 能保住 core 的边界
- 能给团队提供统一默认方案
- 能按子包管理依赖与演进节奏

缺点：

- 包结构比现在更复杂
- 需要更严格的职责划分

## 核心结论

采用方案 3。

推荐分层如下：

```text
httpserver/
├── config.go
├── server.go
├── lifecycle.go
├── health.go
├── readiness.go
├── handler.go
├── module.go
├── options.go
├── errors.go
├── doc.go
├── README.md
├── middleware/
├── observability/
├── preset/
└── swagger/
```

其中：

- `httpserver`：只放 core
- `httpserver/middleware`：放与具体观测后端无关的通用 Gin 中间件
- `httpserver/observability`：放 Prometheus、OpenTelemetry 这类集成
- `httpserver/preset`：放团队官方推荐装配
- `httpserver/swagger`：继续保持子包模式

## 分层职责

### 1. `httpserver` core

只负责：

- `Config` 默认化与校验
- 主监听与健康监听
- 启动、停止、优雅关闭
- readiness / liveness 状态管理
- `RouteModule` 与 typed handler
- 生命周期 hooks

明确不负责：

- access log
- metrics exporter
- tracing exporter
- rate limit
- auth
- compression
- pprof / debug endpoint

### 2. `httpserver/middleware`

放“通用但不依赖具体后端”的中间件：

- `Recovery`
- `Timeout`
- `TraceID`
- `RequestID`
- `CORS`
- `SecurityHeaders`
- `MaxBodySize`
- `AccessLog`

设计约束：

- 不自动注册路由
- 不依赖具体 metrics/tracing 后端
- 通过显式 `srv.Use(...)` 或路由组方式挂载

### 3. `httpserver/observability`

放与具体可观测性系统耦合的能力：

- `observability/prometheus`
- `observability/otel`

设计约束：

- 路由显式注册，不隐式挂 `/metrics`
- exporter、provider、meter/tracer 配置显式传入
- 不回灌到 core

### 4. `httpserver/preset`

只负责组装，不引入新的底层能力。

例如：

- `NewProductionServer(...)`
- `ApplyProductionDefaults(...)`

它应当基于 `core + middleware + observability` 组合，而不是把具体逻辑重新拷贝一遍。

## Server Core API 设计

### Config

`Config` 只承载可序列化、可校验、可从配置文件或环境变量注入的参数：

```go
type Config struct {
    Host              string
    Port              int
    ReadTimeout       time.Duration
    ReadHeaderTimeout time.Duration
    WriteTimeout      time.Duration
    IdleTimeout       time.Duration
    MaxHeaderBytes    int

    ShutdownTimeout   time.Duration
    DrainTimeout      time.Duration

    EnableHealthCheck bool
    HealthCheckPath   string
    ReadinessPath     string
    LivenessPath      string
    HealthCheckPort   int
}
```

新增：

- `Normalize()`
- `Validate() error`

原则：

- Host、Port、Timeout、Path 进入 `Config`
- logger、meter、tracer、回调函数不进入 `Config`

### Option

`Option` 只承载代码层注入和运行时装配：

- `WithHooks(...)`
- `WithModules(...)`
- `WithBaseContext(...)`
- `WithConnContext(...)`
- `WithErrorLog(...)`
- `WithManualReadiness()`
- `WithHTTPServerMutator(...)`

明确不建议新增：

- `WithHost(...)`
- `WithPort(...)`
- `WithReadTimeout(...)`

这些会和 `Config` 重叠，破坏配置模型的一致性。

### 保持 Gin-first

`Engine()`、`Use()`、`Group()`、`GET/POST/...` 应继续保留为一等入口。

不引入新的路由元数据 DSL，不把 `Server` 变成一套脱离 Gin 的抽象层。

## 生命周期与状态机

推荐状态机：

```text
new -> starting -> ready -> draining -> stopping -> stopped
                      \
                       -> failed
```

语义：

- `new`：已构造，未监听
- `starting`：校验配置并建立 listener
- `ready`：对外接流量
- `draining`：停止接收新流量，等待在途请求
- `stopping`：执行 `Shutdown(ctx)`
- `stopped`：正常关闭
- `failed`：出现不可恢复错误

默认策略建议为“自动 ready”，但关闭自动 ready 应通过 `WithManualReadiness()` 这类 `Option` 控制，而不是依赖 `Config` 中的布尔零值。

### 启动流程

```text
Normalize/Validate
-> 创建 listener
-> 构造主 server / health server
-> 注册 health/readiness/liveness
-> state=starting
-> emit OnStarting
-> 启动 serve goroutine
-> 自动或手动切换 ready
-> emit OnStarted
```

### 关闭流程

```text
收到信号或显式 Shutdown
-> state=draining
-> readiness 返回 503
-> 等待 DrainTimeout
-> state=stopping
-> 调用 http.Server.Shutdown(ctx)
-> 关闭 health server
-> state=stopped
-> emit OnShutdownComplete
```

## 健康、Readiness 与 Liveness

建议将现有健康检查能力细分为三类端点：

- `/health`
  - 兼容历史调用
  - 默认等价于 readiness 语义
- `/readyz`
  - 只有 `ready` 返回 200
  - `starting`、`draining`、`stopping`、`failed` 返回 503
- `/livez`
  - 进程可存活则返回 200
  - `stopped`、`failed` 返回非 200

`HealthCheckManager` 更适合参与 readiness，而不是 liveness。

## Core 中应该优先补的能力

### 1. 统一启动路径

当前 `Start`、`Run`、`RunTLS`、`Serve` 存在重复流程，未来继续扩展只会增加分支漂移风险。应收敛为一套内部启动流水线。

### 2. Config 校验

当前缺少显式 `Validate()`，应该在启动前拦截：

- 非法端口
- 负超时
- 空路径
- `HealthCheckPort == Port` 等冲突场景

### 3. ReadHeaderTimeout

相比 buffer 类配置，它更接近传输层内核，也更值得优先纳入 `Config`。

### 4. Readiness / Draining

优雅关闭不应只是 `Shutdown()`，而应把“摘流”建模为一等能力。

## 不建议进入 Core 的内容

- Prometheus metrics
- OpenTelemetry tracing
- Access log
- Rate limiting
- Auth middleware
- Compression
- pprof / expvar
- 自动 TLS
- 热更新
- Zero-downtime reload

这些能力要么依赖具体外部系统，要么属于运维/部署问题，不适合作为 `Server core` 默认职责。

## 渐进演进路径

### 阶段 0：内部整理，不改外部 API

- 拆分 `server.go`
- 引入 `Config.Normalize()` / `Validate()`
- 收敛 `Start/Run/RunTLS/Serve` 公共流程

### 阶段 1：引入状态机与 readiness/liveness

- 增加 `State`
- 默认自动 ready，并通过 `WithManualReadiness()` 支持手动控制
- 增加 `DrainTimeout`
- 新增 `/readyz` 与 `/livez`

### 阶段 2：抽出 `middleware`

- 迁移现有 `TraceIDMiddleware`、`RequestIDMiddleware`、`CORSMiddleware`
- 新增 `Recovery`、`Timeout`、`MaxBodySize`、`SecurityHeaders`
- 在主包保留兼容包装器

### 阶段 3：补 `observability` 与 `preset`

- 新增 `observability/prometheus`
- 新增 `observability/otel`
- 新增 `preset`

## 测试与验证要求

- 保持 table-driven tests
- 为状态机、readiness、draining、健康端口模式补单测与集成测试
- 为兼容包装器补回归测试
- 创建新包时遵循仓库约束：
  - 先更新 `.ai/capabilities.yaml`
  - 再创建包目录、`doc.go`、`README.md`
  - 最后更新 `AGENTS.md` 中的“库能力速查表”

## 风险与取舍

- 风险：包结构变深，初期理解成本上升
  - 取舍：用清晰职责边界换取长期可维护性
- 风险：主包保留兼容包装器会暂时存在双入口
  - 取舍：优先保证平滑迁移，后续再逐步弱化旧入口
- 风险：readiness 进入 core 后需要重新梳理关闭语义
  - 取舍：这是值得的，因为它本来就是生命周期问题

## 结论

`httpserver` 不应该继续横向堆功能，而应该纵向分层：

- `core` 只做生命周期和协议基础
- `middleware` 承载通用 HTTP 横切能力
- `observability` 承载具体观测生态集成
- `preset` 承载团队默认装配

这条路径既能保住当前包的定位，也能为团队提供稳定的生产化演进空间。
