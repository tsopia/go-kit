# httpserver SSE/WS Stream Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 SSE/WS 流式连接与普通 HTTP 请求共享同一套 Gin 中间件链，并修正观测型中间件（AccessLog/prometheus/otel）对流式连接的语义错配。

**Architecture:** 引入一个贯穿性的"流式标记 + 观测层感知"机制：流式 handler 在 `c.Next()` 链内用 `c.Set(StreamingKey, transport)` 打标，观测型中间件在 `c.Next()` 后读标记决定如何处理；活跃连接 gauge 通过 `StreamObserver` 接口 + context 传播实现，保持 `子包 → 核心包` 的单向依赖。

**Tech Stack:** Go, Gin (`github.com/gin-gonic/gin`), gorilla/websocket, 标准 `testing` + `net/http/httptest`。

## Global Constraints

- 模块路径：`github.com/tsopia/go-kit`
- 流式标记 key 统一通过 `utils` 包管理（与 `TraceIDKey` 同模式），值为 transport 字符串 `"sse"` / `"ws"`，空串表示非流式
- 依赖方向铁律：`observability/*` 子包可 import `httpserver/middleware` 与 `utils`；`httpserver` 核心包与 `middleware` 子包**禁止** import 任何 `observability/*` 子包
- `StreamObserver` 埋点必须 nil-safe（未挂 prometheus 时 observer 不存在）
- `OnDisconnect` 必须经 `defer` 调用，保证 panic 时 gauge 不泄漏
- 每个 Task 实现改动 ≤ 50 行；测试 : 实现 ≥ 1.5:1
- commit message 用 conventional commit 格式，结尾加：
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`
- 行为变更：流式连接不再产生 `access_log` / 请求延迟指标（已与用户确认现网未消费）

---

### Task 1: 流式基础设施（StreamingKey + StreamObserver + MarkStreaming）

**Files:**
- Create: `utils/stream.go`
- Create: `utils/stream_test.go`
- Modify: `httpserver/middleware/stream_log.go`（在文件末尾追加）
- Test: `httpserver/middleware/stream_log_test.go`（若不存在则创建）

**Interfaces:**
- Produces:
  - `utils.StreamingKey` (string 常量 = `"stream"`)
  - `middleware.StreamObserver` interface: `OnConnect(transport string)`, `OnDisconnect(transport string)`
  - `middleware.WithStreamObserver(ctx context.Context, obs StreamObserver) context.Context`
  - `middleware.StreamObserverFromContext(ctx context.Context) (StreamObserver, bool)`
  - `middleware.MarkStreaming(transport string) gin.HandlerFunc`

- [ ] **Step 1: 写失败测试 — utils.StreamingKey**

在 `utils/stream_test.go`：

```go
package utils

import "testing"

func TestStreamingKey(t *testing.T) {
	if StreamingKey != "stream" {
		t.Errorf("StreamingKey = %q, want %q", StreamingKey, "stream")
	}
}
```

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./utils/ -run TestStreamingKey -v`
Expected: FAIL（`undefined: StreamingKey`，编译失败）

- [ ] **Step 3: 写实现 — utils/stream.go**

```go
package utils

// StreamingKey 是流式连接标记在 gin.Context 中的 key。
// 值为 transport 字符串（"sse" / "ws"），空串表示非流式请求。
// 观测型中间件在 c.Next() 之后读取此 key 以正确处理流式连接。
const StreamingKey = "stream"
```

- [ ] **Step 4: 运行测试 → 确认通过**

Run: `go test ./utils/ -run TestStreamingKey -v`
Expected: PASS

- [ ] **Step 5: 写失败测试 — StreamObserver context round-trip + MarkStreaming**

在 `httpserver/middleware/stream_log_test.go`（若文件已存在则追加这两个测试函数）：

