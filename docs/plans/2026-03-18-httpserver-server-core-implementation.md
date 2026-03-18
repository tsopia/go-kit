# HTTPServer Server Core Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 收敛 `httpserver` 的 `Server core`，统一启动路径，真正落地 `DrainTimeout`，收正状态与低层扩展点语义。

**Architecture:** 保持 `Serve/Start/Run/RunTLS/Shutdown/WaitForShutdown` 现有公开 API 不变，对内收敛到统一 lifecycle pipeline。通过更严格的状态机和真实的 drain/shutdown 流程，让 `State()` 成为运行状态真相，并补一个受控的 `http.Server` 低层 mutator option。

**Tech Stack:** Go 1.24、Gin、`net/http`、`context`、table-driven tests

---

### Task 1: 先补设计约束相关文档与回归测试清单

**Files:**
- Modify: `httpserver/README.md`
- Modify: `httpserver/server_test.go`
- Modify: `httpserver/lifecycle_test.go`

**Step 1: Write the failing tests**

先补/改测试，锁定这几个现状问题：

- `IsRunning()` 在 `Shutdown()` 后不应继续返回 `true`
- `DrainTimeout` 应影响 shutdown 流程，而不是死配置
- `HealthAddr()` 暂时不存在，先用 TODO/失败断言形式表达预期

测试应使用 table-driven tests，并尽量沿用现有 `httpserver` 测试风格。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestIsRunning|TestDrainTimeout|TestHealthAddr' -v`
Expected: FAIL，至少暴露当前 `IsRunning`/`DrainTimeout` 语义缺口

**Step 3: Commit**

```bash
git add httpserver/README.md httpserver/server_test.go httpserver/lifecycle_test.go
git commit -m "test(httpserver): lock server core lifecycle contract"
```

### Task 2: 收敛启动路径到统一内部主流程

**Files:**
- Modify: `httpserver/server.go`
- Modify: `httpserver/lifecycle.go`
- Test: `httpserver/server_test.go`
- Test: `httpserver/lifecycle_test.go`

**Step 1: Write the failing test**

补一个最小测试，锁定：

- `Serve/Start/Run/RunTLS` 走统一 lifecycle 行为
- `OnStarting` / `OnStarted` 的触发顺序保持不变

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestServerLifecycleHooks|TestServerStartPaths' -v`
Expected: FAIL 或需要修改现有测试以适配更严格 contract

**Step 3: Write minimal implementation**

重构内部结构：

- 在 `lifecycle.go` 增加统一 helper，例如：
  - `startWithListener(...)`
  - `startListening(...)`
  - `startBlocking(...)`
- `Serve/Start/Run/RunTLS` 只保留各自差异部分
- 不修改公开方法签名

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestServerLifecycleHooks|TestServerStartPaths' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/lifecycle.go httpserver/server_test.go httpserver/lifecycle_test.go
git commit -m "refactor(httpserver): unify server startup pipeline"
```

### Task 3: 真正落地 DrainTimeout 与 draining 语义

**Files:**
- Modify: `httpserver/server.go`
- Modify: `httpserver/readiness.go`
- Test: `httpserver/lifecycle_test.go`
- Test: `httpserver/health_server_test.go`

**Step 1: Write the failing test**

补测试表达以下行为：

- `WaitForShutdown()` 收到信号后先 `MarkDraining()`
- readiness 立即返回 `503`
- 等待 `DrainTimeout` 后才进入 `Shutdown()`
- 最终进入 `StateStopped`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestDrainTimeout|TestReadinessDuringDraining' -v`
Expected: FAIL，因为当前 draining 后会直接 shutdown

**Step 3: Write minimal implementation**

实现：

- 在 `WaitForShutdown()` 中显式等待 `DrainTimeout`
- 保持 `ShutdownTimeout` 只控制 `http.Server.Shutdown()`
- 不把 `DrainTimeout` 混入 `Shutdown(ctx)` 内部默认 ctx

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestDrainTimeout|TestReadinessDuringDraining' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/readiness.go httpserver/lifecycle_test.go httpserver/health_server_test.go
git commit -m "feat(httpserver): apply drain timeout during shutdown"
```

### Task 4: 收紧状态机并收正 IsRunning 语义

**Files:**
- Modify: `httpserver/readiness.go`
- Modify: `httpserver/server.go`
- Test: `httpserver/server_test.go`
- Test: `httpserver/lifecycle_test.go`

**Step 1: Write the failing test**

补测试锁定：

- 不允许明显非法状态跳转
- `IsRunning()` 与 `State()` 保持一致
- `Shutdown()` 之后 `IsRunning()` 为 `false`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestIsRunning|TestServerStateTransitions' -v`
Expected: FAIL，因为当前 `IsRunning()` 语义过宽，状态跳转也无约束

**Step 3: Write minimal implementation**

实现：

- 用内部 `transitionTo(...)` 代替直接赋值
- `IsRunning()` 改为基于 `State()` 判断
- `MarkReady()` / `MarkDraining()` 经过迁移校验

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestIsRunning|TestServerStateTransitions' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/readiness.go httpserver/server.go httpserver/server_test.go httpserver/lifecycle_test.go
git commit -m "refactor(httpserver): tighten server state transitions"
```

### Task 5: 增加 HealthAddr 与 http.Server 低层 mutator

**Files:**
- Modify: `httpserver/options.go`
- Modify: `httpserver/lifecycle.go`
- Modify: `httpserver/server.go`
- Test: `httpserver/server_test.go`
- Test: `httpserver/health_server_test.go`

**Step 1: Write the failing test**

补测试锁定：

- `HealthAddr()` 在共享端口/独立健康端口下返回正确地址
- `WithHTTPServerMutator(...)` 能在启动前修改底层 `http.Server`

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHealthAddr|TestHTTPServerMutator' -v`
Expected: FAIL，因为 API 尚不存在

**Step 3: Write minimal implementation**

实现：

- 新增 `WithHTTPServerMutator(func(*http.Server)) Option`
- 在构造 `http.Server` 后应用 mutator
- 新增 `HealthAddr() string`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHealthAddr|TestHTTPServerMutator' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/options.go httpserver/lifecycle.go httpserver/server.go httpserver/server_test.go httpserver/health_server_test.go
git commit -m "feat(httpserver): add server core extension points"
```

### Task 6: 文档收尾与全量验证

**Files:**
- Modify: `httpserver/README.md`
- Modify: `httpserver/doc.go`
- Verify only

**Step 1: Update docs**

同步文档：

- `DrainTimeout` 真实语义
- `State()` 与 `IsRunning()` 推荐用法
- `HealthAddr()` 用法
- `WithHTTPServerMutator(...)` 的定位

**Step 2: Run focused package verification**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 3: Run targeted race check**

Run: `GOCACHE=/tmp/go-build go test -race ./httpserver -run 'TestDrainTimeout|TestServerStateTransitions|TestHTTPServerMutator' -count=1 -v`
Expected: PASS

**Step 4: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 5: Optional full repository verification**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add httpserver docs/plans
git commit -m "refactor(httpserver): strengthen server core lifecycle"
```
