# HTTPServer Timeout/Lifecycle Alignment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 收敛 `httpserver` 的超时与停机语义，修复 `preset` 扩展断裂、统一 `RunWithContext` / `WaitForShutdown` 的优雅关闭流程，并为 `HandlerTimeout` 与显式禁用超时建立稳定 contract。

**Architecture:** 保持 `Server`、`preset.NewProductionServer(...)`、`SetGroups(...)` 等公开 API 不变。对内把“分流路由组”从预先缓存的 `*gin.RouterGroup` 改成可重建的 group spec，让后续 `srv.Use(...)` 能影响未来注册的 regular/streaming 路由；同时把 graceful shutdown 收敛为一条内部流水线，并用显式 sentinel 支持“保留默认值”和“显式禁用超时”两种语义共存。

**Tech Stack:** Go 1.24、Gin、`net/http`、`context`、table-driven tests、`go test`、`golangci-lint`

---

## 前置检查

- [x] 已完整阅读相关代码文件：`httpserver/config.go`、`httpserver/server.go`、`httpserver/lifecycle.go`、`httpserver/readiness.go`、`httpserver/ws.go`、`httpserver/preset/production.go`、`httpserver/middleware/timeout.go`
- [x] 已完整阅读相关测试文件：`httpserver/server_test.go`、`httpserver/lifecycle_test.go`、`httpserver/integration_test.go`、`httpserver/ws_test.go`、`httpserver/ws_integration_test.go`、`httpserver/preset/production_test.go`、`httpserver/middleware/timeout_test.go`、`httpserver/middleware/concurrency_limit_test.go`
- [x] 已列出代码现状（每个判断附行号）

## 代码现状

- [x] **确定**：`preset.NewProductionServer(...)` 先创建 `regularGroup` / `streamingGroup`，再通过 `srv.SetGroups(...)` 缓存到 `Server`，后续 `srv.GET(...)` / `srv.SSE(...)` / `srv.WS(...)` 都走缓存 group，而 `srv.Use(...)` 只改根 `engine`，因此 `preset` 后续扩展无法影响未来注册的 helper 路由。证据：`httpserver/preset/production.go:14-33`、`httpserver/server.go:83-129`、`httpserver/server.go:364-384`
- [x] **确定**：Gin 的 `RouterGroup` 在创建时会复制父链 `Handlers`，不是动态引用父链，因此已创建 group 天生不会追随之后的 `engine.Use(...)`。证据：`github.com/gin-gonic/gin@v1.11.0/routergroup.go:72-77`、`github.com/gin-gonic/gin@v1.11.0/routergroup.go:241-247`
- [x] **确定**：`RunWithContext(...)` 在 `ctx.Done()` 后直接创建 `ShutdownTimeout` context 并调用 `Shutdown(...)`，不会先 `MarkDraining()`、不会等待 `DrainTimeout`、不会触发 `OnShuttingDown` / `OnShutdownComplete`。证据：`httpserver/server.go:431-441`、`httpserver/server.go:445-475`
- [x] **确定**：`HandlerTimeout` 和 `WriteTimeout` 的约束只写在注释里，没有进入 `Validate()`。证据：`httpserver/config.go:36-39`、`httpserver/config.go:97-139`
- [x] **确定**：`Normalize()` 对多个 timeout 字段采用 `<= 0` 回填默认值，导致调用方无法通过 `Config` 显式把这些字段设成 `0` 以表示禁用。证据：`httpserver/config.go:57-93`
- [x] **确定**：`Shutdown(ctx nil)` 当前总是调用 `context.WithTimeout(..., s.config.ShutdownTimeout)`；即使以后允许 `ShutdownTimeout == 0`，也会立刻超时，必须同步修正 shutdown context 的构造方式。证据：`httpserver/server.go:490-493`

## 方案比较

### 方案 A：只改文档，承认 preset 初始化后冻结

优点：

- 改动最小
- 不触碰 `Server` 内部结构

缺点：

- 与 `preset/README.md` 现有“保留 `srv.Use(...)` 和 `srv.Group(...)` 扩展空间”的承诺冲突
- 会把 helper API 变成“看起来像 Gin，实际是部分冻结模型”
- 不能解决 `RunWithContext` / `DrainTimeout` / `HandlerTimeout` / 显式禁用超时的真实设计缺口

