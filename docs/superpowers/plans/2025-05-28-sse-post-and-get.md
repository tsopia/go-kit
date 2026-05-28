# SSEPost 与 SSEStream Get/GetString 实现计划

> **For agentic workers:** Use superpowers:executing-plans or inline execution. Steps use checkbox syntax.

**Goal:** 在 httpserver 包中新增 SSEPost 方法支持 POST 方式 SSE，以及在 SSEStream 接口中暴露 gin context keys 的读取能力。

**Architecture:** 改动 A 提取现有 SSE 的 handler 逻辑为私有方法 `sseRegister`，`SSE` 委托 GET，`SSEPost` 新增 POST。改动 B 在 `SSEStream` 接口和 `sseSender` 实现上新增 `Get`/`GetString`。

**Tech Stack:** Go, Gin, 已有 httpserver 包

---

## 文件结构

| 文件 | 操作 | 预计行数 | 责任 |
|------|------|----------|------|
| `httpserver/server.go` | 修改 | 现有 +12 | 提取 sseRegister，新增 SSEPost |
| `httpserver/sse.go` | 修改 | 现有 +14 | SSEStream 接口加 Get/GetString，sseSender 实现 |
| `httpserver/sse_test.go` | 修改 | 现有 +45 | SSEPost 路由注册测试，Get/GetString 测试 |

---

### Task 1: SSEPost + sseRegister 提取

**目标**: 提取 SSE handler 创建逻辑，新增 SSEPost 方法。

**文件变更**:
- 修改: `httpserver/server.go`
- 修改: `httpserver/sse_test.go`

**子步骤（TDD）:**

- [ ] **Step 1: 写失败测试**
```go
func TestServer_SSEPost_routeRegistered(t *testing.T) {
    server := NewServer(&Config{Port: 8080})
    server.SetGroups(
        server.Engine().Group("/api"),
        server.Engine().Group("/stream"),
    )
    server.SSEPost("/events", func(stream SSEStream) {
        <-stream.Context().Done()
    })
    routes := server.Engine().Routes()
    found := false
    for _, r := range routes {
        if r.Path == "/stream/events" && r.Method == "POST" {
            found = true
            break
        }
    }
    if !found {
        t.Error("SSEPost route not registered")
    }
}
```

- [ ] **Step 2: 运行测试 → 确认失败**
运行: `go test -v ./httpserver -run TestServer_SSEPost_routeRegistered`
预期: FAIL（SSEPost 未定义）

- [ ] **Step 3: 写最简实现**
在 `httpserver/server.go` 中，提取现有 `SSE` 方法中的 handler 逻辑到 `sseRegister`：
```go
func (s *Server) sseRegister(method, relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
    cfg := &sseConfig{}
    for _, opt := range opts {
        opt.apply(cfg)
    }

    ginHandler := func(c *gin.Context) {
        startedAt := time.Now()
        rc := http.NewResponseController(c.Writer)
        _ = rc.SetWriteDeadline(time.Time{})
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("X-Accel-Buffering", "no")
        c.Status(http.StatusOK)
        c.Writer.Flush()
        ctx, cancel := context.WithCancel(c.Request.Context())
        defer cancel()
        stream := &sseSender{ginCtx: c, ctx: ctx, config: cfg, startedAt: startedAt}
        stream.logConnect()
        stopHeartbeat := stream.startHeartbeat(ctx)
        handler(stream)
        stopHeartbeat()
        stream.logDisconnect(ctx)
    }

    switch method {
    case "POST":
        s.getStreamingGroup().POST(relativePath, ginHandler)
    default:
        s.getStreamingGroup().GET(relativePath, ginHandler)
    }
}
```
将现有 `SSE` 方法改为委托：
```go
func (s *Server) SSE(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
    s.sseRegister("GET", relativePath, handler, opts...)
}
```
新增 `SSEPost`：
```go
func (s *Server) SSEPost(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
    s.sseRegister("POST", relativePath, handler, opts...)
}
```

- [ ] **Step 4: 运行测试 → 确认通过**
运行: `go test -v ./httpserver -run TestServer_SSEPost_routeRegistered`
预期: PASS

