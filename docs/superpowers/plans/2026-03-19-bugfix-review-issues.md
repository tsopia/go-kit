# Bug Fix: Code Review Issues Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 code review 发现的 4 个问题：CORS fallback 安全漏洞、OnStateChange 异步调用顺序问题、CheckHealth 继承外部 ctx 导致假阴性、RateLimit 冗余 Abort 调用。

**Architecture:** 每个 Task 对应一个独立 bug，修改单一文件，TDD 先写/补测试再修复。所有 Task 之间无依赖，可独立执行。

**Tech Stack:** Go 1.21+, gin, golang.org/x/time/rate, standard library context

---

## 文件结构

| 文件 | 操作 | 责任 |
|------|------|------|
| `httpserver/middleware/cors.go` | 修改 `matchOrigin` + 更新注释 | 修复 AllowOriginFunc fallback 逻辑 |
| `httpserver/middleware/cors_test.go` | 新增测试用例 | 验证 func 拒绝后不 fallback |
| `httpserver/readiness.go` | 修改 `tryTransitionTo` | 同步调用 OnStateChange |
| `httpserver/options.go` | 修改 `OnStateChange` 注释 | 说明调用方负责 goroutine |
| `httpserver/server_test.go` | 新增测试用例 | 验证 OnStateChange 顺序；验证 CheckHealth 独立 ctx |
| `httpserver/server.go` | 修改 `CheckHealth` | 使用 context.Background() 代替继承 ctx |
| `httpserver/middleware/rate_limit.go` | 修改 `RateLimitWithConfig` | 删除冗余 c.Abort()，保留 OnRejected 前的 Abort 语义 |
| `httpserver/middleware/rate_limit_test.go` | 新增测试用例 | 验证 Retry-After 在 OnRejected 未写响应时存在 |

---

## Task 1：修复 CORS AllowOriginFunc fallback 安全漏洞

**文件：**
- 修改：`httpserver/middleware/cors.go:101-119`（`matchOrigin` 函数）
- 测试：`httpserver/middleware/cors_test.go`（新增用例到现有 table）

**问题描述：** `matchOrigin` 中，`AllowOriginFunc` 返回 `false` 后仍继续遍历 `AllowOrigins`，导致 func 无法独立阻断 origin。唯一需要修改的函数是 `matchOrigin`——`normalizeCORSConfig` 已经有 `&& config.AllowOriginFunc == nil` 条件保护，无需改动。

- [ ] **Step 1: 写失败测试**

在 `cors_test.go` 的 `testCases` 切片末尾添加两个用例（位于现有最后一个 case 的右花括号之后、切片闭合括号之前）：

```go
{
    name: "AllowOriginFunc rejects - does NOT fallback to AllowOrigins",
    config: CORSConfig{
        AllowOrigins: []string{"https://allowed.com"},
        AllowOriginFunc: func(origin string) bool {
            return false // 明确拒绝所有
        },
    },
    origin:       "https://allowed.com", // 在 AllowOrigins 里，但 func 拒绝
    method:       http.MethodGet,
    wantStatus:   http.StatusOK,
    wantNoOrigin: true, // func 拒绝后不应 fallback
},
{
    name: "AllowOriginFunc rejects non-matching origin - no fallback to wildcard",
    config: CORSConfig{
        // AllowOrigins 未设置时 normalizeCORSConfig 不填充默认值（因为 AllowOriginFunc != nil）
        AllowOriginFunc: func(origin string) bool {
            return origin == "https://explicit.com"
        },
    },
    origin:       "https://other.com",
    method:       http.MethodGet,
    wantStatus:   http.StatusOK,
    wantNoOrigin: true,
},
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test -v ./httpserver/middleware/ -run TestCORS
```

预期：新增的第一个 case `"AllowOriginFunc rejects - does NOT fallback to AllowOrigins"` FAIL（当前 `matchOrigin` 会 fallback 到 `AllowOrigins`，返回了 origin header）。第二个 case 可能 PASS 也可能 FAIL，取决于 `normalizeCORSConfig` 是否已正确保护。

- [ ] **Step 3: 修复 `matchOrigin`（cors.go:101-119）**

将整个 `matchOrigin` 函数改为：
```go
func matchOrigin(origin string, config CORSConfig) string {
	if config.AllowOriginFunc != nil {
		if config.AllowOriginFunc(origin) {
			return origin
		}
		return "" // func 是最终判决，不 fallback 到 AllowOrigins
	}

	for _, allowed := range config.AllowOrigins {
		if allowed == "*" {
			return "*"
		}
		if strings.EqualFold(allowed, origin) {
			return origin
		}
	}

	return ""
}
```

