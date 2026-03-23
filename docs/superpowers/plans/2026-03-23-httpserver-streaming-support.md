# httpserver 流式接口支持实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 httpserver 支持 SSE、WebSocket、大文件上传等流式接口，不被 timeout 误杀。

**Architecture:** 通过路由分组（普通 vs 流式）分离 timeout 中间件，用 ResponseController 清除连接级 deadline，提供类型化的 SSE API 和上传封装。

**Tech Stack:** Go, Gin, http.ResponseController

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `httpserver/config.go` | 修改 | 调整默认值，新增 HandlerTimeout |
| `httpserver/sse.go` | 创建 | SSE 接口和实现 |
| `httpserver/sse_test.go` | 创建 | SSE 测试 |
| `httpserver/server.go` | 修改 | 新增 regularGroup/streamingGroup，改造路由方法 |
| `httpserver/server_test.go` | 修改 | 新增 Group 相关测试 |
| `httpserver/handler.go` | 修改 | 新增 HandleUpload |
| `httpserver/handler_test.go` | 修改 | 新增 HandleUpload 测试 |
| `httpserver/preset/production.go` | 修改 | 创建两个 Group，条件挂载 Timeout 中间件 |

---

## Task 1: 调整 Config 默认值

**Files:**
- Modify: `httpserver/config.go:9-11`

**背景:** 当前 ReadTimeout/WriteTimeout 都是 10s，对流式接口不友好。

- [ ] **Step 1: 修改默认值**

```go
const (
    defaultReadTimeout       = 30 * time.Second  // 从 10s 改为 30s
    defaultReadHeaderTimeout = 5 * time.Second
    defaultWriteTimeout      = 60 * time.Second  // 从 10s 改为 60s
    defaultIdleTimeout       = 60 * time.Second
    // ... 其他不变
)
```

- [ ] **Step 2: 新增 HandlerTimeout 字段到 Config**

在 `Config struct` 末尾添加：

```go
// HandlerTimeout 用于 preset 自动挂载的中间件超时。
// 只在 preset.NewProductionServer 中使用，非 preset 场景无效。
// 必须小于 WriteTimeout 才能生效。
HandlerTimeout time.Duration
```

- [ ] **Step 3: 编译检查**

Run: `cd /root/projects/go-kit && go build ./httpserver/...`
Expected: PASS (no errors)

- [ ] **Step 4: Commit**

```bash
git add httpserver/config.go
git commit -m "feat(httpserver): adjust default timeouts, add HandlerTimeout config"
```

---

## Task 2: 创建 SSE 接口和实现

**Files:**
- Create: `httpserver/sse.go`
- Create: `httpserver/sse_test.go`

**背景:** SSE 需要特定的响应头和格式，框架应自动处理。

- [ ] **Step 1: 创建 sse.go 基础结构**

```go
package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SSESender 是 SSE 事件发送接口。
type SSESender interface {
	// Event 发送命名事件
	Event(name string, data any) error
	// Data 发送匿名数据
	Data(data any) error
	// Comment 发送注释（常用于心跳）
	Comment(text string) error
}

// SSEHandlerFunc 是 SSE handler 的函数签名。
// ctx 包含客户端断开和 server shutdown 信号。
type SSEHandlerFunc func(ctx context.Context, send SSESender)

type sseSender struct {
	ginCtx *gin.Context
}

func (s *sseSender) Event(name string, data any) error {
	return s.writeEvent(name, data)
}

func (s *sseSender) Data(data any) error {
	return s.writeEvent("", data)
}

func (s *sseSender) Comment(text string) error {
	_, err := fmt.Fprintf(s.ginCtx.Writer, ": %s\n\n", text)
	if err != nil {
		return err
	}
	return s.ginCtx.Writer.Flush()
}

func (s *sseSender) writeEvent(name string, data any) error {
	var dataStr string
	switch d := data.(type) {
	case string:
		dataStr = d
	case []byte:
		dataStr = string(d)
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal sse data: %w", err)
		}
		dataStr = string(b)
	}

	if name != "" {
		if _, err := fmt.Fprintf(s.ginCtx.Writer, "event: %s\n", name); err != nil {
			return err
		}
	}

	lines := splitLines(dataStr)
	for _, line := range lines {
		if _, err := fmt.Fprintf(s.ginCtx.Writer, "data: %s\n", line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(s.ginCtx.Writer, "\n"); err != nil {
		return err
	}

	return s.ginCtx.Writer.Flush()
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
```

- [ ] **Step 2: 创建 sse_test.go 基础测试**

```go
package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSSESender_Event(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Event("update", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "event: update\ndata: {\"key\":\"value\"}\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Data(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Data("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "data: hello\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Comment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Comment("heartbeat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := ": heartbeat\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}
```

