# 回调函数式路由注册演示

本示例展示了 `pkg/httpserver` 包的新增功能：**回调函数式路由注册**，这是一种更加优雅和集中的路由管理方式。

## 🎯 设计理念

### 问题
传统的路由注册方式可能会导致：
- 路由分散在代码各处，难以整体了解API结构
- 需要多次调用 `server.GET()`, `server.POST()` 等方法
- 路由组织不够清晰

### 解决方案
回调函数式注册提供了：
- **集中化路由管理** - 所有路由在一个地方定义
- **清晰的层次结构** - 通过嵌套和分组清晰展示API结构
- **完整的Gin功能** - 直接访问Gin引擎的所有功能

## 🚀 核心功能

### RegisterRoutes 方法

```go
func (s *Server) RegisterRoutes(routes func(r *gin.Engine)) {
    routes(s.engine)
}
```

这个方法接收一个回调函数，该函数接收 `*gin.Engine` 参数，让你可以：
- 直接使用所有Gin的原生功能
- 创建复杂的路由组结构
- 添加路由级别的中间件
- 设置静态文件服务

## 💡 使用示例

### 基础用法

```go
server := httpserver.NewServer(nil)

server.RegisterRoutes(func(r *gin.Engine) {
    // 基础路由
    r.GET("/health", healthHandler)
    r.POST("/users", createUserHandler)
    
    // 路由组
    api := r.Group("/api/v1")
    {
        api.GET("/users", listUsersHandler)
        api.GET("/products", listProductsHandler)
    }
})
```

### 复杂结构示例（本示例实现）

```go
server.RegisterRoutes(func(r *gin.Engine) {
    // 基础路由
    r.GET("/health", healthCheckHandler)
    r.GET("/ping", pingHandler)

    // 用户相关路由
    userRoutes := r.Group("/users")
    {
        userRoutes.GET("", listUsersHandler)
        userRoutes.POST("", createUserHandler)
        userRoutes.GET("/:id", getUserHandler)
        userRoutes.PUT("/:id", updateUserHandler)
        userRoutes.DELETE("/:id", deleteUserHandler)
    }

    // API v1 路由组
    v1 := r.Group("/api/v1")
    {
        // 产品相关路由
        products := v1.Group("/products")
        {
            products.GET("", listProductsHandler)
            products.POST("", createProductHandler)
            products.GET("/:id", getProductHandler)
            products.PUT("/:id", updateProductHandler)
            products.DELETE("/:id", deleteProductHandler)
        }

        // 管理员路由（带中间件）
        admin := v1.Group("/admin")
        admin.Use(adminAuthMiddleware()) // 仅管理员可访问
        {
            admin.GET("/stats", getStatsHandler)
            admin.GET("/users", adminListUsersHandler)
            admin.DELETE("/users/:id", adminDeleteUserHandler)
        }
    }

    // 静态文件和WebSocket
    r.GET("/ws", websocketHandler)
    r.Static("/static", "./static")
    r.StaticFile("/favicon.ico", "./static/favicon.ico")
})
```

## 🧪 运行演示

### 启动服务器

```bash
cd examples/http-callback-routes
go run main.go
```

**启动后显示：**
```
=== 回调函数式路由注册演示 ===
使用回调函数式路由注册完成
注册的路由包括:
- GET /health - 基础健康检查
- GET /ping - 简单ping
- GET /users - 用户列表
- POST /users - 创建用户
- GET /api/v1/products - 产品列表
- GET /api/v1/admin/stats - 管理员统计（需认证）
- GET /ws - WebSocket连接
- GET /health/live - 存活检查
服务器启动在: http://localhost:8080
使用 Ctrl+C 优雅关闭服务器
```

### 测试路由

