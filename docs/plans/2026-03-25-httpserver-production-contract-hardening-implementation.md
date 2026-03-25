# HTTPServer Production Contract Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 `httpserver` 收敛成“运行时零配置可用、策略边界默认安全、文档和测试可支撑对外发布判断”的工具包 contract。

**Architecture:** 保留 `timeout`、`shutdown`、`health` 这类运行时基线默认值；把 `CORS`、`WebSocket Origin`、代理信任边界等策略型能力改成“默认关闭/默认拒绝/必须显式配置”。日志层采用“一套配置入口，多种事件类型”的方案，复用现有 `AccessLog` 的 logger、trace/request_id、header allowlist 和脱敏基础设施，但不强行把 SSE/WS 塞进同一个 request/response 语义。

**Tech Stack:** Go 1.24、Gin、Gorilla WebSocket、`net/http`、table-driven tests、`go test`、`golangci-lint`

---

## 前置检查

- [x] 已完整阅读相关代码文件：`httpserver/ws.go`、`httpserver/ws_session.go`、`httpserver/server.go`、`httpserver/sse.go`、`httpserver/config.go`、`httpserver/middleware/cors.go`、`httpserver/middleware/access_log.go`、`httpserver/middleware/recovery.go`、`httpserver/middleware/real_ip.go`、`httpserver/preset/production.go`、`httpserver/README.md`、`httpserver/preset/README.md`
- [x] 已完整阅读相关测试文件：`httpserver/ws_test.go`、`httpserver/sse_test.go`、`httpserver/lifecycle_test.go`、`httpserver/health_server_test.go`、`httpserver/middleware/cors_test.go`、`httpserver/middleware/access_log_test.go`、`httpserver/middleware/recovery_test.go`、`httpserver/middleware/real_ip_test.go`
- [x] 已列出代码现状（每个判断附行号）

## 代码现状

- [x] **确定**：`WSUpgrader.CheckOrigin` 当前无条件 `return true`，等于默认允许所有浏览器 `Origin`。证据：`httpserver/ws.go:40-47`
- [x] **确定**：`CORSConfig{}` 当前会在 `normalizeCORSConfig(...)` 中补成 `AllowOrigins: []string{"*"}`。证据：`httpserver/middleware/cors.go:89-92`
- [x] **确定**：`AccessLog` 已经能统一输出 `request_id`、`trace_id`、`client_ip`、`bytes_in/out` 等 HTTP 摘要字段，但仅覆盖 request/response 生命周期。证据：`httpserver/middleware/access_log.go:96-157`
- [x] **确定**：`SSE` 与 `WS` 当前都走 streaming helper 路由，因此会经过根 middleware；但连接级日志仍使用 `slog` 直出，未复用 `AccessLog` 体系。证据：`httpserver/server.go:175-216`、`httpserver/server.go:227-398`、`httpserver/sse.go:165-171`
- [x] **确定**：`Recovery` 现在只负责 panic containment 和结构化日志，响应侧固定为裸 `500`。证据：`httpserver/middleware/recovery.go:21-57`
- [x] **确定**：`RealIP` 默认不信任任何代理，这个默认是保守且正确的。证据：`httpserver/middleware/real_ip.go:23-45`
- [x] **确定**：生命周期、超时、健康检查主链路已经收敛，重点在于补强真实集成验证，而不是再次改动主架构。证据：`httpserver/server.go:467-530`、`httpserver/health_server_test.go:1-154`、`httpserver/lifecycle_test.go:376-424`

## 设计决策

### 决策 1：区分“运行时默认值”和“策略默认值”

- 运行时默认值保留：`ReadTimeout`、`WriteTimeout`、`IdleTimeout`、`ShutdownTimeout`、`DrainTimeout`、`MaxHeaderBytes`、健康检查路径。
- 策略默认值收紧：`CORS`、`WebSocket Origin`、代理信任边界、访问日志捕获细节。

原因：

- 运行时默认值解决的是“服务裸跑是否稳定”的问题。
- 策略默认值决定的是“信任谁、暴露什么、记录什么”的问题，这类能力必须显式化。

### 决策 2：采纳“CORS 未启用时不主动拒绝”的行为

默认语义：