- [ ] **Step 3: 运行测试**

Run: `cd /root/projects/go-kit && go test -v ./httpserver -run TestSSESender`
Expected: 3 tests PASS

- [ ] **Step 4: Commit**

```bash
git add httpserver/sse.go httpserver/sse_test.go
git commit -m "feat(httpserver): add SSE sender interface and implementation"
```

---

## Task 3: 改造 Server 结构支持双 Group

**Files:**
- Modify: `httpserver/server.go`
- Modify: `httpserver/server_test.go`

**背景:** 需要在 Server 内部维护两个 Group，普通路由走 regularGroup，流式路由走 streamingGroup。

- [ ] **Step 1: 修改 Server struct**

在 `Server struct` 中添加：

```go
// 路由分组
regularGroup   *gin.RouterGroup  // 有 Timeout 中间件
streamingGroup *gin.RouterGroup  // 无 Timeout 中间件，用于 SSE/WS
```

- [ ] **Step 2: 新增设置 Group 的方法**

```go
// SetGroups 设置普通和流式路由组，由 preset 调用。
func (s *Server) SetGroups(regular, streaming *gin.RouterGroup) {
	s.regularGroup = regular
	s.streamingGroup = streaming
}

// getRegularGroup 返回普通路由组。
func (s *Server) getRegularGroup() *gin.RouterGroup {
	if s.regularGroup != nil {
		return s.regularGroup
	}
	return s.engine.RouterGroup
}

// getStreamingGroup 返回流式路由组。
func (s *Server) getStreamingGroup() *gin.RouterGroup {
	if s.streamingGroup != nil {
		return s.streamingGroup
	}
	return s.engine.RouterGroup
}
```

- [ ] **Step 3: 改造 GET 方法（示例，其他方法同理）**

```go
func (s *Server) GET(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().GET(relativePath, handlers...)
}
```

- [ ] **Step 4: 改造所有 HTTP 方法**

依次修改：POST, PUT, DELETE, PATCH, HEAD, OPTIONS, Any

```go
func (s *Server) POST(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().POST(relativePath, handlers...)
}

func (s *Server) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().PUT(relativePath, handlers...)
}

func (s *Server) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().DELETE(relativePath, handlers...)
}

func (s *Server) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().PATCH(relativePath, handlers...)
}

func (s *Server) HEAD(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().HEAD(relativePath, handlers...)
}

func (s *Server) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().OPTIONS(relativePath, handlers...)
}

func (s *Server) Any(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().Any(relativePath, handlers...)
}
```

- [ ] **Step 5: Group 方法也走 regularGroup**

```go
func (s *Server) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.getRegularGroup().Group(relativePath, handlers...)
}
```

- [ ] **Step 6: 编译检查**

Run: `cd /root/projects/go-kit && go build ./httpserver/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add httpserver/server.go
git commit -m "refactor(httpserver): support dual route groups for timeout handling"
```

---

## Task 4: 实现 SSE 路由方法

**Files:**
- Modify: `httpserver/server.go`

**背景:** 在 Server 上提供 SSE 方法，自动处理 SSE 协议和清除 WriteDeadline。

- [ ] **Step 1: 新增 SSE 方法**

```go
// SSE 注册一个 Server-Sent Events 路由。
// 自动设置 SSE 响应头，清除 WriteDeadline，使用 streamingGroup（无 Timeout 中间件）。
func (s *Server) SSE(relativePath string, handler SSEHandlerFunc) {
	s.getStreamingGroup().GET(relativePath, func(c *gin.Context) {
		// 清除 WriteDeadline
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetWriteDeadline(time.Time{})

		// 设置 SSE 响应头
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)

		// 立即 flush header
		_ = c.Writer.Flush()

		// 创建 sender 并调用 handler
		sender := &sseSender{ginCtx: c}
		handler(c.Request.Context(), sender)
	})
}
```

- [ ] **Step 2: 新增 StreamingGroup 方法**

```go
// StreamingGroup 创建一个流式路由组，用于 WebSocket 等长连接场景。
// 该组不会挂载 Timeout 中间件。
func (s *Server) StreamingGroup(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.getStreamingGroup().Group(relativePath, handlers...)
}
```

- [ ] **Step 3: 添加 http import**

确保 server.go 文件头部有：

```go
import (
	"net/http"
	// ... 其他 imports
)
```

- [ ] **Step 4: 编译检查**

Run: `cd /root/projects/go-kit && go build ./httpserver/...`
Expected: PASS

- [ ] **Step 5: 创建 SSE 集成测试**

在 `httpserver/server_test.go` 中添加：