```go
package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

type recordingObserver struct {
	connects    []string
	disconnects []string
}

func (o *recordingObserver) OnConnect(transport string)    { o.connects = append(o.connects, transport) }
func (o *recordingObserver) OnDisconnect(transport string) { o.disconnects = append(o.disconnects, transport) }

func TestStreamObserverContextRoundTrip(t *testing.T) {
	obs := &recordingObserver{}
	ctx := WithStreamObserver(context.Background(), obs)

	got, ok := StreamObserverFromContext(ctx)
	if !ok {
		t.Fatal("StreamObserverFromContext returned ok=false")
	}
	got.OnConnect("sse")
	if len(obs.connects) != 1 || obs.connects[0] != "sse" {
		t.Errorf("connects = %v, want [sse]", obs.connects)
	}
}

func TestStreamObserverFromContextMissing(t *testing.T) {
	if _, ok := StreamObserverFromContext(context.Background()); ok {
		t.Error("expected ok=false for empty context")
	}
}

func TestMarkStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	MarkStreaming("ws")(c)

	if got := c.GetString(utils.StreamingKey); got != "ws" {
		t.Errorf("StreamingKey = %q, want %q", got, "ws")
	}
}
```

- [ ] **Step 6: 运行测试 → 确认失败**

Run: `go test ./httpserver/middleware/ -run 'TestStreamObserver|TestMarkStreaming' -v`
Expected: FAIL（`undefined: WithStreamObserver` 等，编译失败）

- [ ] **Step 7: 写实现 — 追加到 stream_log.go**

在 `httpserver/middleware/stream_log.go` 末尾追加（文件已 import `context`, `net/http`；需新增 import `github.com/gin-gonic/gin` 和 `github.com/tsopia/go-kit/utils`）：

```go
// StreamObserver 观测流式连接的建立与断开。
// 实现方（如 observability/prometheus）通过 WithStreamObserver 注入 context，
// SSE/WS handler 在连接建立/断开时回调，从而在不反向依赖 observability 子包的
// 前提下维护活跃连接指标。
type StreamObserver interface {
	OnConnect(transport string)
	OnDisconnect(transport string)
}

type streamObserverKey struct{}

// WithStreamObserver 将 StreamObserver 写入 context。
func WithStreamObserver(ctx context.Context, obs StreamObserver) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, streamObserverKey{}, obs)
}

// StreamObserverFromContext 从 context 读取 StreamObserver。
func StreamObserverFromContext(ctx context.Context) (StreamObserver, bool) {
	if ctx == nil {
		return nil, false
	}
	obs, ok := ctx.Value(streamObserverKey{}).(StreamObserver)
	return obs, ok
}

// MarkStreaming 返回一个把流式标记写入 gin.Context 的中间件，
// 供通过 srv.StreamingGroup() 自定义注册的流式路由使用。
// SSE/SSEPost/WS 便利方法已自动打标，无需再用此中间件。
func MarkStreaming(transport string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(utils.StreamingKey, transport)
		c.Next()
	}
}
```

同步更新文件顶部 import 块为：

```go
import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)
```

- [ ] **Step 8: 运行测试 → 确认通过**

Run: `go test ./httpserver/middleware/ -run 'TestStreamObserver|TestMarkStreaming' -v`
Expected: PASS

- [ ] **Step 9: 全量编译 + 包测试**

Run: `go build ./... && go test ./utils/ ./httpserver/middleware/`
Expected: 全部 PASS

- [ ] **Step 10: Commit**

```bash
git add utils/stream.go utils/stream_test.go httpserver/middleware/stream_log.go httpserver/middleware/stream_log_test.go
git commit -m "feat(httpserver): add streaming marker and StreamObserver primitives

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: SSE/WS handler 打标 + observer 埋点

**Files:**
- Modify: `httpserver/server.go`（SSE ginHandler `:208-244`，WS ginHandler `:262-436`）
- Test: `httpserver/stream_observer_test.go`（创建）

**Interfaces:**
- Consumes: `utils.StreamingKey`, `middleware.StreamObserverFromContext`（Task 1）
- Produces: SSE handler 设置 `c.Set(StreamingKey, "sse")` 并在连接期间回调 observer；WS handler 设置 `c.Set(StreamingKey, "ws")` 并回调 observer

- [ ] **Step 1: 写失败测试**

在 `httpserver/stream_observer_test.go`：

```go
package httpserver

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
)

type countingObserver struct {
	mu          sync.Mutex
	connects    map[string]int
	disconnects map[string]int
}

func newCountingObserver() *countingObserver {
	return &countingObserver{connects: map[string]int{}, disconnects: map[string]int{}}
}

func (o *countingObserver) OnConnect(transport string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connects[transport]++
}

func (o *countingObserver) OnDisconnect(transport string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disconnects[transport]++
}

