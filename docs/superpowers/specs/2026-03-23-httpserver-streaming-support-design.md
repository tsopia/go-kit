# httpserver 流式接口支持设计文档

## 背景与问题

当前 httpserver 的 timeout 设计对流式接口（SSE、WebSocket、大文件上传）不友好：

1. `WriteTimeout = 10s` 会导致 SSE 连接在 10s 后被强制切断
2. `middleware.Timeout` 的 ctx cancel 会中断 SSE/WS handler
3. 大文件上传可能因 `ReadTimeout` 被截断

## 设计目标

让 httpserver 同时支持：
- 普通 REST 接口：有合理的超时保护
- SSE / WebSocket：不被 timeout 误杀，能长时间运行
- 大文件上传：不受网速限制，能传完大文件

## 方案概览

### 1. 调整默认 Timeout 值

```go
// config.go 默认值调整
const (
    defaultReadTimeout       = 30 * time.Second  // 10s → 30s
    defaultWriteTimeout      = 60 * time.Second  // 10s → 60s
    // 其他不变
)
```

**设计意图**：WriteTimeout > ReadTimeout，保证 handler 在 body 读取完成后还有充足时间处理。

### 2. Config 新增 HandlerTimeout

```go
type Config struct {
    // ... 现有字段
    HandlerTimeout time.Duration  // 应用层超时，preset 使用
}
```

- 只在 `preset.NewProductionServer` 中使用
- 非 preset 场景，此字段无效，超时完全由用户自己控制

### 3. 路由分组：普通 vs 流式

```
preset.NewProductionServer 内部：

1. engine.Use(Recovery, RequestID, TraceID, SecurityHeaders)
2. 创建 streamingGroup（继承上面中间件，不包含 Timeout）
3. 创建 regularGroup（继承上面中间件，加上 Timeout(HandlerTimeout)）

GET/POST/PUT...  → regularGroup  （有 Timeout 中间件）
SSE()            → streamingGroup（无 Timeout 中间件）
StreamingGroup() → streamingGroup（无 Timeout 中间件）
```

### 4. 新增 API

#### SSE

```go
// httpserver/server.go
func (s *Server) SSE(path string, handler SSEHandlerFunc)

type SSEHandlerFunc func(ctx context.Context, send SSESender)

type SSESender interface {
    Event(name string, data any) error
    Data(data any) error
    Comment(text string) error
}
```

**框架自动处理**：
- 设置 SSE 响应头：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`X-Accel-Buffering: no`
- 用 `http.ResponseController` 清除 `WriteDeadline`
- 每次发送后自动 `Flush`
- ctx 包含 client 断开和 server shutdown 信号

#### WebSocket

```go
// httpserver/server.go
func (s *Server) StreamingGroup(path string, handlers ...gin.HandlerFunc) *gin.RouterGroup
```

返回基于 `streamingGroup` 的子 Group，用于注册 WebSocket 路由。

**设计理由**：WS 库选择多样（gorilla/nhooyr/标准库），框架不封装 Upgrade 过程，只保证路由不被 Timeout 中间件干扰。

#### 文件上传

```go
// httpserver/handler.go
func HandleUpload[Req any, Resp any](
    fn HandlerFunc[Req, Resp],
    maxBytes int64,
    opts ...HandlerOption,
) gin.HandlerFunc
```

**框架自动处理**：
- 用 `http.ResponseController` 清除 `ReadDeadline` 和 `WriteDeadline`
- 用 `http.MaxBytesReader` 限制 body 大小

## 实现细节

### ResponseController 使用

```go
// SSE 内部
rc := http.NewResponseController(c.Writer)
_ = rc.SetWriteDeadline(time.Time{})  // 禁用 WriteTimeout

// Upload 内部
rc := http.NewResponseController(c.Writer)
_ = rc.SetReadDeadline(time.Time{})   // 禁用 ReadTimeout
_ = rc.SetWriteDeadline(time.Time{})  // 禁用 WriteTimeout
```

### Server 结构调整

```go
type Server struct {
    // ... 现有字段
    regularGroup   *gin.RouterGroup  // 有 Timeout 中间件
    streamingGroup *gin.RouterGroup  // 无 Timeout 中间件
}
```

### 路由注册方法改造

```go
func (s *Server) GET(path string, handlers ...gin.HandlerFunc) {
    if s.regularGroup != nil {
        s.regularGroup.GET(path, handlers...)
    } else {
        s.engine.GET(path, handlers...)
    }
}

// POST/PUT/DELETE 等同理
```

## 使用示例

### 标准用法（preset）

```go
config := &httpserver.Config{
    Port: 8080,
    HandlerTimeout: 55 * time.Second,  // 需 < WriteTimeout(60s)
}

srv := preset.NewProductionServer(config)

// 普通路由，有 Timeout 中间件保护
srv.GET("/api/users", httpserver.Handle(getUsers))

// SSE，无 Timeout 中间件，WriteDeadline 已清
srv.SSE("/events", func(ctx context.Context, send httpserver.SSESender) {
    for {
        select {
        case <-ctx.Done():
            return
        case data := <-updates:
            send.Event("update", data)
        }
    }
})

// WebSocket，基于 streamingGroup
ws := srv.StreamingGroup("/ws")
ws.GET("/chat", func(c *gin.Context) {
    conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
    // ...
})

// 上传，清 ReadDeadline + WriteDeadline
srv.POST("/upload", httpserver.HandleUpload(uploadHandler, 100<<20))
```

### 高级用法（不用 preset）

```go
config := httpserver.DefaultConfig()
srv := httpserver.NewServer(config)

// 完全自己控制中间件
srv.Use(middleware.Recovery())
srv.Use(middleware.Timeout(30 * time.Second))

// 想支持 SSE，手动清 deadline 或使用框架方法
srv.GET("/events", func(c *gin.Context) {
    rc := http.NewResponseController(c.Writer)
    rc.SetWriteDeadline(time.Time{})
    // ... SSE 逻辑
})
```

## 文件变更清单

| 文件 | 操作 | 变更内容 |
|------|------|---------|
| `httpserver/config.go` | 修改 | 调整默认值；新增 HandlerTimeout 字段 |
| `httpserver/server.go` | 修改 | 新增 regularGroup/streamingGroup；改造路由方法；新增 SSE/StreamingGroup 方法 |
| `httpserver/handler.go` | 修改 | 新增 HandleUpload |
| `httpserver/sse.go` | 新增 | SSEHandlerFunc、SSESender 接口及实现 |
| `httpserver/preset/production.go` | 修改 | 创建两个 Group；根据 HandlerTimeout 挂载中间件 |
| `httpserver/middleware/timeout.go` | 不变 | 保留现有实现，preset 里选择性使用 |

## 兼容性

- 现有代码使用 `httpserver.NewServer` + 手动 `Use(middleware.Timeout)` 不受影响
- 现有代码使用 `preset.NewProductionServer` 需确保 `HandlerTimeout < WriteTimeout`
- 新增 API 均为新增方法，不破坏现有接口