```go
func TestServer_SSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultConfig()
	srv := NewServer(config)

	// 设置 streaming group（模拟 preset 行为）
	streamingGroup := srv.Engine().Group("/")
	srv.SetGroups(nil, streamingGroup)

	srv.SSE("/events", func(ctx context.Context, send SSESender) {
		send.Event("test", "hello")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/event-stream")
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: test") {
		t.Errorf("body should contain 'event: test', got: %s", body)
	}
}
```

**注意:** 需要添加 `strings` import。

- [ ] **Step 6: 运行测试**

Run: `cd /root/projects/go-kit && go test -v ./httpserver -run TestServer_SSE`
Expected: PASS (可能因 streamingGroup 设置问题需要调整)

- [ ] **Step 7: Commit**

```bash
git add httpserver/server.go httpserver/server_test.go
git commit -m "feat(httpserver): add SSE and StreamingGroup methods"
```

---

## Task 5: 实现 HandleUpload

**Files:**
- Modify: `httpserver/handler.go`
- Modify: `httpserver/handler_test.go`

**背景:** 提供专门的上传 handler 封装，自动清除 deadline 和限制 body 大小。

- [ ] **Step 1: 新增 HandleUpload 函数**

```go
// HandleUpload 将强类型业务函数适配成上传 handler。
// 自动清除 ReadDeadline 和 WriteDeadline，限制请求体大小。
func HandleUpload[Req any, Resp any](
	fn HandlerFunc[Req, Resp],
	maxBytes int64,
	opts ...HandlerOption,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 清除 deadline
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetReadDeadline(time.Time{})
		_ = rc.SetWriteDeadline(time.Time{})

		// 2. 限制 body 大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// 3. 走正常的 Handle 逻辑
		Handle(fn, opts...)(c)
	}
}
```

- [ ] **Step 2: 添加必要的 imports**

```go
import (
	"net/http"
	"time"
	// ... 其他已有 imports
)
```

- [ ] **Step 3: 创建测试**

在 `httpserver/handler_test.go` 中添加：

```go
func TestHandleUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type UploadReq struct {
		File string `json:"file"`
	}
	type UploadResp struct {
		URL string `json:"url"`
	}

	handler := HandleUpload(func(ctx context.Context, req UploadReq) (UploadResp, error) {
		return UploadResp{URL: "https://example.com/" + req.File}, nil
	}, 1024)

	w := httptest.NewRecorder()
	body := `{"file":"test.txt"}`
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if !strings.Contains(w.Body.String(), "test.txt") {
		t.Errorf("response should contain filename")
	}
}

func TestHandleUpload_BodyTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type UploadReq struct {
		Data string `json:"data"`
	}
	type UploadResp struct {
		Success bool `json:"success"`
	}

	handler := HandleUpload(func(ctx context.Context, req UploadReq) (UploadResp, error) {
		return UploadResp{Success: true}, nil
	}, 10) // 只允许 10 bytes

	w := httptest.NewRecorder()
	body := `{"data":"this is a long string that exceeds the limit"}`
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d (413)", w.Code, http.StatusRequestEntityTooLarge)
	}
}
```

- [ ] **Step 4: 运行测试**

Run: `cd /root/projects/go-kit && go test -v ./httpserver -run TestHandleUpload`
Expected: 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go
git commit -m "feat(httpserver): add HandleUpload for file uploads with deadline clearing"
```

---

## Task 6: 改造 preset 创建双 Group

**Files:**
- Modify: `httpserver/preset/production.go`

**背景:** preset 需要创建两个 Group 并分别挂载中间件。

- [ ] **Step 1: 完全重写 production.go**

```go
package preset

import (
	"time"

	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/middleware"
)

