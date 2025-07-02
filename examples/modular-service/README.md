# 模块化服务架构演示

本示例展示了**企业级模块化服务架构**，完美体现了你描述的使用场景：
> 实现自己的接口，然后注册到一个统一的路由组里，最后调用server包，new一个gin客户端并且把路由注册传入

## 🏗️ 架构设计

### 核心理念
- **接口驱动**：每个服务模块都定义了清晰的接口
- **统一注册**：所有服务通过统一的注册器管理
- **路由分组**：服务自动注册到指定的路由组
- **回调注入**：通过回调函数将路由注册逻辑传入server

### 架构层次
```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Server                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │            Route Registry                       │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐     │   │
│  │  │   User    │ │  Product  │ │   Order   │ ... │   │
│  │  │  Service  │ │  Service  │ │  Service  │     │   │
│  │  └───────────┘ └───────────┘ └───────────┘     │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## 🎯 实现步骤

### 1. 定义服务接口
```go
// 业务接口
type UserService interface {
    ListUsers(c *gin.Context)
    CreateUser(c *gin.Context)
    GetUser(c *gin.Context)
    UpdateUser(c *gin.Context)
    DeleteUser(c *gin.Context)
}

// 路由注册接口
type RouteRegistrar interface {
    RegisterRoutes(group *gin.RouterGroup)
}
```

### 2. 实现服务
```go
type userServiceImpl struct {
    logger *logger.Logger
}

func (s *userServiceImpl) RegisterRoutes(group *gin.RouterGroup) {
    users := group.Group("/users")
    {
        users.GET("", s.ListUsers)
        users.POST("", s.CreateUser)
        users.GET("/:id", s.GetUser)
        users.PUT("/:id", s.UpdateUser)
        users.DELETE("/:id", s.DeleteUser)
    }
}

func (s *userServiceImpl) ListUsers(c *gin.Context) {
    // 业务逻辑实现
}
```

### 3. 创建服务注册器
```go
type ServiceRegistry struct {
    services []RouteRegistrar
}

func (r *ServiceRegistry) Register(service RouteRegistrar) {
    r.services = append(r.services, service)
}