### 方案 B：保留 API，重建内部 contract（推荐）

优点：

- 保住 `Gin-first` 与“可继续扩展”的定位
- 修正 `preset`、lifecycle、config 三条语义线
- 风险集中在内部 helper，不要求业务代码迁移

缺点：

- 需要新增 group spec / shutdown helper
- 必须明确“已创建的原生 `*gin.RouterGroup` 不会追溯继承后续 `srv.Use(...)`”

### 方案 C：直接做 `Server v2` / 改公开 API

优点：

- 语义最彻底
- 能彻底摆脱 Gin group 快照限制

缺点：

- 改动面过大
- 与当前仓库增量演进节奏不匹配
- 不符合本轮“修复现有设计断裂”的目标

**推荐：采用方案 B。**

## 可能否定本设计的技术约束

- [x] Gin 原生 `*gin.RouterGroup` 已创建后无法被框架方追溯补链，因此本轮只能保证“未来通过 `srv.GET(...)` / `srv.Group(...)` / `srv.SSE(...)` / `srv.WS(...)` 注册的路由”继承后续 `srv.Use(...)`；无法承诺“外部已拿到的 raw group 也动态补链”
- [x] `preset.NewProductionServer(...)` 公开返回值是 `*Server` 而不是 `(*Server, error)`，因此 `HandlerTimeout` / `WriteTimeout` 约束必须落在 `Config.Validate()` 的启动期校验，而不是构造期返回错误
- [x] `Config{}` 当前广泛依赖“零值补默认”，因此不能直接把所有 `0` 改成“显式禁用”；需要引入显式 sentinel 保持兼容

## 证伪证据

- [x] 如果 `RouterGroup` 不是快照而是动态引用父链，那么现有实现不应出现 `preset` 后续 `srv.Use(...)` 失效；Gin 源码已证伪该假设
- [x] 如果 `RunWithContext(...)` 已共享 `WaitForShutdown()` 的逻辑，那么 `server.go` 中不应存在一条独立的“直接 `Shutdown(...)`”路径；源码已证伪该假设
- [x] 如果 `0` 现在已经支持显式禁用，那么 `Normalize()` 不应对 `<= 0` 一律回填默认值；源码已证伪该假设

### Task 1: 锁定 preset 后续扩展 contract

**目标：** 先用失败测试锁定“`preset` 后续 `srv.Use(...)` 应影响未来 helper 路由，但 regular/streaming 的 timeout 分流仍保留”。

**文件变更：**
- 修改：`httpserver/preset/production_test.go:14-85`（预计 +35 行）
- 修改：`httpserver/server_test.go:827-945`（预计 +20 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

```go
func TestNewProductionServerLateUseAppliesToFutureHelperRoutes(t *testing.T) {
	t.Parallel()

	srv := NewProductionServer(&httpserver.Config{
		EnableHealthCheck: false,
		HandlerTimeout:    5 * time.Millisecond,
		WriteTimeout:      50 * time.Millisecond,
	})

	srv.Use(func(c *gin.Context) {
		c.Header("X-Late-Use", "1")
		c.Next()
	})

	srv.GET("/late", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	srv.SSE("/events", func(stream httpserver.SSEStream) {
		_ = stream.Event("ok", "1")
	})

	// regular route 应继承 late Use
	// streaming route 也应继承 late Use，但不应因此重新挂 HandlerTimeout
}
```

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver/preset ./httpserver -run 'TestNewProductionServerLateUseAppliesToFutureHelperRoutes|TestServer_SetGroups' -v`
Expected: FAIL，至少出现 `X-Late-Use` 缺失，证明当前缓存 group 截断了后续 `srv.Use(...)`

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

本 Task 不写实现，只锁定 contract。

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

本 Task 不执行，通过在 Task 2 完成。

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/preset/production_test.go httpserver/server_test.go
git commit -m "test(httpserver): lock preset late-use contract"
```

### Task 2: 用可重建 group spec 取代缓存 RouterGroup

**目标：** 保持 `SetGroups(...)` 签名不变，但把内部状态从缓存 group 改成“base path + local handlers”的可重建描述，让未来 helper 路由重新继承根 `engine` 的新增 middleware。

