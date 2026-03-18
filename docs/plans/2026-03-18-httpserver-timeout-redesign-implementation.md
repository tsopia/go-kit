# HTTPServer Timeout Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `httpserver/middleware.Timeout` 从当前并发抢写响应的实现，重构为同步 `context deadline` 注入的协作式超时 middleware，并修正它与 `ConcurrencyLimit` 的组合测试语义。

**Architecture:** 保留 `Timeout(timeout time.Duration) gin.HandlerFunc` 公开 API，但重写内部语义为同步 `c.Next()` + `context.WithTimeout`。同时修正 `timeout_test.go` 与 `concurrency_limit_test.go`，只验证 ctx-aware handler 的 timeout 退出和槽位释放，不再给“504 一返回就释放槽位”的异步语义背书。

**Tech Stack:** Go 1.24、Gin、`context.WithTimeout`、table-driven tests、`go test -race`

---

### Task 1: 文档澄清 Timeout 与 ConcurrencyLimit 语义

**Files:**
- Modify: `httpserver/middleware/README.md`
- Modify: `httpserver/README.md`

**Step 1: Update docs first**

在 README 中明确：

- `Timeout` 是协作式执行预算
- 不是 goroutine 强制终止器
- `ConcurrencyLimit` 统计真实执行中的 handler
- 与 `Timeout` 组合时，槽位释放以“执行结束”为准，而不是以“504 已返回”为准

**Step 2: Commit**

```bash
git add httpserver/middleware/README.md httpserver/README.md
git commit -m "docs(middleware): clarify timeout and concurrency contracts"
```

### Task 2: 用失败测试锁定 Timeout 的新 contract

**Files:**
- Modify: `httpserver/middleware/timeout_test.go`

**Step 1: Write the failing test**

新增或改写测试，锁定以下语义：

- ctx-aware handler 在 deadline 后主动退出
- middleware 在 handler 返回后写出 `504`
- 测试不再依赖并发执行 `c.Next()` 的旧行为

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestTimeout' -v`
Expected: FAIL，因为当前 `timeout.go` 仍然是并发实现

**Step 3: Commit**

```bash
git add httpserver/middleware/timeout_test.go
git commit -m "test(middleware): redefine timeout as cooperative deadline"
```

### Task 3: 重写 Timeout 为同步协作式实现

**Files:**
- Modify: `httpserver/middleware/timeout.go`

**Step 1: Write minimal implementation**

重构 `timeout.go`：

- 保留 `Timeout(timeout time.Duration) gin.HandlerFunc`
- 用 `context.WithTimeout` 派生新 ctx
- 同步执行 `c.Next()`
- `c.Next()` 返回后，如 `ctx.Err() == context.DeadlineExceeded` 且响应未写出，则 `AbortWithStatus(http.StatusGatewayTimeout)`
- 不再起 goroutine 并发操作 `gin.Context`

**Step 2: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestTimeout' -v`
Expected: PASS

**Step 3: Commit**

```bash
git add httpserver/middleware/timeout.go httpserver/middleware/timeout_test.go
git commit -m "refactor(middleware): make timeout cooperative and synchronous"
```

### Task 4: 修正 ConcurrencyLimit 的 timeout 测试方向

**Files:**
- Modify: `httpserver/middleware/concurrency_limit_test.go`

**Step 1: Write the failing test**

调整 timeout 相关测试，要求：

- 使用推荐顺序：`Recovery -> ConcurrencyLimit -> Timeout`
- 使用 ctx-aware slow handler，在 `ctx.Done()` 后退出
- 只在 handler 真正退出后，断言新的请求可以进入

同时移除或改写当前“504 返回即释放槽位”的争议测试方向。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitReleasesSlotAfterTimeout|TestConcurrencyLimitReleasesSlotAfterPanic' -v`
Expected: FAIL，因为当前 timeout 组合语义仍基于旧方向

**Step 3: Write minimal implementation / test fix**

只改测试和必要的最小实现，不扩展 `ConcurrencyLimit` API。

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitReleasesSlotAfterTimeout|TestConcurrencyLimitReleasesSlotAfterPanic' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/concurrency_limit_test.go httpserver/middleware/concurrency_limit.go
git commit -m "test(middleware): align concurrency timeout behavior with cooperative contract"
```

### Task 5: 竞争检测与全量验证

**Files:**
- Verify only

**Step 1: Run middleware package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -v`
Expected: PASS

**Step 2: Run targeted race check**

Run: `GOCACHE=/tmp/go-build go test -race ./httpserver/middleware -run 'TestTimeout|TestConcurrencyLimitReleasesSlotAfter(Panic|Timeout)$' -count=1 -v`
Expected: PASS

**Step 3: Run httpserver package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 4: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 5: Optional full repository tests**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add httpserver docs/plans
git commit -m "refactor(middleware): redesign timeout contract"
```
