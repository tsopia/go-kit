# HTTPServer Concurrency Limit Middleware Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `httpserver/middleware` 增加一个单进程全局并发闸门中间件，在超过上限时立即返回 `503`，保护服务进程承载能力。

**Architecture:** 在 `httpserver/middleware` 中新增 `concurrency_limit.go` 与对应测试，使用 `chan struct{}` 实现固定容量信号量。第一版只做全局并发限制，不排队、不分桶，并通过可选 `OnRejected` 允许业务覆盖默认拒绝响应。

**Tech Stack:** Go 1.24、Gin、channel-based semaphore、table-driven tests

---

### Task 1: 能力清单和文档入口

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`
- Modify: `httpserver/middleware/doc.go`
- Modify: `httpserver/middleware/README.md`
- Modify: `httpserver/README.md`

**Step 1: Update capability metadata first**

在 `.ai/capabilities.yaml` 中补充 `middleware` 的并发保护场景，在 `AGENTS.md` 与 `README` 中增加 `ConcurrencyLimit` 的简要说明。

**Step 2: Verify YAML syntax**

Run: `ruby -e 'require "yaml"; YAML.load_file(".ai/capabilities.yaml")'`
Expected: exit 0

**Step 3: Commit**

```bash
git add .ai/capabilities.yaml AGENTS.md httpserver/middleware/doc.go httpserver/middleware/README.md httpserver/README.md
git commit -m "docs(middleware): describe concurrency limit capability"
```

### Task 2: 基础并发闸门

**Files:**
- Create: `httpserver/middleware/concurrency_limit.go`
- Test: `httpserver/middleware/concurrency_limit_test.go`

**Step 1: Write the failing test**

新增 table-driven tests，断言：

- 未超过上限时请求正常通过
- 第一个请求占住槽位时，第二个请求立即返回 `503`
- 第一个请求释放后，后续请求可再次通过

测试建议使用 channel 控制第一个请求阻塞，确保行为稳定可重复。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitAllowsWithinLimit|TestConcurrencyLimitRejectsWhenFull|TestConcurrencyLimitReleasesSlotAfterCompletion' -v`
Expected: FAIL，因为当前没有 `ConcurrencyLimit`

**Step 3: Write minimal implementation**

在 `httpserver/middleware/concurrency_limit.go` 中新增：

- `ConcurrencyLimitConfig`
- `ConcurrencyLimit(limit int)`
- `ConcurrencyLimitWithConfig(config ConcurrencyLimitConfig)`
- 基于 `chan struct{}` 的固定容量信号量

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitAllowsWithinLimit|TestConcurrencyLimitRejectsWhenFull|TestConcurrencyLimitReleasesSlotAfterCompletion' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/concurrency_limit.go httpserver/middleware/concurrency_limit_test.go
git commit -m "feat(middleware): add concurrency limit middleware"
```

### Task 3: 自定义拒绝响应与配置校验

**Files:**
- Modify: `httpserver/middleware/concurrency_limit.go`
- Test: `httpserver/middleware/concurrency_limit_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- 默认超限时只返回 `503`
- `OnRejected` 可覆盖默认响应
- `Limit <= 0` 时直接 panic

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitDefaultRejection|TestConcurrencyLimitCustomRejection|TestConcurrencyLimitPanicsOnInvalidLimit' -v`
Expected: FAIL，因为当前默认拒绝与配置校验尚未完整实现

**Step 3: Write minimal implementation**

在 `concurrency_limit.go` 中补充：

- 默认 `AbortWithStatus(http.StatusServiceUnavailable)`
- `OnRejected` 自定义覆盖逻辑
- `Limit <= 0` 的 fail-fast 校验

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitDefaultRejection|TestConcurrencyLimitCustomRejection|TestConcurrencyLimitPanicsOnInvalidLimit' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/concurrency_limit.go httpserver/middleware/concurrency_limit_test.go
git commit -m "feat(middleware): support configurable concurrency rejection"
```

### Task 4: 槽位释放边界

**Files:**
- Modify: `httpserver/middleware/concurrency_limit.go`
- Test: `httpserver/middleware/concurrency_limit_test.go`

**Step 1: Write the failing test**

新增 tests，断言：

- handler panic 后槽位会释放
- 与 `Timeout(...)` 叠加时，超时完成后槽位会释放

测试需要明确验证：前一个请求出错结束后，新的请求可以重新进入。

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitReleasesSlotAfterPanic|TestConcurrencyLimitReleasesSlotAfterTimeout' -v`
Expected: FAIL，因为当前释放边界尚未验证完整

**Step 3: Write minimal implementation**

在 `concurrency_limit.go` 中确保槽位释放使用 `defer` 包裹整个后续链路，覆盖：

- 正常返回
- `Abort`
- `panic`
- `Timeout`

**Step 4: Run test to verify it passes**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestConcurrencyLimitReleasesSlotAfterPanic|TestConcurrencyLimitReleasesSlotAfterTimeout' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/concurrency_limit.go httpserver/middleware/concurrency_limit_test.go
git commit -m "fix(middleware): release concurrency slots on failure"
```

### Task 5: 全量验证

**Files:**
- Verify only

**Step 1: Run package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -v`
Expected: PASS

**Step 2: Run httpserver package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`
Expected: PASS

**Step 3: Run lint**

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: `0 issues.`

**Step 4: Verify capability listing**

Run:

```bash
cd cmd/gokit && GOCACHE=/tmp/go-build go build -o /tmp/gokit .
cd /Users/kj/projects/go-kit && /tmp/gokit list
```

Expected: `middleware` 能力描述体现 `ConcurrencyLimit`

**Step 5: Run full repository tests**

Run: `GOCACHE=/tmp/go-build go test ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add docs/plans .ai/capabilities.yaml AGENTS.md httpserver
git commit -m "feat(middleware): add concurrency limit middleware"
```