func (r *ServiceRegistry) RegisterAllRoutes(group *gin.RouterGroup) {
    for _, service := range r.services {
        service.RegisterRoutes(group)
    }
}
```

### 4. 统一注册和启动
```go
func main() {
    // 1. 创建各个服务实例
    userService := NewUserService()
    productService := NewProductService()
    orderService := NewOrderService()
    authService := NewAuthService()

    // 2. 创建服务注册器，统一管理所有服务
    registry := NewServiceRegistry()
    registry.Register(userService.(RouteRegistrar))
    registry.Register(productService.(RouteRegistrar))
    registry.Register(orderService.(RouteRegistrar))
    registry.Register(authService.(RouteRegistrar))

    // 3. 创建HTTP服务器
    server := httpserver.NewServer(&httpserver.Config{
        Host: "0.0.0.0",
        Port: 8080,
    })

    // 4. 使用回调函数注册所有服务的路由
    server.RegisterRoutes(func(r *gin.Engine) {
        // API v1 路由组
        v1 := r.Group("/api/v1")
        {
            // 统一注册所有服务的路由
            registry.RegisterAllRoutes(v1)
        }
    })

    // 5. 启动服务器并自动处理优雅关闭
    if err := server.RunWithGracefulShutdown(); err != nil {
        log.Fatal("服务器启动失败:", err)
    }
}
```

## 🚀 运行演示

### 启动服务
```bash
cd examples/modular-service
go run main.go
```

**启动输出：**
```
=== 模块化服务架构演示 ===
每个服务实现自己的接口，统一注册到路由组
✅ 所有服务路由注册完成
📡 API接口列表:
   健康检查: GET /health
   用户服务: /api/v1/users/*
   产品服务: /api/v1/products/*
   订单服务: /api/v1/orders/*
   认证服务: /api/v1/auth/*
   管理后台: /admin/api/v1/*
🚀 服务器启动: http://localhost:8080
💡 使用 Ctrl+C 优雅关闭服务器
```

### 测试接口

#### 1. 健康检查
```bash
curl http://localhost:8080/health
```
**响应：**
```json
{
  "status": "healthy",
  "timestamp": 1703012345,
  "services": ["user", "product", "order", "auth"],
  "trace_id": "abc123...",
  "request_id": "def456..."
}
```

#### 2. 用户服务接口
```bash
# 获取用户列表
curl http://localhost:8080/api/v1/users

# 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "张三", "email": "zhangsan@example.com", "role": "admin"}'

# 获取单个用户
curl http://localhost:8080/api/v1/users/123

# 更新用户
curl -X PUT http://localhost:8080/api/v1/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name": "张三(更新)"}'

# 删除用户
curl -X DELETE http://localhost:8080/api/v1/users/123
```

#### 3. 产品服务接口
```bash
# 获取产品列表
curl http://localhost:8080/api/v1/products

# 创建产品
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name": "新产品", "price": 199.99, "category": "电子产品"}'
```

#### 4. 订单服务接口
```bash
# 获取订单列表
curl http://localhost:8080/api/v1/orders

# 创建订单
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "products": [{"id": 1, "quantity": 2}]}'
```

#### 5. 认证服务接口
```bash
# 用户登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "zhangsan", "password": "password123"}'

# 刷新令牌
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 验证令牌
curl http://localhost:8080/api/v1/auth/validate \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 6. 管理后台接口（需要认证）
```bash
# 未认证访问（会返回401）
curl http://localhost:8080/admin/api/v1/stats

# 带认证头访问
curl -H "Authorization: Bearer admin-token" \
  http://localhost:8080/admin/api/v1/stats
```

## 🎨 架构优势

### 1. **模块化设计**
- ✅ 每个服务独立实现，职责清晰
- ✅ 服务间解耦，便于单独开发和测试
- ✅ 支持团队并行开发

### 2. **统一管理**
- ✅ 路由注册集中化，避免散乱
- ✅ 服务发现和注册自动化
- ✅ 统一的中间件和配置

### 3. **扩展性强**
- ✅ 新增服务只需实现接口并注册
- ✅ 支持不同版本的API分组
- ✅ 可以灵活配置不同的中间件策略

### 4. **企业级特性**
- ✅ 完整的trace_id和request_id支持
- ✅ 结构化日志记录
- ✅ 优雅关闭机制
- ✅ 健康检查和监控支持

## 🔧 扩展示例

### 添加新服务
```go
// 1. 定义接口
type NotificationService interface {
    SendEmail(c *gin.Context)
    SendSMS(c *gin.Context)
}

// 2. 实现服务
type notificationServiceImpl struct {
    logger *logger.Logger
}

func (s *notificationServiceImpl) RegisterRoutes(group *gin.RouterGroup) {
    notifications := group.Group("/notifications")
    {
        notifications.POST("/email", s.SendEmail)
        notifications.POST("/sms", s.SendSMS)
    }
}

// 3. 注册到registry
notificationService := NewNotificationService()
registry.Register(notificationService.(RouteRegistrar))
```

### 多版本API支持
```go
server.RegisterRoutes(func(r *gin.Engine) {
    // API v1
    v1 := r.Group("/api/v1")
    {
        registry.RegisterAllRoutes(v1)
    }
    
    // API v2
    v2 := r.Group("/api/v2")
    {
        registryV2.RegisterAllRoutes(v2)
    }
})
```

### 不同中间件策略
```go
server.RegisterRoutes(func(r *gin.Engine) {
    // 公开API（无认证）
    public := r.Group("/api/v1/public")
    {
        publicRegistry.RegisterAllRoutes(public)
    }
    
    // 需要认证的API
    private := r.Group("/api/v1/private")
    private.Use(authMiddleware())
    {
        privateRegistry.RegisterAllRoutes(private)
    }
    
    // 管理员API
    admin := r.Group("/api/v1/admin")
    admin.Use(authMiddleware(), adminMiddleware())
    {
        adminRegistry.RegisterAllRoutes(admin)
    }
})
```

## ✨ 总结

这个架构模式完美实现了你提出的使用场景：

1. **✅ 实现自己的接口**：每个服务都有清晰的业务接口定义
2. **✅ 注册到统一路由组**：通过ServiceRegistry统一管理
3. **✅ 调用server包创建gin客户端**：使用httpserver.NewServer()
4. **✅ 把路由注册传入**：通过RegisterRoutes回调函数注入

这种设计非常适合：
- **微服务内部的模块化架构**
- **大型单体应用的服务分层**
- **团队协作开发**
- **企业级应用的标准化接口管理**

同时保持了Go-Kit一贯的设计哲学：**提供结构化的解决方案，但不限制开发者的灵活性**。 