func (o *countingObserver) snapshot(transport string) (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.connects[transport], o.disconnects[transport]
}

// injectObserver 是一个把 observer 注入 request context 的测试中间件。
func injectObserver(obs httpmiddleware.StreamObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(
			httpmiddleware.WithStreamObserver(c.Request.Context(), obs),
		)
		c.Next()
	}
}

func startTestServer(t *testing.T, srv *Server) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(100 * time.Millisecond)
	return ln
}

func TestSSE_ObserverConnectDisconnect(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	obs := newCountingObserver()
	srv.Use(injectObserver(obs))
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	srv.SSE("/events", func(s SSEStream) {
		_ = s.Data("hi")
	})

	ln := startTestServer(t, srv)
	defer func() { _ = ln.Close() }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/stream/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()

	time.Sleep(150 * time.Millisecond)
	c, d := obs.snapshot("sse")
	if c != 1 || d != 1 {
		t.Errorf("sse connect/disconnect = %d/%d, want 1/1", c, d)
	}
}

func TestWS_ObserverConnectDisconnect(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	obs := newCountingObserver()
	srv.Use(injectObserver(obs))
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	srv.WS("/ws", func(session WSSession) {
		// 立即返回，触发断开
	})

	ln := startTestServer(t, srv)
	defer func() { _ = ln.Close() }()

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/stream/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	time.Sleep(200 * time.Millisecond)
	c, d := obs.snapshot("ws")
	if c != 1 || d != 1 {
		t.Errorf("ws connect/disconnect = %d/%d, want 1/1", c, d)
	}
}
```

> 注：上面 `http.NewRequest` 那行仅为占位避免未用 import，实现时若无需要可删除并清理 import。

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./httpserver/ -run 'TestSSE_Observer|TestWS_Observer' -v`
Expected: FAIL（observer 从未被调用，connect/disconnect = 0/0）

- [ ] **Step 3: 写实现 — SSE handler 打标 + 埋点**

在 `httpserver/server.go` 的 `sseRegister` 内，`ginHandler` 函数体里，`stream.logConnect()`（约 `:231`）**之后**插入：

```go
		c.Set(utils.StreamingKey, "sse")
		if obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context()); ok && obs != nil {
			obs.OnConnect("sse")
			defer obs.OnDisconnect("sse")
		}
```

（`server.go` 已 import `utils` 与 `httpmiddleware`，无需新增 import。）

- [ ] **Step 4: 写实现 — WS handler 打标 + 埋点**

在 `httpserver/server.go` 的 `WS` 内，`ginHandler` 函数体里，成功 `upgrader.Upgrade` 且 `logStreamEvent(c, "info", "stream_connect", ...)`（约 `:276`）**之后**插入：

```go
		c.Set(utils.StreamingKey, "ws")
		if obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context()); ok && obs != nil {
			obs.OnConnect("ws")
			defer obs.OnDisconnect("ws")
		}
```

> 放在 `Upgrade` 成功之后，确保只对真正建立的连接计数；`defer` 与同函数内已有的 `defer conn.Close()` 一同在 handler 返回时执行，保证 panic 也能 OnDisconnect。

- [ ] **Step 5: 运行测试 → 确认通过**

Run: `go test ./httpserver/ -run 'TestSSE_Observer|TestWS_Observer' -v`
Expected: PASS

- [ ] **Step 6: 回归 + 编译**

Run: `go build ./... && go test ./httpserver/`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add httpserver/server.go httpserver/stream_observer_test.go
git commit -m "feat(httpserver): mark SSE/WS requests and emit stream observer events

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: WSSession.Get / GetString

**Files:**
- Modify: `httpserver/ws.go`（`WSSession` 接口 `:76-85`）
- Modify: `httpserver/ws_session.go`（`wsSession` struct `:11-24`，新增方法）
- Modify: `httpserver/server.go`（`WS` 内构造 `wsSession` 处 `:333-355`）
- Test: `httpserver/ws_get_test.go`（创建）

**Interfaces:**
- Produces: `WSSession.Get(key string) (any, bool)`, `WSSession.GetString(key string) (string, bool)`，行为镜像 `SSEStream.Get`/`GetString`

- [ ] **Step 1: 写失败测试**

在 `httpserver/ws_get_test.go`：