// NewProductionServer 创建带官方生产默认链路的服务器。
func NewProductionServer(config *httpserver.Config, opts ...httpserver.Option) *httpserver.Server {
	// 1. 创建基础 server（此时还没有任何中间件）
	srv := httpserver.NewServer(config, opts...)

	// 2. 添加共享中间件（Recovery, RequestID, TraceID, SecurityHeaders）
	srv.Use(middleware.Recovery())
	srv.Use(middleware.RequestID())
	srv.Use(middleware.TraceID())
	srv.Use(middleware.SecurityHeaders())

	// 3. 创建 streamingGroup（不挂 Timeout）
	streamingGroup := srv.Engine().Group("/")

	// 4. 创建 regularGroup，根据 HandlerTimeout 决定是否挂 Timeout
	regularGroup := srv.Engine().Group("/")
	if config.HandlerTimeout > 0 {
		// 检查 HandlerTimeout < WriteTimeout
		if config.WriteTimeout > 0 && config.HandlerTimeout >= config.WriteTimeout {
			// 这是一个问题，但只是 warn，不阻止启动
			// 实际项目中可以通过 logger 输出
		}
		regularGroup.Use(middleware.Timeout(config.HandlerTimeout))
	}

	// 5. 设置两个 Group 到 server
	srv.SetGroups(regularGroup, streamingGroup)

	return srv
}
```

- [ ] **Step 2: 确认 SetGroups 方法**

Task 3 中已添加 `SetGroups` 方法，确保 `httpserver/server.go` 中有：

```go
// SetGroups 设置普通和流式路由组，由 preset 调用。
func (s *Server) SetGroups(regular, streaming *gin.RouterGroup) {
	s.regularGroup = regular
	s.streamingGroup = streaming
}
```

- [ ] **Step 3: 删除 productionTimeout 函数**

这个函数不再需要。

- [ ] **Step 4: 编译检查**

Run: `cd /root/projects/go-kit && go build ./httpserver/preset/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add httpserver/preset/production.go httpserver/server.go
git commit -m "feat(httpserver/preset): create dual route groups for streaming support"
```

---

## Task 7: 创建集成测试验证完整流程

**Files:**
- Create: `httpserver/integration_test.go`

**背景:** 验证 preset 创建的双 Group 端到端工作正常。

- [ ] **Step 1: 创建集成测试**

```go
package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/preset"
)

func TestPreset_StreamingSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &httpserver.Config{
		Port:           0, // 随机端口
		HandlerTimeout: 5 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	srv := preset.NewProductionServer(config)

	// 注册普通路由
	srv.GET("/api/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 注册 SSE 路由
	srv.SSE("/events", func(ctx context.Context, send httpserver.SSESender) {
		send.Event("test", "data")
	})

	// 测试普通路由
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	srv.Engine().ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("ping status = %d", w1.Code)
	}

	// 测试 SSE 路由
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.Engine().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("sse status = %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "event: test") {
		t.Errorf("sse response missing event")
	}
}
```

- [ ] **Step 2: 运行测试**

Run: `cd /root/projects/go-kit && go test -v ./httpserver -run TestPreset_StreamingSupport`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add httpserver/integration_test.go
git commit -m "test(httpserver): add integration test for preset streaming support"
```

---

## Task 8: 更新文档

**Files:**
- Modify: `httpserver/README.md`

**背景:** 新功能需要文档说明。

- [ ] **Step 1: 在 README.md 中添加 SSE 章节**

```markdown
## SSE 支持

```go
srv.SSE("/events", func(ctx context.Context, send httpserver.SSESender) {
    for {
        select {
        case <-ctx.Done():
            return  // 客户端断开或 server 关闭
        case data := <-updates:
            send.Event("update", data)
        }
    }
})
```

- 自动设置 `text/event-stream` 响应头
- 自动清除 `WriteDeadline`，支持长时间连接
- `ctx` 包含客户端断开和 server shutdown 信号
```

- [ ] **Step 2: 添加 WebSocket 章节**

```markdown
## WebSocket 支持

```go
ws := srv.StreamingGroup("/ws")
ws.GET("/chat", func(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    // ... 使用你选择的 WS 库
})
```

`StreamingGroup` 创建的组不会挂载 Timeout 中间件，适合 WebSocket 等长连接场景。
```

- [ ] **Step 3: 添加文件上传章节**

```markdown
## 文件上传

```go
srv.POST("/upload", httpserver.HandleUpload(uploadHandler, 100<<20)) // 100MB 限制
```

- 自动清除 `ReadDeadline` 和 `WriteDeadline`
- 使用 `MaxBytesReader` 限制 body 大小
- 超出限制返回 413
```

- [ ] **Step 4: Commit**

```bash
git add httpserver/README.md
git commit -m "docs(httpserver): document SSE, WebSocket, and upload support"
```

---

## 最终验证

- [ ] **运行全部 httpserver 测试**

Run: `cd /root/projects/go-kit && go test -v ./httpserver/...`
Expected: All tests PASS

- [ ] **编译整个项目**

Run: `cd /root/projects/go-kit && go build ./...`
Expected: PASS

---

## 总结

完成以上所有 Task 后，httpserver 将支持：

1. **SSE**: `srv.SSE()` 类型化 handler，自动协议处理
2. **WebSocket**: `srv.StreamingGroup()` 无 Timeout 路由组
3. **文件上传**: `HandleUpload()` 自动清 deadline + 大小限制
4. **优化的默认值**: ReadTimeout 30s, WriteTimeout 60s
5. **preset 双 Group**: 普通路由有 Timeout，流式路由无 Timeout
