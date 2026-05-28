# SSEPost 与 SSEStream Get/GetString

## 背景

当前 `Server.SSE` 仅支持 GET 方法注册 SSE 路由。当 SSE 输入较长时（如 chat completion 的 prompt），GET query 参数长度受限且不适合传递大段文本。同时，auth middleware 通过 `gin.Context.Set()` 注入的信息无法在 SSE handler 中读取。

## 改动 A：SSEPost + sseRegister 提取

提取现有 `SSE` 方法中的 handler 创建逻辑到私有方法 `sseRegister`，根据 method 参数注册 GET 或 POST 路由。

```go
func (s *Server) sseRegister(method, relativePath string, handler SSEHandlerFunc, opts ...SSEOption)

func (s *Server) SSE(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
    s.sseRegister("GET", relativePath, handler, opts...)
}

func (s *Server) SSEPost(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
    s.sseRegister("POST", relativePath, handler, opts...)
}
```

POST body 通过 `stream.Request().Body` 读取，无额外改动。

## 改动 B：SSEStream 接口暴露 Get/GetString

```go
type SSEStream interface {
    SSESender
    Context() context.Context
    Request() *http.Request
    Param(name string) string
    Get(key string) (any, bool)         // 新增
    GetString(key string) (string, bool) // 新增
}
```

实现：
```go
func (s *sseSender) Get(key string) (any, bool) {
    if s.ginCtx == nil { return nil, false }
    return s.ginCtx.Get(key)
}
func (s *sseSender) GetString(key string) (string, bool) {
    val, ok := s.Get(key)
    if !ok { return "", false }
    str, ok := val.(string)
    return str, ok
}
```

## Commit 策略

两个独立 commit：
1. `feat(httpserver): add SSEPost method with shared sseRegister`
2. `feat(httpserver): add Get/GetString to SSEStream interface`