- 无 `Origin` 头：正常处理请求。
- 有 `Origin` 头但未配置 CORS：不返回任何 `Access-Control-*` 头，让浏览器自然阻止。
- 只有显式配置 `AllowOrigins` 或 `AllowOriginFunc` 时，才真正启用 CORS 行为。

原因：

- 这更符合 HTTP 语义，也更接近“middleware 未启用 = 不介入协议协商”。
- 这样不会把 CORS middleware 变成一个隐藏的访问控制器。

### 决策 3：采纳“WS 无 Origin 允许，有 Origin 默认拒绝”的行为

默认策略：

```go
func defaultWSOriginCheck(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return false
}
```

原因：

- 浏览器握手天然带 `Origin`，默认拒绝可以把浏览器场景收进显式授权。
- 非浏览器客户端可能不带 `Origin`，默认允许可以保留 CLI、服务间调用、测试客户端的零配置体验。

### 决策 4：日志采用“一套配置入口，多事件模型”

- `access_log`：HTTP 请求摘要，包括 SSE/WS 握手请求。
- `payload_log`：仅用于 HTTP request/response 载荷，不覆盖 WS frame。
- `stream_connect` / `stream_disconnect`：统一流式连接日志。
- `ws_upgrade_failed`：统一握手失败日志。

原因：

- SSE 和 WS 需要共享 trace/request_id/header/redaction 的配置基础。
- 但 WS frame 不是 request/response，不能直接复用现有 payload 模型。

## 方案比较

### 方案 A：只翻转默认值，不重整 API

优点：

- 改动最小
- 可以快速把默认行为从宽松改成安全

缺点：

- `WSUpgrader` 仍是全局可变状态，安全策略仍然分散
- 日志和文档 contract 仍然不完整

### 方案 B：强收口默认值，并补齐显式配置入口（推荐）

优点：

- 一次定干净未来 contract
- 没有存量用户，breaking change 成本最低
- 可以同步清理 `WS`、`CORS`、日志、文档、测试几条线

缺点：

- 需要修改公开 API 和测试
- 文档需要整段重写

### 方案 C：只修文档，承认“用户自己负责安全边界”

优点：

- 无实现成本

缺点：

- 不能支撑“可直接给用户用”的工具包定位
- 会把不安全默认值沉淀成历史债

**推荐：采用方案 B。**

## 可能否定本设计的技术约束

- [x] `CORS` 的 no-op 默认必须保证不会意外吞掉 `OPTIONS` 行为，因此实现上只能“不写头并继续走路由”，不能在未配置时统一返回 403 或 204。
- [x] `Gorilla WebSocket` 的 `Upgrade(...)` 在 `CheckOrigin` 返回 false 时会直接以握手失败结束，这个行为是默认拒绝浏览器 `Origin` 的基础。
- [x] SSE/WS 连接日志要复用 `AccessLog` 的 logger 和脱敏能力，但不能直接依赖 `gin.ResponseWriter` 的 request/response body capture，因为 WS frame 绕过了这层 writer。

## 证伪证据

- [x] 如果 `CORSConfig{}` 不是全开放，`normalizeCORSConfig(...)` 不应填充 `*`；源码已证伪。证据：`httpserver/middleware/cors.go:89-92`
- [x] 如果 `WS` 不是默认放开浏览器来源，`CheckOrigin` 不应直接 `return true`；源码已证伪。证据：`httpserver/ws.go:44-45`
- [x] 如果 SSE/WS 已经复用 `AccessLog` 的日志契约，就不应还有 `slog.Info("sse client disconnected")` / `slog.Info("websocket client disconnected")` 这样的直出；源码已证伪。证据：`httpserver/sse.go:165-171`、`httpserver/server.go:392-397`

## 范围与非目标

本轮范围：

- 收紧 `CORS` 和 `WS Origin` 默认策略
- 建立 SSE/WS 的统一日志契约
- 给 `Recovery` 增加可配置响应出口
- 补强生命周期、健康检查、代理、默认策略的测试和文档

本轮非目标：

- 不重写 `Server` 生命周期主架构
- 不把 `WS` frame payload 纳入默认日志
- 不在 `preset` 中自动挂载 `AccessLog`、`RealIP`、认证、限流、CORS

## Phase 划分

- `P0`：默认安全边界和公开 contract
- `P1`：流式日志统一与错误恢复契约
- `P2`：测试与文档封口

