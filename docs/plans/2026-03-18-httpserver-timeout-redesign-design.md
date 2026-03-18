# HTTPServer Timeout Redesign Design

## 背景

当前 [`httpserver/middleware/timeout.go`](/Users/kj/projects/go-kit/.worktrees/httpserver-concurrency-limit/httpserver/middleware/timeout.go) 通过额外 goroutine 执行 `c.Next()`，外层在 `ctx.Done()` 后直接对同一个 `*gin.Context` 调用 `AbortWithStatus(http.StatusGatewayTimeout)`。

这带来两个问题：

- `go test -race` 已能复现 `gin.Context` 的并发访问 race
- `Timeout` 与 `ConcurrencyLimit` 的组合语义不清晰：超时响应返回后，真实业务执行是否还在继续，进而槽位是否应该释放，当前没有稳定 contract

用户已经确认：

- `context.Context` 只是取消信号，不是 goroutine 强制终止器
- `ConcurrencyLimit` 应该统计“真实仍在执行的 handler”
- `Timeout` 更应该表达“协作式执行预算”，而不是“强制中断执行”

## 目标

重构 `httpserver/middleware.Timeout`，使其具备以下语义：

- 作为协作式执行预算 middleware
- 不再并发操作同一个 `gin.Context`
- 与 `ConcurrencyLimit` 的 contract 稳定
- 对 ctx-aware handler 能正确形成 timeout 行为

## 非目标

- 不提供 goroutine 级别的强制终止能力
- 不尝试在 middleware 内实现真正的硬超时中断
- 不把 transport-level 超时（socket/read/write）混进 Gin middleware
- 不在这一轮重构里修改 `http.Server` 配置结构

## 设计

### 1. Timeout 的新语义

`Timeout` 只做一件事：给当前请求链注入一个 deadline。

建议语义：

1. `timeout <= 0` 时直接透传
2. 用 `context.WithTimeout(c.Request.Context(), timeout)` 派生新 ctx
3. 用新 ctx 替换 `c.Request`
4. 同步执行 `c.Next()`
5. `c.Next()` 返回后，如果：
   - `ctx.Err() == context.DeadlineExceeded`
   - 且响应尚未开始写出
   则返回 `504 Gateway Timeout`

这意味着：

- `Timeout` 不再自己起 goroutine
- `Timeout` 不再试图和下游 handler 并行争抢 `gin.Context`
- 超时预算只靠 `ctx` 协作传播

### 2. 协作式 contract

重构后的 `Timeout` 明确只支持“协作式超时”：

- handler 自己需要感知 `ctx.Done()`
- service / repository / downstream client 需要继续传递 `ctx`
- 如果代码不尊重 ctx，`Timeout` 不会也不应该强制终止它

这也是 Go 的标准语义：

- `ctx cancel` 发出停止信号
- 是否真正停止，取决于代码是否协作退出

### 3. 与 ConcurrencyLimit 的最终契约

`ConcurrencyLimit` 只统计真实执行中的 handler 数。

因此：

- 进入 handler 时占槽
- handler 正常返回时释放
- `panic` 展开时释放
- 不能因为 `ctx.Done()` 触发了，就提前释放

换句话说：

- “客户端已经拿到 504” 不等于 “业务执行已经结束”
- `ConcurrencyLimit` 只跟“执行是否结束”绑定

这要求 `Task 4` 中当前有争议的 timeout 测试方向被修正：

- 不能再验证“504 返回后立刻释放槽位”
- 应改成验证“ctx-aware handler 因 deadline 退出后，槽位释放”

### 4. 推荐分层

超时治理建议明确分层：

- 客户端 timeout：客户端最多愿意等多久
- `http.Server` 级超时：socket/read/write/header/idle 保护
- middleware `Timeout`：协作式执行预算
- DB / Redis / HTTP / RPC client timeout：更短的下游预算
- 长任务：改为异步 job，而不是同步 HTTP

这意味着，如果未来需要“无论 handler 是否配合，N 秒后一定回超时响应”，应该设计为单独的 transport-level 能力，而不是继续混在 Gin middleware 里。

### 5. 测试语义

Timeout 重构后，建议测试矩阵如下：

- ctx-aware handler：
  - 感知 `ctx.Done()`
  - 在 deadline 后主动退出
  - middleware 返回 `504`
- 非 ctx-aware handler：
  - 不应在这轮 middleware 测试里伪造“被强制打断”的假语义
  - 行为应通过文档明确为“不受强制保证”
- `ConcurrencyLimit + Timeout`：
  - 只测试 ctx-aware handler 在 deadline 后退出，随后槽位释放
- `ConcurrencyLimit + panic`：
  - 保持已有 panic 后释放测试

## 结论

`Timeout` 应从“并发抢写 504 的伪硬超时”收敛成“同步 deadline 注入的协作式超时”。

`ConcurrencyLimit` 的语义应保持简单：

- 统计真实执行中的 handler
- 槽位只在执行结束时释放

这样可以同时解决：

- `gin.Context` 并发安全问题
- `Timeout` 与 `ConcurrencyLimit` 的语义冲突

并把 transport-level 的硬超时问题留给后续单独设计。