**文件变更：**
- 修改：`httpserver/server.go:23-42`（预计 +20 行）
- 修改：`httpserver/server.go:364-384`（预计 +30 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

复用 Task 1 的失败测试，不新增测试文件。

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver/preset ./httpserver -run 'TestNewProductionServerLateUseAppliesToFutureHelperRoutes|TestServer_SetGroups' -v`
Expected: FAIL，仍缺少 `X-Late-Use`

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

```go
type routeGroupSpec struct {
	basePath      string
	localHandlers gin.HandlersChain
	frozen        *gin.RouterGroup
}

func newRouteGroupSpec(root gin.HandlersChain, group *gin.RouterGroup) *routeGroupSpec {
	if group == nil {
		return nil
	}
	if len(group.Handlers) < len(root) {
		return &routeGroupSpec{frozen: group}
	}
	for i := range root {
		if reflect.ValueOf(root[i]).Pointer() != reflect.ValueOf(group.Handlers[i]).Pointer() {
			return &routeGroupSpec{frozen: group}
		}
	}
	return &routeGroupSpec{
		basePath:      group.BasePath(),
		localHandlers: append(gin.HandlersChain(nil), group.Handlers[len(root):]...),
	}
}

func (s *Server) buildGroup(spec *routeGroupSpec) *gin.RouterGroup {
	if spec == nil {
		return &s.engine.RouterGroup
	}
	if spec.frozen != nil {
		return spec.frozen
	}
	g := s.engine.Group(spec.basePath)
	if len(spec.localHandlers) > 0 {
		g.Use(spec.localHandlers...)
	}
	return g
}
```

并将 `SetGroups(...)`、`getRegularGroup()`、`getStreamingGroup()` 改成基于 spec 工作；保留 fallback 到 frozen group 的兼容路径。

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver/preset ./httpserver -run 'TestNewProductionServerLateUseAppliesToFutureHelperRoutes|TestServer_SetGroups' -v`
Expected: PASS

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/server.go httpserver/preset/production_test.go httpserver/server_test.go
git commit -m "refactor(httpserver): rebuild helper groups from live engine chain"
```

### Task 3: 锁定 RunWithContext 与 WaitForShutdown 的统一停机 contract

**目标：** 用失败测试表达 `RunWithContext(...)` 必须和 `WaitForShutdown()` 一样，先 draining、再等待 `DrainTimeout`、再 shutdown，并触发同一组 hooks。

**文件变更：**
- 修改：`httpserver/lifecycle_test.go:323-430`（预计 +45 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

```go
func TestRunWithContextUsesGracefulShutdownPipeline(t *testing.T) {
	t.Parallel()

	var hooks []string
	srv := NewServer(&Config{
		Host:            "127.0.0.1",
		Port:            0,
		DrainTimeout:    50 * time.Millisecond,
		ShutdownTimeout: time.Second,
	}, WithHooks(Hooks{
		OnShuttingDown: func(_ context.Context, _ LifecycleEvent) {
			hooks = append(hooks, "shutting_down")
		},
		OnShutdownComplete: func(_ context.Context, _ LifecycleEvent) {
			hooks = append(hooks, "shutdown_complete")
		},
	}))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := srv.RunWithContext(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunWithContext: %v", err)
	}
	if elapsed < srv.config.DrainTimeout {
		t.Fatalf("elapsed %v < drain timeout %v", elapsed, srv.config.DrainTimeout)
	}
	if got := srv.State(); got != StateStopped {
		t.Fatalf("state = %q, want %q", got, StateStopped)
	}
	if len(hooks) != 2 || hooks[0] != "shutting_down" || hooks[1] != "shutdown_complete" {
		t.Fatalf("hooks = %v, want [shutting_down shutdown_complete]", hooks)
	}
}
```

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestRunWithContextUsesGracefulShutdownPipeline|TestDrainTimeout' -v`
Expected: FAIL，`elapsed < DrainTimeout` 或 hooks 缺失，暴露当前 `RunWithContext(...)` 的旁路 shutdown

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

本 Task 不写实现，只锁定 contract。

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

