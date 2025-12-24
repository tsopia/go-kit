# HTTP 服务器改进功能演示

本示例展示了 `httpserver` 包的改进功能，包括**便利的路由注册方法**和**自动优雅关闭机制**。

## 🚀 新增功能

### 1. 便利的路由注册方法

在保持原有 `Engine()` 方法完全可用的基础上，新增了便利的路由注册方法：

```go
server := httpserver.NewServer(nil)

// 新的便利方法
server.GET("/users", handler)
server.POST("/users", handler)
server.PUT("/users/:id", handler)
server.DELETE("/users/:id", handler)
server.PATCH("/users/:id/status", handler)
server.HEAD("/ping", handler)
server.OPTIONS("/options", handler)
server.Any("/any", handler)

// 中间件和路由组
server.Use(middleware...)
api := server.Group("/api/v1")

// 原始方法仍然完全支持
engine := server.Engine()
engine.GET("/old-way", handler)
```

### 2. 自动优雅关闭机制

新增了内置信号处理的优雅关闭功能：

```go
server := httpserver.NewServer(nil)

// 注册路由...
server.GET("/health", healthHandler)

// 自动处理 SIGINT 和 SIGTERM 信号
if err := server.RunWithGracefulShutdown(); err != nil {
    log.Fatal(err)
}
```

#### 支持的方法：

- `RunWithGracefulShutdown()` - 启动服务器并自动处理优雅关闭（推荐）
- `WaitForShutdown()` - 等待关闭信号并执行优雅关闭
- `Shutdown(ctx)` - 手动优雅关闭（原有功能）

## 🎯 设计原则

### 向后兼容

所有原有的API都完全保持兼容：

```go
// 原有方式仍然完全支持
server := httpserver.NewServer(nil)
engine := server.Engine()
engine.GET("/path", handler)
engine.Use(middleware...)
```

### 可选便利

新增的便利方法是**完全可选的**：

```go
// 方式1：使用便利方法（推荐）
server.GET("/users", handler)
server.Use(middleware...)

// 方式2：使用原始方法（仍然支持）
engine := server.Engine()
engine.GET("/users", handler)
engine.Use(middleware...)

// 方式3：混合使用（完全可以）
server.Use(commonMiddleware...)        // 便利方法
engine := server.Engine()              // 获取引擎
api := engine.Group("/api/v1")         // 使用原始方法
server.GET("/health", healthHandler)   // 再次使用便利方法
```

### 最小化封装

- ✅ 不改变原有的设计哲学
- ✅ 不强制使用新功能
- ✅ 不增加额外的复杂性
- ✅ 完全暴露 Gin 的原生功能

## 🧪 运行演示

```bash
cd examples/http-server-improved
go run main.go
```

**启动后你会看到：**

```
=== HTTP 服务器改进功能演示 ===
1. 演示便利的路由注册方法
2. 演示自动优雅关闭功能
服务器启动中...
- 访问地址: http://localhost:8080
- 使用 Ctrl+C 或发送 SIGTERM 信号来优雅关闭服务器

可用端点:
- GET /health - 健康检查
- POST /users - 创建用户
- PUT /users/123 - 更新用户
- DELETE /users/123 - 删除用户
- PATCH /users/123/status - 更新用户状态
- GET /api/v1/users - 用户列表
- GET /api/v1/users/123 - 获取用户详情
```

## 🔬 功能测试

### 1. 测试路由注册

```bash
# 健康检查
curl http://localhost:8080/health

# 创建用户
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "张三", "email": "zhangsan@example.com"}'

# 更新用户
curl -X PUT http://localhost:8080/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name": "张三updated", "email": "updated@example.com"}'

# 更新用户状态
curl -X PATCH http://localhost:8080/users/123/status \
  -H "Content-Type: application/json" \
  -d '{"status": "inactive"}'

# 删除用户
curl -X DELETE http://localhost:8080/users/123

# 获取用户列表
curl http://localhost:8080/api/v1/users

# 获取单个用户
curl http://localhost:8080/api/v1/users/456
```

### 2. 测试优雅关闭

在服务器运行时，按 `Ctrl+C` 或发送 `SIGTERM` 信号：

```bash
# 发送 SIGTERM 信号
kill -TERM <进程ID>
```

**预期输出：**
```
^C收到关闭信号，开始优雅关闭服务器...
服务器已优雅关闭
程序退出
```

## 📋 使用模式对比

### 模式1：完全使用便利方法（推荐用于简单场景）

```go
server := httpserver.NewServer(nil)

// 中间件
server.Use(gin.Logger())
server.Use(gin.Recovery())
server.Use(httpserver.TraceIDMiddleware())

// 路由
server.GET("/health", healthHandler)
server.POST("/users", createUserHandler)
api := server.Group("/api/v1")
api.GET("/users", listUsersHandler)

// 自动优雅关闭
if err := server.RunWithGracefulShutdown(); err != nil {
    log.Fatal(err)
}
```

### 模式2：完全使用原始方法（适用于复杂定制）

```go
server := httpserver.NewServer(nil)
engine := server.Engine()

// 复杂的中间件配置
engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{...}))
engine.Use(gin.CustomRecovery(customRecoveryHandler))
engine.Use(customMiddleware())

// 复杂的路由配置
v1 := engine.Group("/api/v1")
v1.Use(authMiddleware())
{
    users := v1.Group("/users")
    users.GET("", listUsersHandler)
    users.POST("", createUserHandler)
}

// 手动优雅关闭
go func() {
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}()

// 自定义信号处理
// ...
```

### 模式3：混合使用（平衡简洁性和灵活性）

```go
server := httpserver.NewServer(nil)

// 使用便利方法添加通用中间件
server.Use(gin.Logger())
server.Use(gin.Recovery())

// 使用原始方法进行复杂配置
engine := server.Engine()
api := engine.Group("/api/v1")
api.Use(authMiddleware())

// 混合注册路由
server.GET("/health", healthHandler)           // 便利方法
api.GET("/users", listUsersHandler)           // 原始方法

// 使用便利的优雅关闭
if err := server.RunWithGracefulShutdown(); err != nil {
    log.Fatal(err)
}
```

## ✨ 改进总结

### 解决的问题

1. **优雅退出完整性**：
   - ✅ 添加了内置信号处理
   - ✅ 提供了 `RunWithGracefulShutdown()` 一键启动
   - ✅ 支持 `WaitForShutdown()` 灵活等待

2. **路由注册便利性**：
   - ✅ 添加了所有HTTP方法的便利函数
   - ✅ 支持 `Use()` 和 `Group()` 便利方法
   - ✅ 完全保持向后兼容

### 保持的优势

- ✅ **零强制策略** - 所有新功能都是可选的
- ✅ **最小化封装** - 不重复封装 Gin 功能
- ✅ **完全控制** - 原始 `Engine()` 方法完全可用
- ✅ **向后兼容** - 所有现有代码无需修改

这个改进在保持原有设计哲学的基础上，为常见使用场景提供了更便利的API，同时通过内置信号处理解决了优雅关闭的完整性问题。 