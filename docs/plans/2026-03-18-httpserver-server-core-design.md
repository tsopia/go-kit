# HTTPServer Server Core Design

## 背景

当前 `httpserver` 已经完成了 `middleware`、`observability`、`preset`、`typed handler` 等外围能力的分层，但 `Server core` 本身仍有几处结构性问题：

- 启动路径重复：[`Serve()`](/Users/kj/projects/go-kit/httpserver/server.go#L132)、[`Start()`](/Users/kj/projects/go-kit/httpserver/server.go#L163)、[`Run()`](/Users/kj/projects/go-kit/httpserver/server.go#L202)、[`RunTLS()`](/Users/kj/projects/go-kit/httpserver/server.go#L237) 共享大量重复逻辑。
- `DrainTimeout` 是死配置：定义在 [`Config`](/Users/kj/projects/go-kit/httpserver/config.go#L19) 中，但在 [`WaitForShutdown()`](/Users/kj/projects/go-kit/httpserver/server.go#L289) 和 [`Shutdown()`](/Users/kj/projects/go-kit/httpserver/server.go#L314) 里没有真正生效。
- `IsRunning()` 语义不可靠：当前只看 `s.server != nil`，见 [`server.go`](/Users/kj/projects/go-kit/httpserver/server.go#L357)，服务停止后仍可能返回 `true`。
- 状态机过于宽松：[`setState()`](/Users/kj/projects/go-kit/httpserver/readiness.go#L22) 只是直接赋值，没有合法迁移约束。
- hooks 传递能力弱：[`emitHook()`](/Users/kj/projects/go-kit/httpserver/lifecycle.go#L43) 永远传 `context.Background()`，无法表达更精确的生命周期上下文。
- 部分 server 级 API 有误导性，例如启动后修改健康检查路径的语义并不和已注册路由强一致。

这些问题暂时没有阻断 `httpserver` 使用，但已经开始影响 core 的可维护性和后续演进成本。

## 目标

本轮只聚焦 `Server core`，目标是：

- 把 `Server` 收敛成稳定的生命周期内核
- 让现有配置项，尤其是 `DrainTimeout`，变成真实行为
- 收紧状态机和关闭语义
- 在不破坏现有公开 API 的前提下，消除启动路径重复
- 为后续高级接入方补最小的低层扩展点

## 非目标

本轮不做这些事：

- 不引入新的业务中间件或 observability 能力
- 不重新设计 Gin 路由抽象
- 不把 `httpserver` core 做成更重的框架
- 不扩张到 Auth、RateLimit、ProxyHeaders 等非 core 能力
- 不大改 `Config` 成嵌套层级结构

## 推荐方案

推荐采用“生命周期内核重构”方案：

1. 对外保留 `Serve/Start/Run/RunTLS/Shutdown/WaitForShutdown` 这些 API。
2. 对内把启动流程统一收敛到一条内部主路径，例如 `startWithListener(...)`。
3. 通过更明确的状态机和 shutdown/drain 语义，让 `State()` 成为状态真相。
4. 在 `Option` 层补少量低层扩展点，而不是继续扩张 `Config` 字段。

这是在保持兼容的前提下，回报最高、风险可控的路线。

## 设计

### 1. 启动路径统一

内部应统一成一条主流程，大致拆成：

- `validateConfig()`
- 建立主 listener
- 构造 `http.Server`
- 构造健康检查 listener / server
- 触发 `OnStarting`
- 启动 main/health serve
- 切换 `ready`
- 触发 `OnStarted`

对外方法关系：

- `Serve(ln)`：复用统一流程，只是跳过 `net.Listen`
- `Start()`：内部 listen 后非阻塞启动
- `Run()`：内部 listen 后阻塞启动
- `RunTLS()`：内部 listen 后改用 `ServeTLS`

这样后续再改启动逻辑时，不需要四处复制修改。

### 2. 状态机与真状态

建议保留现有状态集合：

- `new`
- `starting`
- `ready`
- `draining`
- `stopping`
- `stopped`
- `failed`

但不再允许任意直接赋值，而是通过内部迁移函数，例如：

```text
new -> starting -> ready -> draining -> stopping -> stopped
                  \-> failed
starting -> failed
stopping -> failed
```

重点约束：

- `MarkReady()` 不能从 `stopped/failed` 直接进入 `ready`
- `MarkDraining()` 只能从 `ready` 进入
- `Shutdown()` 应保证进入 `stopping -> stopped` 或 `failed`

`State()` 应继续作为公开 API 保留，并被文档明确为运行状态的唯一真相。

### 3. 真正落地 DrainTimeout

当前 `DrainTimeout` 没有真实行为。本轮建议明确 shutdown 流程：

1. 收到关闭信号
2. `MarkDraining()`
3. readiness 立即返回 `503`
4. 等待 `DrainTimeout`
5. 再执行 `Shutdown(ctx)`
6. 进入 `stopping -> stopped`

语义拆分：

- `DrainTimeout`：给负载均衡摘流和请求自然退出的窗口
- `ShutdownTimeout`：控制 `http.Server.Shutdown()` 的最长时间

这样 `DrainTimeout` 和 `ShutdownTimeout` 职责明确，不再互相混淆。

### 4. IsRunning 与状态 API

当前 `IsRunning()` 只看 `s.server != nil`，语义不可靠。本轮建议：

- 文档主路径改为推荐 `State()`
- `IsRunning()` 要么重新定义为：
  - `starting`
  - `ready`
  - `draining`
  - `stopping`
  时返回 `true`
- 要么保留兼容但标注文档弱化

推荐直接收正语义，而不是继续保留一个误导性的 helper。

### 5. hooks 收紧

`Hooks` 仍然只服务生命周期，但建议增强：

- 至少保证 `OnShuttingDown` / `OnShutdownComplete` 能拿到更贴近真实流程的上下文
- 中期建议补：
  - `OnReady`
  - 或统一的 `OnStateChange`

本轮为了控制范围，可以先不扩 `Hooks` 结构，只把内部触发时机和语义统一。

### 6. 低层扩展点

本轮建议补一个受控逃生口：

```go
func WithHTTPServerMutator(fn func(*http.Server)) Option
```

它的作用是：

- 给高级接入方调 `http.Server` 低层能力
- 避免为了单个高级需求扩张 `Config`
- 不把 core 公开面做得过大

后续如果确实需要，再补：

- `WithBaseContext(...)`
- `WithConnContext(...)`
- `WithErrorLog(...)`

### 7. 建议弱化/废弃的 API

本轮建议开始在文档里弱化这些入口：

- `IsRunning()`：推荐改用 `State()`
- `SetHealthCheckPath()` / `GetHealthCheckPath()`：限制为启动前配置使用
- 主包里的 legacy middleware helper：继续兼容，但文档主路径改为 `httpserver/middleware`

这不是立即删除，而是逐步收紧边界。

## 推荐新增的小 API

本轮只建议新增一个小而稳的 API：

```go
func (s *Server) HealthAddr() string
```

价值：

- 明确返回健康检查监听地址
- 比通过 hook event 或内部字段间接推断更稳定

`Ready() bool`、`Listening() bool` 这类 helper 暂时不急。

## 测试策略

重点补这些测试：

- `DrainTimeout` 生效：draining 后 readiness 立即变 `503`，然后才进入 stopped
- `IsRunning()` 与 `State()` 的一致性
- `Serve/Start/Run/RunTLS` 经过统一主流程后，hooks 触发顺序不变
- `HealthAddr()` 在共享端口和独立健康端口下都正确

## 分阶段落地顺序

推荐按以下顺序实施：

1. 收敛启动路径
2. 真正落地 `DrainTimeout`
3. 收正状态与 `IsRunning()` 语义
4. 补 `HealthAddr()` 和 `WithHTTPServerMutator(...)`
5. 更新 README 和测试

## 结论

`httpserver` 当前最值得做的 core 优化，不是继续横向加功能，而是把 `Server` 收敛成稳定的生命周期内核。

这轮建议的重点是：

- 启动路径统一
- `DrainTimeout` 活起来
- 状态机变严格
- `State()` 成为唯一真相
- 给高级使用方补最小低层扩展点

这样既不会把 `httpserver` 做厚，也能显著降低后续演进成本。