```go
package httpserver

import (
	"net"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWS_GetFromGinKeys(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	// 模拟鉴权中间件：c.Set("user", ...)
	srv.Use(func(c *gin.Context) {
		c.Set("user", "alice")
		c.Next()
	})
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	got := make(chan string, 1)
	srv.WS("/ws", func(session WSSession) {
		if v, ok := session.GetString("user"); ok {
			got <- v
		} else {
			got <- ""
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/stream/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case v := <-got:
		if v != "alice" {
			t.Errorf("GetString(user) = %q, want %q", v, "alice")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not run")
	}
}
```

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./httpserver/ -run TestWS_GetFromGinKeys -v`
Expected: FAIL（`session.GetString undefined`，编译失败）

- [ ] **Step 3: 写实现 — WSSession 接口加方法**

在 `httpserver/ws.go` 的 `WSSession` interface（`:76-85`）中，`Param(name string) string` 之后加两行：

```go
	Get(key string) (any, bool)
	GetString(key string) (string, bool)
```

- [ ] **Step 4: 写实现 — wsSession 加 keys 字段与方法**

在 `httpserver/ws_session.go` 的 `wsSession` struct（`:11-24`）中，`params  gin.Params` 之后加：

```go
	keys map[string]any
```

并在 `Param` 方法（`:34-36`）之后新增：

```go
func (s *wsSession) Get(key string) (any, bool) {
	v, ok := s.keys[key]
	return v, ok
}

func (s *wsSession) GetString(key string) (string, bool) {
	v, ok := s.keys[key]
	if !ok {
		return "", false
	}
	str, ok := v.(string)
	return str, ok
}
```

- [ ] **Step 5: 写实现 — server.go 快照 c.Keys**

在 `httpserver/server.go` 的 `WS` 内构造 `session := &wsSession{...}`（`:333`）处，新增 `keys` 字段赋值。在 `params:  c.Params,` 之后加：

```go
			keys:    cloneGinKeys(c.Keys),