```bash
# 基础路由
curl http://localhost:8080/health
curl http://localhost:8080/ping

# 用户路由
curl http://localhost:8080/users
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "张三", "email": "zhangsan@example.com"}'
curl http://localhost:8080/users/123

# API v1 路由
curl http://localhost:8080/api/v1/products
curl http://localhost:8080/api/v1/products/456

# 管理员路由（需要认证）
curl http://localhost:8080/api/v1/admin/stats
# 未认证会返回401

curl -H "Authorization: Bearer admin-token" http://localhost:8080/api/v1/admin/stats
# 认证后可以访问

# 健康检查路由组
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/metrics
```

## 📊 路由注册方式对比

### 方式1：回调函数式（推荐用于复杂应用）

```go
server.RegisterRoutes(func(r *gin.Engine) {
    // 集中管理，结构清晰
    r.GET("/health", handler)
    
    api := r.Group("/api/v1")
    api.Use(middleware())
    {
        api.GET("/users", handler)
        api.POST("/users", handler)
    }
})
```

**优势：**
- ✅ 集中化路由管理
- ✅ 清晰的API结构展示
- ✅ 支持复杂的路由组织
- ✅ 完整的Gin功能支持

### 方式2：便利方法（适用于简单场景）

```go
server.GET("/health", handler)
server.POST("/users", handler)
api := server.Group("/api/v1")
api.GET("/users", handler)
```

**优势：**
- ✅ 代码简洁
- ✅ 适合简单应用
- ✅ 逐步添加路由

### 方式3：直接使用Engine（适用于高度定制）

```go
engine := server.Engine()
engine.GET("/health", handler)
// 完全的Gin控制权
```

**优势：**
- ✅ 最大的灵活性
- ✅ 所有Gin功能可用
- ✅ 无任何封装限制

## 🎨 最佳实践

### 1. 结构化组织

```go
server.RegisterRoutes(func(r *gin.Engine) {
    // 1. 基础路由
    r.GET("/health", healthHandler)
    r.GET("/ping", pingHandler)
    
    // 2. 按业务模块分组
    setupUserRoutes(r)
    setupProductRoutes(r)
    setupOrderRoutes(r)
    
    // 3. 管理功能
    setupAdminRoutes(r)
    
    // 4. 静态资源
    r.Static("/static", "./static")
})

func setupUserRoutes(r *gin.Engine) {
    users := r.Group("/users")
    {
        users.GET("", listUsers)
        users.POST("", createUser)
        // ...
    }
}
```

### 2. 中间件分层

```go
server.RegisterRoutes(func(r *gin.Engine) {
    // 全局中间件已在外部设置
    
    // API路由（公开）
    api := r.Group("/api/v1")
    {
        api.GET("/public", publicHandler)
    }
    
    // 需要认证的路由
    auth := r.Group("/api/v1")
    auth.Use(authMiddleware())
    {
        auth.GET("/profile", profileHandler)
    }
    
    // 管理员路由
    admin := r.Group("/api/v1/admin")
    admin.Use(authMiddleware(), adminMiddleware())
    {
        admin.GET("/stats", statsHandler)
    }
})
```

### 3. 模块化路由

```go
// routes/user.go
func SetupUserRoutes(r *gin.RouterGroup) {
    r.GET("", listUsers)
    r.POST("", createUser)
    r.GET("/:id", getUser)
}

// main.go
server.RegisterRoutes(func(r *gin.Engine) {
    v1 := r.Group("/api/v1")
    
    routes.SetupUserRoutes(v1.Group("/users"))
    routes.SetupProductRoutes(v1.Group("/products"))
    routes.SetupOrderRoutes(v1.Group("/orders"))
})
```

## ✨ 总结

回调函数式路由注册为Go-Kit的httpserver包提供了：

1. **更好的代码组织** - 集中化的路由管理
2. **清晰的API结构** - 一目了然的路由层次
3. **完整的功能支持** - 所有Gin功能都可使用
4. **灵活的选择** - 与其他注册方式完全兼容

这种方式特别适合：
- 复杂的API应用
- 需要清晰路由结构的项目
- 团队协作开发
- API文档生成需求

同时保持了Go-Kit一贯的设计哲学：**提供便利功能，但不限制用户选择**。 