本 Task 不执行，通过在 Task 4 完成。

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/lifecycle_test.go
git commit -m "test(httpserver): lock graceful shutdown pipeline"
```

### Task 4: 收敛 graceful shutdown 到单一内部 helper

**目标：** 让 `RunWithContext(...)` 与 `WaitForShutdown()` 走同一条 draining/shutdown 流程，并为 `ShutdownTimeout == 0` 预留正确语义。

**文件变更：**
- 修改：`httpserver/server.go:429-475`（预计 +35 行）
- 修改：`httpserver/lifecycle.go:31-49`（预计 +20 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

复用 Task 3 的失败测试，不新增测试文件。

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestRunWithContextUsesGracefulShutdownPipeline|TestDrainTimeout' -v`
Expected: FAIL

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

```go
func (s *Server) shutdownContext() (context.Context, context.CancelFunc) {
	if s.config.ShutdownTimeout <= 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
}

func (s *Server) gracefulShutdownSequence() error {
	if err := s.MarkDraining(); err != nil {
		return fmt.Errorf("mark draining: %w", err)
	}
	s.emitHook(s.hooks.OnShuttingDown, s.lifecycleEvent(nil))

	if s.config.DrainTimeout > 0 {
		time.Sleep(s.config.DrainTimeout)
	}

	ctx, cancel := s.shutdownContext()
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	s.emitHook(s.hooks.OnShutdownComplete, s.lifecycleEvent(nil))
	return nil
}
```

然后让 `RunWithContext(...)` 与 `WaitForShutdown()` 都调用 `gracefulShutdownSequence()`。

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestRunWithContextUsesGracefulShutdownPipeline|TestDrainTimeout' -v`
Expected: PASS

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/server.go httpserver/lifecycle.go httpserver/lifecycle_test.go
git commit -m "refactor(httpserver): unify graceful shutdown pipeline"
```

### Task 5: 收紧 HandlerTimeout 与 WriteTimeout 的配置约束

**目标：** 让 `HandlerTimeout` 的 contract 从注释变成可测试、可报错的配置校验。

**文件变更：**
- 修改：`httpserver/config.go:36-39`（预计 +10 行）
- 修改：`httpserver/config.go:121-139`（预计 +12 行）
- 修改：`httpserver/server_test.go:76-183`（预计 +25 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

```go
{
	name: "reject handler timeout >= write timeout",
	cfg: Config{
		Host:              "127.0.0.1",
		Port:              8080,
		ReadTimeout:       time.Second,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
		DrainTimeout:      time.Second,
		HealthCheckPath:   "/health",
		ReadinessPath:     "/readyz",
		LivenessPath:      "/livez",
		HandlerTimeout:    5 * time.Second,
	},
	wantErr: true,
}
```

另补一例：`WriteTimeout == 0 && HandlerTimeout > 0` 应允许通过。

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizeAndValidate' -v`
Expected: FAIL，因为当前 `Validate()` 不检查 `HandlerTimeout`

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

```go
if c.HandlerTimeout < 0 {
	return fmt.Errorf("%w: handler timeout must be >= 0", ErrInvalidConfig)
}
if c.HandlerTimeout > 0 && c.WriteTimeout > 0 && c.HandlerTimeout >= c.WriteTimeout {
	return fmt.Errorf("%w: handler timeout must be < write timeout", ErrInvalidConfig)
}
```

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizeAndValidate' -v`
Expected: PASS

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/config.go httpserver/server_test.go
git commit -m "test(httpserver): validate handler timeout contract"
```

### Task 6: 引入显式 DisableTimeout sentinel，解决“默认值”和“显式禁用”冲突

**目标：** 在不打破 `Config{}` 默认补全行为的前提下，支持显式关闭 transport/drain/shutdown timeout。

**文件变更：**
- 修改：`httpserver/config.go:8-14`（预计 +8 行）
- 修改：`httpserver/config.go:57-93`（预计 +25 行）
- 修改：`httpserver/config.go:121-139`（预计 +20 行）
- 修改：`httpserver/server_test.go:76-183`（预计 +25 行）
- 修改：`httpserver/lifecycle_test.go:323-430`（预计 +25 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 写失败测试**（完整代码）

```go
func TestConfigNormalizePreservesExplicitDisableTimeout(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ReadTimeout:       DisableTimeout,
		ReadHeaderTimeout: DisableTimeout,
		WriteTimeout:      DisableTimeout,
		IdleTimeout:       DisableTimeout,
		ShutdownTimeout:   DisableTimeout,
		DrainTimeout:      DisableTimeout,
	}

	cfg.Normalize()

	if cfg.ReadTimeout != 0 || cfg.WriteTimeout != 0 || cfg.DrainTimeout != 0 {
		t.Fatalf("disable sentinel should normalize to zero-valued timeout")
	}
}
```

并补一例：`Shutdown(nil)` / `RunWithContext(...)` 在 `ShutdownTimeout` 显式禁用后不应立刻返回 `context deadline exceeded`。

- [ ] **Step 2: 运行测试 → 确认失败**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizePreservesExplicitDisableTimeout|TestRunWithContextUsesGracefulShutdownPipeline' -v`
Expected: FAIL，因为当前 `Normalize()` 会把 sentinel 当成默认值处理，且 `ShutdownTimeout == 0` 会立即超时