```

并在 `server.go` 文件内（建议紧邻 `ContextFromGin` 函数后，文件末尾）新增辅助函数：

```go
// cloneGinKeys 复制 gin.Context.Keys 快照，供 WS pump goroutine 安全读取。
func cloneGinKeys(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
```

> 快照时机：`WS` 的 ginHandler 同步执行，此刻鉴权中间件已写完 `c.Keys`；快照后 pump 在 goroutine 读 `s.keys`，不触碰 live `gin.Context`。

- [ ] **Step 6: 运行测试 → 确认通过**

Run: `go test ./httpserver/ -run TestWS_GetFromGinKeys -v`
Expected: PASS

- [ ] **Step 7: 回归 + 编译**

Run: `go build ./... && go test ./httpserver/`
Expected: 全部 PASS（含既有 WS 测试）

- [ ] **Step 8: Commit**

```bash
git add httpserver/ws.go httpserver/ws_session.go httpserver/server.go httpserver/ws_get_test.go
git commit -m "feat(httpserver): add WSSession.Get/GetString mirroring SSEStream

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: AccessLog 跳过流式汇总条目

**Files:**
- Modify: `httpserver/middleware/access_log.go`（`AccessLog` 返回的 handler，`c.Next()` 后 `:133-161`）
- Test: `httpserver/middleware/access_log_test.go`（追加）

**Interfaces:**
- Consumes: `utils.StreamingKey`（Task 1）
- Produces: 当 `c.GetString(utils.StreamingKey) != ""` 时，`AccessLog` 不输出 `access_log` 与 `payload_log` 事件

- [ ] **Step 1: 写失败测试**

在 `httpserver/middleware/access_log_test.go` 追加：

```go
func TestAccessLog_SkipsStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var events []string
	logger := func(_ context.Context, _ string, event string, _ map[string]any) {
		events = append(events, event)
	}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{Logger: logger}))
	engine.GET("/stream", func(c *gin.Context) {
		c.Set(utils.StreamingKey, "sse")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	engine.ServeHTTP(w, req)

	for _, e := range events {
		if e == accessLogEvent || e == payloadLogEvent {
			t.Errorf("streaming request should not emit %q", e)
		}
	}
}

func TestAccessLog_EmitsForNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var events []string
	logger := func(_ context.Context, _ string, event string, _ map[string]any) {
		events = append(events, event)
	}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{Logger: logger}))
	engine.GET("/normal", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/normal", nil)
	engine.ServeHTTP(w, req)

	found := false
	for _, e := range events {
		if e == accessLogEvent {
			found = true
		}
	}
	if !found {
		t.Error("non-streaming request should emit access_log")
	}
}
```

> 确认 `access_log_test.go` 顶部 import 含 `context`, `net/http`, `net/http/httptest`, `testing`, `github.com/gin-gonic/gin`, `github.com/tsopia/go-kit/utils`。缺则补上。

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./httpserver/middleware/ -run 'TestAccessLog_SkipsStreaming|TestAccessLog_EmitsForNonStreaming' -v`
Expected: `TestAccessLog_SkipsStreaming` FAIL（仍输出 access_log）；`TestAccessLog_EmitsForNonStreaming` PASS

- [ ] **Step 3: 写实现**

在 `httpserver/middleware/access_log.go` 的 `AccessLog` 返回 handler 中，`c.Next()`（`:133`）之后、构建 `status` 与 `fields`（`:135`）之前插入：

```go
		if c.GetString(utils.StreamingKey) != "" {
			return
		}
```

（`access_log.go` 已 import `github.com/tsopia/go-kit/utils`，无需新增。）

- [ ] **Step 4: 运行测试 → 确认通过**

Run: `go test ./httpserver/middleware/ -run 'TestAccessLog_SkipsStreaming|TestAccessLog_EmitsForNonStreaming' -v`
Expected: 两者均 PASS

- [ ] **Step 5: 回归**

Run: `go test ./httpserver/middleware/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver/middleware/access_log.go httpserver/middleware/access_log_test.go
git commit -m "feat(httpserver): skip access_log summary for streaming connections

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: prometheus 跳过流式请求指标 + 活跃连接 gauge

**Files:**
- Modify: `httpserver/observability/prometheus/middleware.go`（`Collector` struct `:24-37`，`Middleware` `:45-76`，`render` `:100-140`）
- Test: `httpserver/observability/prometheus/middleware_stream_test.go`（创建）

**Interfaces:**
- Consumes: `utils.StreamingKey`, `middleware.StreamObserver`, `middleware.WithStreamObserver`（Task 1）
- Produces:
  - `Collector` 维护 `streamGauge map[string]int64`
  - `(*Collector).IncStream(transport string)` / `DecStream(transport string)`
  - `Middleware()` 注入实现了 `StreamObserver` 的对象到 request context；流式请求不计入 `http_requests_total` / `http_request_duration_seconds_sum`
  - `/metrics` 渲染 `streaming_active_connections{transport="..."}`

- [ ] **Step 1: 写失败测试**

在 `httpserver/observability/prometheus/middleware_stream_test.go`：

```go
package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"
)

func TestMiddleware_SkipsStreamingRequestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := NewCollector()
	engine := gin.New()
	engine.Use(collector.Middleware())
	engine.GET("/stream", func(c *gin.Context) {
		c.Set(utils.StreamingKey, "sse")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	if strings.Contains(collector.render(), `route="/stream"`) {
		t.Error("streaming request should not appear in request metrics")
	}
}

func TestMiddleware_InjectsStreamObserver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := NewCollector()
	engine := gin.New()
	engine.Use(collector.Middleware())
	engine.GET("/stream", func(c *gin.Context) {
		obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context())
		if !ok {
			t.Error("expected StreamObserver in context")
			return
		}
		obs.OnConnect("ws")
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	body := collector.render()
	if !strings.Contains(body, `streaming_active_connections{transport="ws"} 1`) {
		t.Errorf("render missing active ws gauge:\n%s", body)
	}
}

func TestCollector_StreamGaugeIncDec(t *testing.T) {
	collector := NewCollector()
	collector.IncStream("sse")
	collector.IncStream("sse")
	collector.DecStream("sse")

	if !strings.Contains(collector.render(), `streaming_active_connections{transport="sse"} 1`) {
		t.Errorf("expected sse gauge = 1:\n%s", collector.render())
	}
}
```

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./httpserver/observability/prometheus/ -run 'TestMiddleware_SkipsStreaming|TestMiddleware_InjectsStreamObserver|TestCollector_StreamGauge' -v`
Expected: FAIL（`IncStream undefined` 等，编译失败）

- [ ] **Step 3: 写实现 — Collector 加 streamGauge**

在 `middleware.go` 的 `Collector` struct（`:24-28`）改为：

```go
type Collector struct {
	mu          sync.RWMutex
	metrics     map[metricKey]requestMetric
	streamGauge map[string]int64
}
```

`NewCollector`（`:33-37`）改为：

```go
func NewCollector() *Collector {
	return &Collector{
		metrics:     make(map[metricKey]requestMetric),
		streamGauge: make(map[string]int64),
	}
}
```

- [ ] **Step 4: 写实现 — IncStream/DecStream + observer adapter**

在 `middleware.go` 的 `observe` 方法（`:78-86`）之后新增：

```go
// IncStream 活跃流式连接 +1。
func (c *Collector) IncStream(transport string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamGauge[transport]++
}

// DecStream 活跃流式连接 -1，归零时移除键以保持 render 干净。
func (c *Collector) DecStream(transport string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamGauge[transport]--
	if c.streamGauge[transport] <= 0 {
		delete(c.streamGauge, transport)
	}
}

type collectorStreamObserver struct {
	collector *Collector
}

func (o collectorStreamObserver) OnConnect(transport string)    { o.collector.IncStream(transport) }
func (o collectorStreamObserver) OnDisconnect(transport string) { o.collector.DecStream(transport) }
```

- [ ] **Step 5: 写实现 — Middleware 注入 observer + 跳过流式**

`middleware.go` 顶部 import 块新增：

```go
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"
```

`(*Collector).Middleware`（`:45-76`）改为：

```go
func (c *Collector) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()

		ctx.Request = ctx.Request.WithContext(
			httpmiddleware.WithStreamObserver(ctx.Request.Context(), collectorStreamObserver{collector: c}),
		)

		ctx.Next()

		if ctx.GetString(utils.StreamingKey) != "" {
			return
		}

		route := ctx.FullPath()
		if route == "" {
			route = ctx.Request.URL.Path
		}
		if route == "" {
			route = "/"
		}

		key := metricKey{
			method: ctx.Request.Method,
			route:  route,
			status: http.StatusText(ctx.Writer.Status()),
		}
		if key.status == "" {
			key.status = http.StatusText(http.StatusOK)
		}
		key.status = strings.TrimSpace(key.status)
		key.status = strings.ReplaceAll(key.status, " ", "_")

		c.observe(metricKey{
			method: key.method,
			route:  key.route,
			status: statusCodeString(ctx.Writer.Status()),
		}, time.Since(startedAt))
	}
}
```

- [ ] **Step 6: 写实现 — render 追加 gauge**

在 `middleware.go` 的 `render`（`:100-140`）中，`return builder.String()` 之前插入：

```go
	c.mu.RLock()
	gauge := make(map[string]int64, len(c.streamGauge))
	for transport, value := range c.streamGauge {
		gauge[transport] = value
	}
	c.mu.RUnlock()

	transports := make([]string, 0, len(gauge))
	for transport := range gauge {
		transports = append(transports, transport)
	}
	sort.Strings(transports)

	builder.WriteString("# HELP streaming_active_connections Currently active streaming connections.\n")
	builder.WriteString("# TYPE streaming_active_connections gauge\n")
	for _, transport := range transports {
		builder.WriteString(`streaming_active_connections{transport="`)
		builder.WriteString(escapeLabelValue(transport))
		builder.WriteString(`"} `)
		builder.WriteString(strconv.FormatInt(gauge[transport], 10))
		builder.WriteString("\n")
	}