- [ ] **Step 5: Commit**
```bash
git add httpserver/server.go httpserver/sse_test.go
git commit -m "feat(httpserver): add SSEPost method with shared sseRegister

Extract common SSE handler setup into sseRegister to support both
GET (existing SSE) and POST (new SSEPost) method registration."
```

---

### Task 2: SSEStream Get/GetString

**目标**: 给 SSEStream 接口和 sseSender 实现新增 Get/GetString 方法。

**文件变更**:
- 修改: `httpserver/sse.go`
- 修改: `httpserver/sse_test.go`

**子步骤（TDD）:**

- [ ] **Step 1: 写失败测试**
```go
func TestSSESender_Get(t *testing.T) {
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("user_id", "123")
    c.Set("count", 42)

    sender := &sseSender{ginCtx: c}

    val, ok := sender.Get("user_id")
    if !ok || val != "123" {
        t.Errorf("Get(user_id) = (%v, %v), want (123, true)", val, ok)
    }

    val, ok = sender.Get("count")
    if !ok || val != 42 {
        t.Errorf("Get(count) = (%v, %v), want (42, true)", val, ok)
    }

    _, ok = sender.Get("missing")
    if ok {
        t.Error("Get(missing) should return ok=false")
    }
}

func TestSSESender_Get_nilGinCtx(t *testing.T) {
    sender := &sseSender{ginCtx: nil}
    val, ok := sender.Get("key")
    if ok || val != nil {
        t.Errorf("Get with nil ginCtx = (%v, %v), want (nil, false)", val, ok)
    }
}

func TestSSESender_GetString(t *testing.T) {
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("user_id", "abc")
    c.Set("count", 42)

    sender := &sseSender{ginCtx: c}

    str, ok := sender.GetString("user_id")
    if !ok || str != "abc" {
        t.Errorf("GetString(user_id) = (%q, %v), want (abc, true)", str, ok)
    }

    _, ok = sender.GetString("count")
    if ok {
        t.Error("GetString(count) should return ok=false for non-string value")
    }

    _, ok = sender.GetString("missing")
    if ok {
        t.Error("GetString(missing) should return ok=false")
    }
}

func TestSSESender_GetString_nilGinCtx(t *testing.T) {
    sender := &sseSender{ginCtx: nil}
    str, ok := sender.GetString("key")
    if ok || str != "" {
        t.Errorf("GetString with nil ginCtx = (%q, %v), want (empty, false)", str, ok)
    }
}
```

- [ ] **Step 2: 运行测试 → 确认失败**
运行: `go test -v ./httpserver -run "TestSSESender_Get"`
预期: FAIL（Get/GetString 未定义）

- [ ] **Step 3: 写最简实现**
在 `httpserver/sse.go` 中，`SSEStream` 接口新增两行：
```go
type SSEStream interface {
    SSESender
    Context() context.Context
    Request() *http.Request
    Param(name string) string
    Get(key string) (any, bool)
    GetString(key string) (string, bool)
}
```

在 `sseSender` 上新增实现：
```go
func (s *sseSender) Get(key string) (any, bool) {
    if s.ginCtx == nil {
        return nil, false
    }
    return s.ginCtx.Get(key)
}

func (s *sseSender) GetString(key string) (string, bool) {
    val, ok := s.Get(key)
    if !ok {
        return "", false
    }
    str, ok := val.(string)
    return str, ok
}
```

- [ ] **Step 4: 运行测试 → 确认通过**
运行: `go test -v ./httpserver -run "TestSSESender_Get"`
预期: PASS

- [ ] **Step 5: Commit**
```bash
git add httpserver/sse.go httpserver/sse_test.go
git commit -m "feat(httpserver): add Get/GetString to SSEStream interface

Allow SSE handlers to read values injected by middleware into
the gin context (e.g. auth information via c.Set())."
```

---

### Task 3: 最终验证

- [ ] **编译检查**: `go build ./httpserver` → PASS
- [ ] **全部测试**: `go test ./httpserver -count=1` → PASS
- [ ] **Lint**: `golangci-lint run ./httpserver` → PASS（如有）