同时更新 `CORSConfig.AllowOriginFunc` 字段注释（cors.go:19-21）：
```go
// AllowOriginFunc 动态判断 Origin 是否允许。
// 设置后完全接管 origin 判断，AllowOrigins 不再生效。
// 返回 true 表示允许，返回 false 表示拒绝（不 fallback 到 AllowOrigins）。
AllowOriginFunc func(origin string) bool
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test -v ./httpserver/middleware/ -run TestCORS
```

预期：所有 case PASS

- [ ] **Step 5: 运行全量测试**

```bash
go test ./httpserver/...
```

预期：全部 PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver/middleware/cors.go httpserver/middleware/cors_test.go
git commit -m "fix(cors): AllowOriginFunc 拒绝时不再 fallback 到 AllowOrigins"
```

---

## Task 2：修复 OnStateChange 异步调用破坏顺序保证

**文件：**
- 修改：`httpserver/readiness.go:60-62`（`tryTransitionTo` 中的 hook 调用）
- 修改：`httpserver/options.go`（`OnStateChange` 字段注释）
- 测试：`httpserver/server_test.go`（新增顺序验证测试，需确认文件已有 `"fmt"` 和 `"sync"` import，如无则添加）

**问题描述：** `go s.hooks.OnStateChange(...)` 异步触发导致多次状态转换时 hook 调用顺序不保证。改为同步调用后，由调用方决定是否开 goroutine。同步调用不会持有锁（锁已在 `Unlock()` 后释放），不阻塞状态机本身，只阻塞当前调用链。

**测试设计说明：** 不使用 `runtime.Gosched()` 或 `time.Sleep`，通过"在 `tryTransitionTo` 返回后立即检查 events"来验证同步语义——若为异步则 events 长度为 0 导致 FAIL，若为同步则 events 长度为 2 且有序。

- [ ] **Step 1: 检查 server_test.go 的 import 并写失败测试**

先确认 `httpserver/server_test.go` 的 import 块是否包含 `"fmt"` 和 `"sync"`。若不包含，在写测试时同时添加这两个 import。

在 `httpserver/server_test.go` 末尾新增：

```go
func TestOnStateChangeCalledSynchronously(t *testing.T) {
    // 验证 OnStateChange 在 tryTransitionTo 返回前已被调用（同步语义）。
    // 若为异步，tryTransitionTo 返回后 events 可能为空，测试 FAIL。
    var (
        mu     sync.Mutex
        events []string
    )

    srv := NewServer(nil, WithHooks(Hooks{
        OnStateChange: func(ctx context.Context, from State, to State) {
            mu.Lock()
            defer mu.Unlock()
            events = append(events, fmt.Sprintf("%s->%s", from, to))
        },
    }))

    _ = srv.tryTransitionTo(StateStarting)
    _ = srv.tryTransitionTo(StateReady)

    // 不加任何 sleep/Gosched：同步调用时此处 events 已填充完毕
    mu.Lock()
    defer mu.Unlock()

    if len(events) != 2 {
        t.Fatalf("expected 2 events synchronously, got %d: %v", len(events), events)
    }
    if events[0] != "new->starting" {
        t.Errorf("events[0] = %q, want %q", events[0], "new->starting")
    }
    if events[1] != "starting->ready" {
        t.Errorf("events[1] = %q, want %q", events[1], "starting->ready")
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test -v -race ./httpserver/ -run TestOnStateChangeCalledSynchronously
```

预期：FAIL，`got 0` 或顺序不定（异步 goroutine 在检查时还没执行）。加 `-race` 以同时验证 data race。

- [ ] **Step 3: 修改 `tryTransitionTo`（readiness.go:60-62）**

将：
```go
if s.hooks.OnStateChange != nil {
    go s.hooks.OnStateChange(context.Background(), from, state)
}
```

改为：
```go
if s.hooks.OnStateChange != nil {
    s.hooks.OnStateChange(context.Background(), from, state)
}
```

- [ ] **Step 4: 更新 `OnStateChange` 注释（options.go）**

找到 `OnStateChange` 字段，更新注释：
```go
// OnStateChange 在每次状态转换完成后同步调用。
// 调用时状态机锁已释放，可安全读取服务器状态。
// 注意：hook 会阻塞当前调用链，请确保其快速返回；
// 如需异步处理，请在 hook 内部自行开启 goroutine。
OnStateChange func(ctx context.Context, from State, to State)
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test -v -race ./httpserver/ -run TestOnStateChangeCalledSynchronously
```

预期：PASS，无 data race

- [ ] **Step 6: 运行全量测试**

```bash
go test -race ./httpserver/...
```

预期：全部 PASS

- [ ] **Step 7: Commit**

```bash
git add httpserver/readiness.go httpserver/options.go httpserver/server_test.go
git commit -m "fix(server): OnStateChange 改为同步调用以保证顺序语义"
```

---

## Task 3：修复 CheckHealth 继承外部已取消 ctx

**文件：**
- 修改：`httpserver/server.go:448`（`CheckHealth` 中 goroutine 内的 context）
- 测试：`httpserver/server_test.go`（新增测试用例）

**问题描述：** `context.WithTimeout(ctx, hcm.checkTimeout)` 若入参 `ctx` 已取消，派生的 `checkCtx` 立即处于取消状态，所有检查 `ctx.Err()` 的 checker 立即返回 error，产生假阴性结果。健康检查生命周期应独立于请求上下文。

**测试设计说明：** checker 必须主动检查 `ctx.Err()`，否则即使传入已取消的 ctx，checker 也能正常返回 nil（掩盖 bug）。测试中的 checker 在执行时先检查 ctx 状态，若已取消则返回 error。

- [ ] **Step 1: 写失败测试**

在 `httpserver/server_test.go` 末尾新增：

```go
func TestCheckHealthWithCancelledContext(t *testing.T) {
    // 验证传入已取消的 ctx 时，健康检查不应因 ctx 取消而立即失败。
    // 健康检查使用独立 context，不继承请求 ctx 的取消状态。
    manager := NewHealthCheckManager("v1.0.0")

    manager.AddChecker(NewCustomHealthChecker("ctx-sensitive", func(ctx context.Context) error {
        // 模拟一个会检查 ctx 状态的 checker（如数据库 Ping）
        if ctx.Err() != nil {
            return fmt.Errorf("context cancelled: %w", ctx.Err())
        }
        return nil
    }))

    // 使用已取消的 ctx
    cancelledCtx, cancel := context.WithCancel(context.Background())
    cancel() // 立即取消

    status := manager.CheckHealth(cancelledCtx)

    // 使用独立 context 后，checker 接收到的 ctx 不应是已取消状态
    if status.Status != "healthy" {
        t.Errorf("status = %q, want %q — health check should use independent context, not inherit cancelled ctx",
            status.Status, "healthy")
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test -v ./httpserver/ -run TestCheckHealthWithCancelledContext
```

预期：FAIL，`status = "unhealthy"`（checker 收到已取消的 checkCtx，`ctx.Err() != nil` 为 true，返回 error）

- [ ] **Step 3: 修改 `CheckHealth`（server.go:447-448）**

将：
```go
go func(c HealthChecker) {
    checkCtx, cancel := context.WithTimeout(ctx, hcm.checkTimeout)
```

改为：
```go
go func(c HealthChecker) {
    checkCtx, cancel := context.WithTimeout(context.Background(), hcm.checkTimeout)
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test -v ./httpserver/ -run TestCheckHealthWithCancelledContext
```

预期：PASS

- [ ] **Step 5: 运行全量测试**

```bash
go test ./httpserver/...
```

预期：全部 PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver/server.go httpserver/server_test.go
git commit -m "fix(server): CheckHealth 使用独立 context 避免继承已取消的请求 ctx"
```

---

## Task 4：清理 RateLimit 冗余的 c.Abort() 调用

**文件：**
- 修改：`httpserver/middleware/rate_limit.go:44-62`
- 测试：`httpserver/middleware/rate_limit_test.go`（新增测试用例）

**问题描述：** 当前代码先调 `c.Abort()`，再在条件分支里调 `c.AbortWithStatus()`，前者冗余。`c.AbortWithStatus` 内部已调用 `c.Abort()`。

**重构边界说明：** 原代码中 `c.Abort()` 先于 `OnRejected` 执行，防止 `OnRejected` 内意外调用 `c.Next()` 使后续 handler 继续执行。重构后，`OnRejected` 执行时 `c.Abort()` 尚未调用，`OnRejected` 内可以调用 `c.Next()`（但文档没说可以这样做）。这是一个微小的语义变化，可接受——`OnRejected` 的设计意图是写拒绝响应，不应调用 `c.Next()`。

- [ ] **Step 1: 写基线测试（确认现有行为正确）**

在 `rate_limit_test.go` 末尾新增，验证 `OnRejected` 不写响应时回退到 `Retry-After + 429`：

```go
func TestRateLimitRetryAfterSetWhenOnRejectedDoesNotWrite(t *testing.T) {
    t.Parallel()

    gin.SetMode(gin.TestMode)

    engine := gin.New()
    engine.Use(RateLimitWithConfig(RateLimitConfig{
        Rate:  1,
        Burst: 1,
        OnRejected: func(c *gin.Context) {
            // 不写响应，应回退到默认 429 + Retry-After
        },
    }))
    engine.GET("/test", func(c *gin.Context) {
        c.Status(http.StatusOK)
    })

    // 消耗 token
    w1 := httptest.NewRecorder()
    engine.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/test", nil))

    // 被拒绝的请求
    w2 := httptest.NewRecorder()
    engine.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/test", nil))

    if w2.Code != http.StatusTooManyRequests {
        t.Fatalf("status = %d, want %d", w2.Code, http.StatusTooManyRequests)
    }
    if w2.Header().Get("Retry-After") == "" {
        t.Fatal("Retry-After header should be set when OnRejected does not write response")
    }
}
```

- [ ] **Step 2: 运行测试确认基线通过**

```bash
go test -v ./httpserver/middleware/ -run TestRateLimitRetryAfterSetWhenOnRejectedDoesNotWrite
```

预期：PASS（验证重构前的行为是正确的）

- [ ] **Step 3: 重构 `RateLimitWithConfig`（rate_limit.go:44-62）**

将：
```go
return func(c *gin.Context) {
    if limiter.Allow() {
        c.Next()
        return
    }

    c.Abort()

    if config.OnRejected != nil {
        config.OnRejected(c)
        if c.Writer.Written() {
            return
        }
    }

    c.Header("Retry-After", "1")
    c.AbortWithStatus(http.StatusTooManyRequests)
}
```

改为（在 `OnRejected` 前调用 `c.Abort()` 阻止后续 handler，移除末尾冗余的单独 `c.Abort()`）：
```go
return func(c *gin.Context) {
    if limiter.Allow() {
        c.Next()
        return
    }

    c.Abort() // 阻止后续 handler，不论 OnRejected 如何处理

    if config.OnRejected != nil {
        config.OnRejected(c)
        if c.Writer.Written() {
            return
        }
    }

    c.Header("Retry-After", "1")
    c.AbortWithStatus(http.StatusTooManyRequests)
}
```

**注意：** 仔细对比原代码，`c.Header("Retry-After", "1")` 必须在 `c.AbortWithStatus` 之前，否则 header 不会被设置（`AbortWithStatus` 写状态码后 header 仍可设置，但代码顺序需保持）。实际上 header 在 WriteHeader 前均可设置，`c.AbortWithStatus` 调用的是 `c.Status(code)`（不立即写），真正写出是在请求结束时，所以顺序无严格要求——但保持在前更清晰。

**实际上和原代码完全相同——经过仔细分析，原代码的 `c.Abort()` 有其语义价值（在 OnRejected 前 abort），真正冗余的是：`c.AbortWithStatus` 内部会再次调用 `c.Abort()`，但这不影响正确性。正确的简化是保持 `c.Abort()` 在 `OnRejected` 前，只是让代码意图更清晰。**

此 Task 的改动量为零（原代码逻辑已是正确的），但新增了一个缺失的测试用例来明确记录行为。

- [ ] **Step 4: 运行全量 RateLimit 测试**

```bash
go test -v ./httpserver/middleware/ -run TestRateLimit
```

预期：所有 RateLimit 测试 PASS

- [ ] **Step 5: 运行全量测试**

```bash
go test ./httpserver/...
```

预期：全部 PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver/middleware/rate_limit_test.go
git commit -m "test(rate_limit): 补充 OnRejected 不写响应时 Retry-After 行为测试"
```

---

## 完成验证

所有 Task 完成后执行最终验证：

```bash
go test -race ./httpserver/... -count=1
```

预期：全部包 PASS，无 data race 报告。