```

`middleware.go` 顶部 import 新增 `"strconv"`（`sort` 已存在）。

- [ ] **Step 7: 运行测试 → 确认通过**

Run: `go test ./httpserver/observability/prometheus/ -v`
Expected: 全部 PASS（含既有 `TestRegisterMetricsRoute`）

- [ ] **Step 8: 编译 + 依赖方向检查**

Run: `go build ./... && go vet ./httpserver/observability/prometheus/`
Expected: PASS（确认 prometheus → middleware/utils 单向依赖成立，无 import 环）

- [ ] **Step 9: Commit**

```bash
git add httpserver/observability/prometheus/middleware.go httpserver/observability/prometheus/middleware_stream_test.go
git commit -m "feat(httpserver): exclude streaming from request metrics, add active-connection gauge

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: otel 流式 span 标注 + 不误判 error

**Files:**
- Modify: `httpserver/observability/otel/middleware.go`（`Middleware` 返回 handler `:40-61`）
- Test: `httpserver/observability/otel/middleware_stream_test.go`（创建）

**Interfaces:**
- Consumes: `utils.StreamingKey`（Task 1）
- Produces: 流式请求的 span 设置 attribute `stream.transport`；流式请求不因 `status>=500` 之外的初始状态被误判，且对 `200/101` 不标记 error