### Task 1: 锁定 `CORS` 零值 no-op contract

**目标：** 用失败测试明确“未配置 CORS 时，只要不命中显式 origin 策略，就不写任何 `Access-Control-*` 头，也不主动拒绝请求”。

**Files:**
- Modify: `httpserver/middleware/cors_test.go`

**Step 1: 写失败测试**

```go
func TestCORS_DefaultConfigIsNoOp(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(CORS(CORSConfig{}))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCORS_DefaultConfigIsNoOp|TestCORS' -v`
Expected: FAIL，现状会拿到 `Access-Control-Allow-Origin: *`

**Step 3: 写最简实现**

- 删除 `normalizeCORSConfig(...)` 中对 `AllowOrigins = []string{"*"}` 的默认填充
- 保留 `AllowMethods`、`AllowHeaders`、`ExposeHeaders` 的默认补全
- 保证未配置 origin 策略时直接 `c.Next()`

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCORS_DefaultConfigIsNoOp|TestCORS' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/cors.go httpserver/middleware/cors_test.go
git commit -m "fix(httpserver): make zero-value cors config a no-op"
```

### Task 2: 把 `CORS` 全开放从隐式默认改成显式声明

**目标：** 保留“显式全开放”的能力，但只能通过 `AllowOrigins: []string{"*"}` 打开。

**Files:**
- Modify: `httpserver/middleware/cors_test.go`
- Modify: `httpserver/README.md`

**Step 1: 写失败测试**

```go
func TestCORS_WildcardMustBeExplicit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(CORS(CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow origin = %q, want *", got)
	}
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCORS_WildcardMustBeExplicit|TestCORS' -v`
Expected: FAIL 或现有用例与新 contract 冲突

**Step 3: 写最简实现**

- 调整 table-driven tests，删除“default allows all origins”
- README 补充“零值 = 不启用，`*` 必须显式声明”

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestCORS_WildcardMustBeExplicit|TestCORS' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/cors_test.go httpserver/README.md
git commit -m "docs(httpserver): require explicit wildcard cors policy"
```

### Task 3: 锁定 `WS` 默认 origin policy

**目标：** 用失败测试明确“无 `Origin` 允许，有 `Origin` 默认拒绝”。

**Files:**
- Modify: `httpserver/ws_test.go`

**Step 1: 写失败测试**

```go
func TestDefaultWSOriginCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "no origin allowed", origin: "", want: true},
		{name: "browser origin denied by default", origin: "https://app.example.com", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if got := defaultWSOriginCheck(req); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestDefaultWSOriginCheck|Test(WS|WSSession)' -v`
Expected: FAIL，当前默认是放开全部来源

**Step 3: 写最简实现**

- 在 `httpserver/ws.go` 新增 `defaultWSOriginCheck(...)`
- 把默认 `CheckOrigin` 改成该函数

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestDefaultWSOriginCheck|Test(WS|WSSession)' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/ws.go httpserver/ws_test.go
git commit -m "fix(httpserver): deny browser websocket origins by default"
```

### Task 4: 去掉 `WS` 安全策略对全局 mutable upgrader 的依赖

**目标：** 把 origin 授权从包级全局变量改成显式配置入口，避免不同 server 间共享可变状态。

**Files:**
- Modify: `httpserver/ws.go`
- Modify: `httpserver/server.go`
- Modify: `httpserver/ws_test.go`
- Modify: `httpserver/README.md`

**Step 1: 写失败测试**

```go
func TestWSRouteOption_AllowsExplicitOrigin(t *testing.T) {
	t.Parallel()

	cfg := defaultWSRouteConfig()
	WithWSAllowedOrigins("https://app.example.com").applyRoute(&cfg)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	if !cfg.CheckOrigin(req) {
		t.Fatal("expected configured origin to be allowed")
	}
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestWSRouteOption_AllowsExplicitOrigin|Test(WS|WSSession)' -v`
Expected: FAIL，当前没有显式 route-level origin 配置入口

**Step 3: 写最简实现**

```go
type WSRouteConfig struct {
	WSConfig
	ReadIdleTimeout time.Duration
	WriteTimeout    time.Duration
	CheckOrigin     func(*http.Request) bool
}

func WithWSAllowedOrigins(origins ...string) WSRouteOption
func WithWSOriginChecker(fn func(*http.Request) bool) WSRouteOption
```

- `defaultWSRouteConfig()` 默认使用 `defaultWSOriginCheck`
- `Server.WS(...)` 内部构造局部 upgrader，并把 `CheckOrigin` 绑定到 route config
- 移除或弱化公开包级 `WSUpgrader` 对安全策略的责任，只保留必要的握手参数配置能力

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestWSRouteOption_AllowsExplicitOrigin|Test(WS|WSSession)' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/ws.go httpserver/server.go httpserver/ws_test.go httpserver/README.md
git commit -m "refactor(httpserver): make websocket origin policy explicit"
```

### Task 5: 锁定流式日志 contract

**目标：** 先用测试锁定 `SSE/WS` 需要输出统一结构化字段，而不是继续走 `slog` 直出。

**Files:**
- Modify: `httpserver/sse_test.go`
- Modify: `httpserver/ws_test.go`
- Modify: `httpserver/middleware/access_log_test.go`

**Step 1: 写失败测试**

```go
func TestStreamLogsIncludeTraceAndRequestMetadata(t *testing.T) {
	t.Parallel()

	// logger 捕获 stream_connect / stream_disconnect / ws_upgrade_failed
	// 断言 path、request_id、trace_id、client_ip 存在
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -run 'TestStreamLogsIncludeTraceAndRequestMetadata|TestAccessLogSummary' -v`
Expected: FAIL，当前 SSE/WS 断开日志不走统一 logger

**Step 3: 写最简实现**

- 新增内部 stream logger adapter，复用 `LoggerFunc`
- `SSE` 连接开始/结束、`WS` upgrade 失败/连接结束统一写结构化事件
- 默认字段至少包含：`event`、`path`、`route`、`request_id`、`trace_id`、`client_ip`、`remote_addr`、`duration_ms`

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -run 'TestStreamLogsIncludeTraceAndRequestMetadata|TestAccessLogSummary' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/sse.go httpserver/sse_test.go httpserver/ws_test.go httpserver/middleware/access_log_test.go
git commit -m "feat(httpserver): unify stream logging contract"
```

### Task 6: 为流式日志建立显式配置入口

**目标：** 让使用方可以用一套配置控制 HTTP/SSE/WS 日志，而不是分别改 `slog` 和 middleware。

**Files:**
- Modify: `httpserver/middleware/access_log.go`
- Modify: `httpserver/server.go`
- Modify: `httpserver/README.md`

**Step 1: 写失败测试**

```go
func TestAccessLogConfigCanDriveStreamEvents(t *testing.T) {
	t.Parallel()

	// AccessLogConfig.Logger 被同时用于 access_log 和 stream_* 事件
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -run 'TestAccessLogConfigCanDriveStreamEvents|TestStreamLogsIncludeTraceAndRequestMetadata' -v`
Expected: FAIL，当前 stream 事件没有统一配置入口

**Step 3: 写最简实现**

- 新增一个轻量日志配置挂载点，优先复用 `LoggerFunc`
- 不把 WS frame 纳入默认 payload capture
- 明确 `payload_log` 仍只服务 HTTP request/response 载荷

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver ./httpserver/middleware -run 'TestAccessLogConfigCanDriveStreamEvents|TestStreamLogsIncludeTraceAndRequestMetadata' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/access_log.go httpserver/server.go httpserver/README.md
git commit -m "refactor(httpserver): share logging config across http and streams"
```

### Task 7: 给 `Recovery` 增加可配置 responder

**目标：** 保留 panic containment，但允许使用方显式定义 panic 后响应格式。

**Files:**
- Modify: `httpserver/middleware/recovery.go`
- Modify: `httpserver/middleware/recovery_test.go`
- Modify: `httpserver/README.md`

**Step 1: 写失败测试**

```go
func TestRecoveryWithConfigResponder(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	engine.Use(RecoveryWithConfig(RecoveryConfig{
		Responder: func(c *gin.Context, recovered any) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		},
	}))
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "internal_error") {
		t.Fatalf("body = %s", w.Body.String())
	}
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestRecoveryWithConfigResponder|TestRecovery' -v`
Expected: FAIL，当前 `RecoveryConfig` 没有 responder

**Step 3: 写最简实现**

```go
type RecoveryConfig struct {
	Logger    LoggerFunc
	OnPanic   func(c *gin.Context, recovered any, stack []byte)
	Responder func(c *gin.Context, recovered any)
}
```

- 默认 responder 仍为 `c.AbortWithStatus(http.StatusInternalServerError)`
- 自定义 responder 时仍先打日志，再调用 `OnPanic`，最后交给 responder 生成响应

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/middleware -run 'TestRecoveryWithConfigResponder|TestRecovery' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/middleware/recovery.go httpserver/middleware/recovery_test.go httpserver/README.md
git commit -m "feat(httpserver): add configurable recovery responder"
```

### Task 8: 补强生命周期、健康检查和代理边界的可证明性

**目标：** 不重构主逻辑，只把“已经成立的行为”升级成更像生产部署的测试和文档。

**Files:**
- Modify: `httpserver/lifecycle_test.go`
- Modify: `httpserver/health_server_test.go`
- Modify: `httpserver/middleware/real_ip_test.go`
- Modify: `httpserver/README.md`
- Modify: `httpserver/preset/README.md`

**Step 1: 写失败测试**

```go
func TestRunWithContextMarksServerUnreadyBeforeShutdown(t *testing.T) {
	t.Parallel()

	// 启动启用 health check 的 server
	// 取消 ctx 后先观察 readyz 变 503，再等待 shutdown 完成
}
```

**Step 2: 运行测试 → 确认失败**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestRunWithContextMarksServerUnreadyBeforeShutdown|TestRunWithContextUsesGracefulShutdownPipeline' -v`
Expected: FAIL 或缺少完整联动覆盖

**Step 3: 写最简实现**

- 仅在确实缺行为时补代码，否则只补测试
- README / preset README 增加生产 checklist：
  - 部署在代理后必须配置 trusted CIDRs
  - `CORS` / `WS Origin` 必须显式配置
  - `preset` 是 transport baseline，不是完整生产框架

**Step 4: 运行测试 → 确认通过**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestRunWithContextMarksServerUnreadyBeforeShutdown|TestRunWithContextUsesGracefulShutdownPipeline' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/lifecycle_test.go httpserver/health_server_test.go httpserver/middleware/real_ip_test.go httpserver/README.md httpserver/preset/README.md
git commit -m "test(httpserver): strengthen production lifecycle and proxy contracts"
```

### Task 9: 文档封口和发布门禁

**目标：** 用 README、preset README、package 文档和验证命令，把这轮 breaking change 固化成外部 contract。

**Files:**
- Modify: `httpserver/README.md`
- Modify: `httpserver/preset/README.md`
- Modify: `httpserver/doc.go`

**Step 1: 写失败测试**

本 Task 不新增 Go 测试，改为文档和门禁检查清单。

**Step 2: 运行验证命令**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/...`
Expected: PASS

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: PASS

**Step 3: 写最简实现**

- README 明确写出：
  - `CORSConfig{}` = 不启用 CORS
  - `AllowOrigins: []string{"*"}` = 显式全开放
  - `WS` 默认只允许无 `Origin` 的客户端
  - `WS` 浏览器场景必须显式配置 origin policy
  - 流式日志和 HTTP 访问日志的关系
- `preset` README 明确：
  - `preset.NewProductionServer(...)` 是 transport baseline
  - 不自动提供 CORS、RealIP、AccessLog、认证、限流

**Step 4: 运行验证命令**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/...`
Expected: PASS

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/README.md httpserver/preset/README.md httpserver/doc.go
git commit -m "docs(httpserver): document hardened production contract"
```

## 汇总验证

所有 Task 完成后，执行：

Run: `GOCACHE=/tmp/go-build go test ./httpserver/...`
Expected: PASS

Run: `GOCACHE=/tmp/go-build go test -race ./httpserver ./httpserver/middleware -count=1`
Expected: PASS

Run: `GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache GOCACHE=/tmp/go-build golangci-lint run ./httpserver/...`
Expected: PASS

## 风险提示

- 这是有意的 breaking change，尤其影响 `CORSConfig{}` 和 `WS` 浏览器接入默认行为。
- 如果实现中继续保留包级 `WSUpgrader`，必须明确其职责已经不再承载默认安全策略；否则 API 语义会再次混乱。
- 流式日志统一时要严格控制默认字段，避免在未显式开启的情况下记录过量 header 或 payload。