- [ ] **Step 3: 写最简实现**（完整代码，行数检查）

```go
const DisableTimeout time.Duration = -1

func normalizeTimeout(v *time.Duration, def time.Duration) {
	switch *v {
	case DisableTimeout:
		*v = 0
	case 0:
		*v = def
	}
}

func validateTimeout(name string, v time.Duration) error {
	if v < DisableTimeout {
		return fmt.Errorf("%w: %s must be >= %v", ErrInvalidConfig, name, DisableTimeout)
	}
	return nil
}
```

把 `ReadTimeout`、`ReadHeaderTimeout`、`WriteTimeout`、`IdleTimeout`、`ShutdownTimeout`、`DrainTimeout` 全部切到该 helper；`shutdownContext()` 已在 Task 4 负责正确解释 `0`。

- [ ] **Step 4: 运行测试 → 确认通过**（命令 + 预期输出）

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestConfigNormalizePreservesExplicitDisableTimeout|TestConfigNormalizeAndValidate|TestRunWithContextUsesGracefulShutdownPipeline' -v`
Expected: PASS

- [ ] **Step 5: Commit**（完整命令）

```bash
git add httpserver/config.go httpserver/server_test.go httpserver/lifecycle_test.go httpserver/server.go httpserver/lifecycle.go
git commit -m "feat(httpserver): support explicit timeout disable sentinel"
```

### Task 7: 文档收尾与全量验证

**目标：** 同步文档，把新 contract 写清楚，并跑完 `httpserver` 相关验证。

**文件变更：**
- 修改：`httpserver/README.md:130-190`（预计 +30 行）
- 修改：`httpserver/README.md:565-700`（预计 +30 行）
- 修改：`httpserver/doc.go:40-57`（预计 +12 行）
- 修改：`httpserver/preset/README.md:5-15`（预计 +10 行）

**约束检查：**
- [ ] 修改后文件总行数 ≤ 50 行
- [ ] 依赖包已确认存在

**子步骤（TDD）：**
- [ ] **Step 1: 更新文档**

必须明确：

- `preset` 之后，未来通过 `srv.GET(...)` / `srv.Group(...)` / `srv.SSE(...)` / `srv.WS(...)` 注册的 helper 路由会继承后续 `srv.Use(...)`
- 已经创建并持有的原生 `*gin.RouterGroup` 仍遵循 Gin 快照语义，不会追溯补链
- `RunWithContext(...)` 与 `WaitForShutdown()` 共享 graceful shutdown pipeline
- `HandlerTimeout` 的有效约束
- `DisableTimeout` 的用途与配置示例

- [ ] **Step 2: 运行 focused verification**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

- [ ] **Step 3: 运行 targeted race check**

Run: `GOCACHE=/tmp/go-build go test -race ./httpserver -run 'TestNewProductionServerLateUseAppliesToFutureHelperRoutes|TestRunWithContextUsesGracefulShutdownPipeline|TestConfigNormalizeAndValidate' -count=1 -v`
Expected: PASS

- [ ] **Step 4: 运行 lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

- [ ] **Step 5: 运行可选全仓验证**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: 若命中已知 `sonic` 基线问题，明确标注“与本次改动无关”；否则 PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver docs/plans
git commit -m "refactor(httpserver): align timeout and shutdown contracts"
```