- [ ] **Step 1: 写失败测试**

先确认 otel 既有测试如何构造 exporter。在 `httpserver/observability/otel/middleware_stream_test.go`：

```go
package otel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMiddleware_StreamingSpanAttribute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))

	engine := gin.New()
	engine.Use(Middleware(Config{TracerProvider: tp}))
	engine.GET("/stream", func(c *gin.Context) {
		c.Set(utils.StreamingKey, "sse")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}

	var hasAttr bool
	for _, attr := range spans[0].Attributes {
		if string(attr.Key) == "stream.transport" && attr.Value.AsString() == "sse" {
			hasAttr = true
		}
	}
	if !hasAttr {
		t.Errorf("span missing stream.transport=sse attribute: %+v", spans[0].Attributes)
	}
	if spans[0].Status.Code == codes.Error {
		t.Error("streaming span should not be marked error for 200")
	}
}
```

> 若 `tracetest`/`sdk/trace` 不在 `go.sum`，先确认 otel 既有测试（`middleware_test.go`）使用的 exporter 方式并对齐；如已有 helper 则复用之。运行 `go test` 前若报缺失依赖，执行 `go get go.opentelemetry.io/otel/sdk@latest` 并 `go mod tidy`。

- [ ] **Step 2: 运行测试 → 确认失败**

Run: `go test ./httpserver/observability/otel/ -run TestMiddleware_StreamingSpanAttribute -v`
Expected: FAIL（span 无 `stream.transport` attribute）

- [ ] **Step 3: 写实现**

在 `otel/middleware.go` 的 handler（`:40-61`）中，`c.Next()`（`:49`）之后、`if len(c.Errors) > 0`（`:51`）之前插入：

```go
		if transport := c.GetString(utils.StreamingKey); transport != "" {
			span.SetAttributes(attribute.String("stream.transport", transport))
			if len(c.Errors) > 0 {
				lastErr := c.Errors.Last().Err
				span.RecordError(lastErr)
				span.SetStatus(codes.Error, lastErr.Error())
			}
			return
		}
```

`otel/middleware.go` 顶部 import 新增：

```go
	"go.opentelemetry.io/otel/attribute"
	"github.com/tsopia/go-kit/utils"
```

> 流式分支提前 return，跳过 `:58-60` 基于 `status>=500` 的判定（SSE 200 / WS 101 不应被当作 error）；同时保留 `c.Errors` 非空时的 error 记录。

- [ ] **Step 4: 运行测试 → 确认通过**

Run: `go test ./httpserver/observability/otel/ -run TestMiddleware_StreamingSpanAttribute -v`
Expected: PASS

- [ ] **Step 5: 回归 + 编译**

Run: `go build ./... && go test ./httpserver/observability/otel/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add httpserver/observability/otel/middleware.go httpserver/observability/otel/middleware_stream_test.go
git commit -m "feat(httpserver): tag streaming spans and avoid false error status

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: 文档（三层模型 + 标准范式）

**Files:**
- Modify: `httpserver/README.md`（新增"流式连接与中间件"章节）
- Modify: `httpserver/doc.go`（补充包级说明）

**Interfaces:**
- Consumes: 全部前序 Task 的最终 API（`SSEStream.Get`, `WSSession.Get`, `MarkStreaming`, prometheus gauge）

- [ ] **Step 1: 在 README.md 新增章节**

在 `httpserver/README.md` 的"核心能力"列表之后、合适位置插入：

````markdown
## 流式连接（SSE / WebSocket）与中间件

SSE/WS 与普通 HTTP 请求共享同一套 Gin 中间件链，遵循三层模型：

| 层 | 作用范围 | 说明 |
|----|---------|------|
| engine 中间件（`srv.Use`） | **所有路由**，含 SSE/WS | Recovery、鉴权、TraceID、RequestID 等。前提：在注册路由前 `Use` |
| Timeout | 仅 `regularGroup` | SSE/WS 走 `streamingGroup`，自动豁免请求级超时 |
| 可观测 | SSE/WS 专属 | 每个连接产生 2 条结构化日志（`stream_connect` / `stream_disconnect`），不产生 `access_log` 与请求延迟指标 |

### 鉴权数据共享

鉴权中间件用 `c.Set("user", u)` 写入的数据，SSE/WS handler 均可读取：

```go
srv.Use(authMiddleware) // 内部 c.Set("user", u)

srv.SSE("/events", func(s httpserver.SSEStream) {
	user, ok := s.Get("user")
	_ = user; _ = ok
})

srv.WS("/chat", func(session httpserver.WSSession) {
	user, ok := session.GetString("user")
	_ = user; _ = ok
})
```

### 流式指标

挂载 `prometheus.Middleware()` 后，`/metrics` 暴露
`streaming_active_connections{transport="sse|ws"}` 活跃连接数 gauge；
流式连接不会污染 `http_requests_total` 与 `http_request_duration_seconds_sum`。

### 自定义流式路由

通过 `srv.StreamingGroup()` 自行注册的流式路由，需用 `middleware.MarkStreaming("sse"|"ws")`
打标，才能享受上述观测层处理（`SSE`/`SSEPost`/`WS` 便利方法已自动打标）。
````

- [ ] **Step 2: 在 doc.go 补充包级说明**

在 `httpserver/doc.go` 适当位置补一段（与既有风格一致）：

```go
// # 流式连接
//
// SSE/WS 与普通请求共享同一中间件链。流式 handler 自动标记 utils.StreamingKey，
// 观测型中间件（AccessLog/prometheus/otel）据此跳过请求级汇总，改用连接级
// 日志与活跃连接 gauge。鉴权中间件经 c.Set 写入的数据可经 SSEStream.Get /
// WSSession.Get 读取。
```

- [ ] **Step 3: 校验文档可构建（doc.go 语法）**

Run: `go build ./httpserver/`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add httpserver/README.md httpserver/doc.go
git commit -m "docs(httpserver): document streaming three-layer model and middleware sharing

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: 全量验证

**Files:** 无新增，仅验证。

- [ ] **Step 1: 全量构建**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 2: 全量测试**

Run: `go test ./httpserver/... ./utils/`
Expected: 全部 PASS

- [ ] **Step 3: vet**

Run: `go vet ./httpserver/...`
Expected: 无输出

- [ ] **Step 4: 依赖方向终检**

Run: `go list -deps ./httpserver | grep observability || echo "core package clean"`
Expected: `core package clean`（httpserver 核心包不依赖任何 observability 子包）

- [ ] **Step 5（可选）：lint**

Run: `golangci-lint run ./httpserver/... 2>/dev/null || echo "golangci-lint not available, skipped"`

---

## 自检报告（Self-Review）

**Spec 覆盖**：
- §3.1 流式标记 → Task 1（StreamingKey/MarkStreaming）+ Task 2（SSE/WS 打标）✓
- §3.2 AccessLog 感知 → Task 4 ✓
- §3.3 prometheus 跳过 + gauge + StreamObserver → Task 1（抽象）+ Task 2（埋点）+ Task 5（实现）✓
- §3.4 otel 感知 → Task 6 ✓
- §3.5 WSSession.Get → Task 3 ✓
- §3.6 文档 → Task 7 ✓
- §6 风险 R2 nil-safe → Task 2 埋点判空；R3 gauge 泄漏 → Task 2 defer OnDisconnect；依赖方向 → Task 5/8 校验 ✓

**类型一致性**：
- `StreamObserver.OnConnect/OnDisconnect(transport string)` 在 Task 1 定义，Task 2 调用、Task 5 实现（`collectorStreamObserver`）签名一致 ✓
- `utils.StreamingKey`（string）在 Task 1/2/4/5/6 一致使用 `c.GetString` 读取 ✓
- `WSSession.Get(key string) (any, bool)` / `GetString(key string) (string, bool)` 在 Task 3 接口与实现一致，镜像 `SSEStream`（`sse.go:53-54`）✓

**占位符扫描**：无 TBD/TODO；测试 import 均为实际使用，无整洁占位 hack ✓
