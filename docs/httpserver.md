# HTTP服务器 (pkg/httpserver)

基于Gin的轻量级HTTP服务器，提供中间件、路由管理、上下文追踪等企业级特性。

## 🚀 特性

- ✅ 基于Gin的高性能HTTP框架
- ✅ 内置中间件系统
- ✅ 上下文追踪和日志集成
- ✅ 优雅关闭和健康检查
- ✅ 路由管理和分组
- ✅ 请求/响应拦截器
- ✅ 线程安全

## 📖 快速开始

### 基本使用

```go
package main

import (
    "log"
    "go-kit/pkg/httpserver"
    "go-kit/pkg/logger"
)

func main() {
    // 创建服务器
    server := httpserver.New(&httpserver.Config{
        Host: "0.0.0.0",
        Port: 8080,
        Mode: "debug",
    })
    
    // 注册路由
    server.GET("/health", healthHandler)
    server.POST("/users", createUserHandler)
    server.GET("/users/:id", getUserHandler)
    
    // 启动服务器
    log := logger.New()
    log.Info("启动HTTP服务器", "port", 8080)
    
    if err := server.Run(); err != nil {
        log.Fatal("服务器启动失败", "error", err)
    }
}

func healthHandler(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}

func createUserHandler(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": "无效的请求数据"})
        return
    }
    
    // 处理用户创建逻辑...
    c.JSON(201, user)
}

func getUserHandler(c *gin.Context) {
    userID := c.Param("id")
    
    // 获取用户逻辑...
    user := &User{ID: userID, Name: "张三"}
    c.JSON(200, user)
}
```

### 带中间件的服务器

```go
func main() {
    server := httpserver.New(&httpserver.Config{
        Host: "0.0.0.0",
        Port: 8080,
        Mode: "release",
    })
    
    // 添加全局中间件
    server.Use(
        httpserver.LoggerMiddleware(),
        httpserver.RecoveryMiddleware(),
        httpserver.CorsMiddleware(),
    )
    
    // 添加路由组
    api := server.Group("/api/v1")
    {
        api.GET("/users", getUsersHandler)
        api.POST("/users", createUserHandler)
        api.GET("/users/:id", getUserHandler)
        api.PUT("/users/:id", updateUserHandler)
        api.DELETE("/users/:id", deleteUserHandler)
    }
    
    // 启动服务器
    server.Run()
}
```

## 🔧 API 参考

### 创建服务器

#### New
使用配置创建HTTP服务器

```go
server := httpserver.New(&httpserver.Config{
    Host: "0.0.0.0",
    Port: 8080,
    Mode: "debug", // debug, release, test
})
```

#### NewWithOptions
使用选项创建服务器

```go
server := httpserver.NewWithOptions(httpserver.Options{
    Host: "0.0.0.0",
    Port: 8080,
    Mode: "release",
    
    // 中间件配置
    Middlewares: []gin.HandlerFunc{
        httpserver.LoggerMiddleware(),
        httpserver.RecoveryMiddleware(),
    },
    
    // 路由配置
    Routes: func(r *gin.Engine) {
        r.GET("/health", healthHandler)
        r.POST("/users", createUserHandler)
    },
})
```

### 配置选项

#### Config 结构体

```go
type Config struct {
    // 基础配置
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
    Mode string `mapstructure:"mode"` // debug, release, test
    
    // 超时配置
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
    
    // 中间件配置
    EnableLogger    bool `mapstructure:"enable_logger"`
    EnableRecovery  bool `mapstructure:"enable_recovery"`
    EnableCors      bool `mapstructure:"enable_cors"`
    EnableMetrics   bool `mapstructure:"enable_metrics"`
    
    // 日志配置
    LogFormat string `mapstructure:"log_format"` // json, console
    LogLevel  string `mapstructure:"log_level"`  // debug, info, warn, error
    
    // 安全配置
    TrustedProxies []string `mapstructure:"trusted_proxies"`
    MaxBodySize     int64   `mapstructure:"max_body_size"`
}
```

### 路由管理

#### 基本路由

```go
// GET请求
server.GET("/users", getUsersHandler)

// POST请求
server.POST("/users", createUserHandler)

// PUT请求
server.PUT("/users/:id", updateUserHandler)

// DELETE请求
server.DELETE("/users/:id", deleteUserHandler)

// PATCH请求
server.PATCH("/users/:id", patchUserHandler)

// 任意方法
server.Any("/webhook", webhookHandler)

// 静态文件
server.Static("/static", "./static")
server.StaticFile("/favicon.ico", "./favicon.ico")
```

#### 路由组

```go
// API v1 路由组
apiV1 := server.Group("/api/v1")
{
    // 用户相关路由
    users := apiV1.Group("/users")
    {
        users.GET("", getUsersHandler)
        users.POST("", createUserHandler)
        users.GET("/:id", getUserHandler)
        users.PUT("/:id", updateUserHandler)
        users.DELETE("/:id", deleteUserHandler)
    }
    
    // 订单相关路由
    orders := apiV1.Group("/orders")
    {
        orders.GET("", getOrdersHandler)
        orders.POST("", createOrderHandler)
        orders.GET("/:id", getOrderHandler)
    }
}

// 管理后台路由组
admin := server.Group("/admin")
admin.Use(authMiddleware, adminMiddleware)
{
    admin.GET("/dashboard", dashboardHandler)
    admin.GET("/users", adminGetUsersHandler)
    admin.GET("/orders", adminGetOrdersHandler)
}
```

### 中间件

#### 内置中间件

```go
// 日志中间件
server.Use(httpserver.LoggerMiddleware())

// 恢复中间件
server.Use(httpserver.RecoveryMiddleware())

// CORS中间件
server.Use(httpserver.CorsMiddleware())

// 请求ID中间件
server.Use(httpserver.RequestIDMiddleware())

// 超时中间件
server.Use(httpserver.TimeoutMiddleware(30 * time.Second))

// 限流中间件
server.Use(httpserver.RateLimitMiddleware(100, time.Minute))

// 指标中间件
server.Use(httpserver.MetricsMiddleware())
```

#### 自定义中间件

```go
// 认证中间件
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "未授权"})
            c.Abort()
            return
        }
        
        // 验证token...
        userID := validateToken(token)
        if userID == "" {
            c.JSON(401, gin.H{"error": "无效的token"})
            c.Abort()
            return
        }
        
        // 设置用户信息到上下文
        c.Set("user_id", userID)
        c.Next()
    }
}

// 使用自定义中间件
server.Use(authMiddleware())
```

### 上下文管理

#### 从Gin Context获取请求上下文

```go
func userHandler(c *gin.Context) {
    // 获取请求上下文
    ctx := httpserver.ContextFromGin(c)
    
    // 创建带上下文的日志记录器
    log := logger.FromContext(ctx)
    
    // 获取请求ID
    requestID := httpserver.GetRequestID(c)
    
    // 获取用户ID（从中间件设置）
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "用户未认证"})
        return
    }
    
    log.Info("处理用户请求", "user_id", userID, "request_id", requestID)
    
    // 处理请求...
    c.JSON(200, gin.H{"message": "success"})
}
```

#### 设置上下文值

```go
func setContextMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 设置请求开始时间
        c.Set("start_time", time.Now())
        
        // 设置客户端IP
        c.Set("client_ip", c.ClientIP())
        
        // 设置用户代理
        c.Set("user_agent", c.GetHeader("User-Agent"))
        
        c.Next()
    }
}
```

### 错误处理

#### 全局错误处理

```go
// 注册全局错误处理器
server.SetErrorHandler(func(c *gin.Context, err error) {
    log := logger.FromContext(httpserver.ContextFromGin(c))
    
    // 记录错误
    log.Error("请求处理失败", "error", err, "path", c.Request.URL.Path)
    
    // 根据错误类型返回不同的响应
    switch {
    case errors.IsInvalidParam(err):
        c.JSON(400, gin.H{"error": "参数错误", "message": err.Error()})
        
    case errors.IsNotFound(err):
        c.JSON(404, gin.H{"error": "资源不存在", "message": err.Error()})
        
    case errors.IsUnauthorized(err):
        c.JSON(401, gin.H{"error": "未授权", "message": err.Error()})
        
    default:
        c.JSON(500, gin.H{"error": "服务器内部错误"})
    }
})
```

#### 中间件错误处理

```go
func errorRecoveryMiddleware() gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        log := logger.FromContext(httpserver.ContextFromGin(c))
        
        log.Error("请求panic", "panic", recovered, "path", c.Request.URL.Path)
        
        c.JSON(500, gin.H{
            "error": "服务器内部错误",
            "request_id": httpserver.GetRequestID(c),
        })
    })
}
```

### 健康检查

```go
// 注册健康检查处理器
func healthHandler(c *gin.Context) {
    // 检查数据库连接
    if err := db.Ping(); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error": "数据库连接失败",
        })
        return
    }
    
    // 检查外部服务
    if err := checkExternalService(); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error": "外部服务不可用",
        })
        return
    }
    
    c.JSON(200, gin.H{
        "status": "healthy",
        "timestamp": time.Now().Unix(),
    })
}

server.GET("/health", healthHandler)
```

## 🏗️ 最佳实践

### 1. 服务器配置

#### 从配置文件加载

```go
type ServerConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
    Mode string `mapstructure:"mode"`
    
    // 超时配置
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
    
    // 中间件配置
    EnableLogger   bool `mapstructure:"enable_logger"`
    EnableRecovery bool `mapstructure:"enable_recovery"`
    EnableCors     bool `mapstructure:"enable_cors"`
}

func loadServerConfig() (*httpserver.Config, error) {
    var cfg struct {
        Server ServerConfig `mapstructure:"server"`
    }
    
    if err := config.LoadConfig(&cfg); err != nil {
        return nil, err
    }
    
    return &httpserver.Config{
        Host:          cfg.Server.Host,
        Port:          cfg.Server.Port,
        Mode:          cfg.Server.Mode,
        ReadTimeout:   cfg.Server.ReadTimeout,
        WriteTimeout:  cfg.Server.WriteTimeout,
        IdleTimeout:   cfg.Server.IdleTimeout,
        EnableLogger:  cfg.Server.EnableLogger,
        EnableRecovery: cfg.Server.EnableRecovery,
        EnableCors:    cfg.Server.EnableCors,
    }, nil
}
```

### 2. 路由组织

#### 模块化路由

```go
// 用户路由模块
func registerUserRoutes(server *httpserver.Server) {
    users := server.Group("/api/v1/users")
    {
        users.GET("", getUsersHandler)
        users.POST("", createUserHandler)
        users.GET("/:id", getUserHandler)
        users.PUT("/:id", updateUserHandler)
        users.DELETE("/:id", deleteUserHandler)
    }
}

// 订单路由模块
func registerOrderRoutes(server *httpserver.Server) {
    orders := server.Group("/api/v1/orders")
    {
        orders.GET("", getOrdersHandler)
        orders.POST("", createOrderHandler)
        orders.GET("/:id", getOrderHandler)
        orders.PUT("/:id", updateOrderHandler)
        orders.DELETE("/:id", deleteOrderHandler)
    }
}

// 主函数
func main() {
    server := httpserver.New(config)
    
    // 注册路由模块
    registerUserRoutes(server)
    registerOrderRoutes(server)
    
    server.Run()
}
```

### 3. 中间件链

#### 中间件顺序

```go
func setupMiddlewares(server *httpserver.Server) {
    // 1. 基础中间件（最先执行）
    server.Use(httpserver.RequestIDMiddleware())
    server.Use(httpserver.LoggerMiddleware())
    server.Use(httpserver.RecoveryMiddleware())
    
    // 2. 安全中间件
    server.Use(httpserver.CorsMiddleware())
    server.Use(httpserver.RateLimitMiddleware(100, time.Minute))
    
    // 3. 业务中间件
    server.Use(authMiddleware())
    server.Use(permissionMiddleware())
    
    // 4. 监控中间件（最后执行）
    server.Use(httpserver.MetricsMiddleware())
}
```

### 4. 处理器设计

#### 结构化处理器

```go
// 用户处理器
type UserHandler struct {
    userService *UserService
    logger      *logger.Logger
}

func NewUserHandler(userService *UserService, logger *logger.Logger) *UserHandler {
    return &UserHandler{
        userService: userService,
        logger:      logger,
    }
}

func (h *UserHandler) GetUsers(c *gin.Context) {
    ctx := httpserver.ContextFromGin(c)
    log := logger.FromContext(ctx)
    
    // 获取查询参数
    page := c.DefaultQuery("page", "1")
    size := c.DefaultQuery("size", "10")
    
    // 调用服务层
    users, total, err := h.userService.GetUsers(ctx, page, size)
    if err != nil {
        log.Error("获取用户列表失败", "error", err)
        c.JSON(500, gin.H{"error": "获取用户列表失败"})
        return
    }
    
    log.Info("获取用户列表成功", "count", len(users), "total", total)
    
    c.JSON(200, gin.H{
        "data": users,
        "total": total,
        "page": page,
        "size": size,
    })
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    ctx := httpserver.ContextFromGin(c)
    log := logger.FromContext(ctx)
    
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        log.Warn("请求数据绑定失败", "error", err)
        c.JSON(400, gin.H{"error": "无效的请求数据"})
        return
    }
    
    // 调用服务层
    createdUser, err := h.userService.CreateUser(ctx, &user)
    if err != nil {
        log.Error("创建用户失败", "error", err)
        c.JSON(500, gin.H{"error": "创建用户失败"})
        return
    }
    
    log.Info("创建用户成功", "user_id", createdUser.ID)
    
    c.JSON(201, createdUser)
}
```

### 5. 错误处理

#### 统一错误响应

```go
// 错误响应结构
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Message string                 `json:"message,omitempty"`
    Code    int                   `json:"code,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// 成功响应结构
type SuccessResponse struct {
    Data    interface{}            `json:"data"`
    Message string                 `json:"message,omitempty"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
}

// 统一响应函数
func sendError(c *gin.Context, statusCode int, err error) {
    response := ErrorResponse{
        Error:   err.Error(),
        Code:    statusCode,
    }
    
    // 如果是自定义错误，获取更多信息
    if customErr, ok := err.(*errors.Error); ok {
        response.Message = customErr.GetMessage()
        response.Details = errors.GetContext(err)
    }
    
    c.JSON(statusCode, response)
}

func sendSuccess(c *gin.Context, data interface{}, message string) {
    response := SuccessResponse{
        Data:    data,
        Message: message,
    }
    
    c.JSON(200, response)
}
```

### 6. 监控和指标

#### 请求指标收集

```go
func metricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // 处理请求
        c.Next()
        
        // 记录指标
        duration := time.Since(start)
        statusCode := c.Writer.Status()
        
        // 记录请求计数
        requestCounter.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            fmt.Sprintf("%d", statusCode),
        ).Inc()
        
        // 记录请求延迟
        requestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration.Seconds())
        
        // 记录响应大小
        responseSize.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(float64(c.Writer.Size()))
    }
}
```

### 7. 优雅关闭

```go
func gracefulShutdown(server *httpserver.Server) {
    // 等待中断信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log := logger.New()
    log.Info("正在关闭服务器...")
    
    // 设置关闭超时
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Error("服务器关闭失败", "error", err)
    }
    
    log.Info("服务器已关闭")
}
```

## 🧪 测试

### 单元测试

```go
func TestUserHandler_GetUsers(t *testing.T) {
    // 创建测试服务器
    server := httpserver.New(&httpserver.Config{
        Host: "localhost",
        Port: 0, // 使用随机端口
        Mode: "test",
    })
    
    // 创建模拟服务
    mockUserService := &MockUserService{}
    handler := NewUserHandler(mockUserService, logger.New())
    
    // 注册路由
    server.GET("/users", handler.GetUsers)
    
    // 启动测试服务器
    go server.Run()
    
    // 等待服务器启动
    time.Sleep(100 * time.Millisecond)
    
    // 发送测试请求
    resp, err := http.Get("http://localhost:8080/users")
    if err != nil {
        t.Fatalf("请求失败: %v", err)
    }
    defer resp.Body.Close()
    
    // 验证响应
    if resp.StatusCode != 200 {
        t.Errorf("期望状态码 200，实际 %d", resp.StatusCode)
    }
    
    var response map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        t.Fatalf("解析响应失败: %v", err)
    }
    
    if response["data"] == nil {
        t.Error("响应中缺少data字段")
    }
}
```

### 集成测试

```go
func TestUserAPI_Integration(t *testing.T) {
    // 创建测试数据库
    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)
    
    // 创建服务
    userService := NewUserService(db)
    handler := NewUserHandler(userService, logger.New())
    
    // 创建测试服务器
    server := httpserver.New(&httpserver.Config{
        Host: "localhost",
        Port: 0,
        Mode: "test",
    })
    
    // 注册路由
    server.POST("/users", handler.CreateUser)
    server.GET("/users/:id", handler.GetUser)
    
    // 启动服务器
    go server.Run()
    time.Sleep(100 * time.Millisecond)
    
    // 测试创建用户
    userData := map[string]interface{}{
        "name":  "测试用户",
        "email": "test@example.com",
    }
    
    userJSON, _ := json.Marshal(userData)
    resp, err := http.Post("http://localhost:8080/users", 
        "application/json", bytes.NewBuffer(userJSON))
    if err != nil {
        t.Fatalf("创建用户请求失败: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 201 {
        t.Errorf("期望状态码 201，实际 %d", resp.StatusCode)
    }
    
    // 验证用户创建成功
    var createdUser User
    if err := json.NewDecoder(resp.Body).Decode(&createdUser); err != nil {
        t.Fatalf("解析响应失败: %v", err)
    }
    
    if createdUser.ID == "" {
        t.Error("用户ID为空")
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 端口被占用

```bash
# 检查端口占用
lsof -i :8080

# 杀死占用进程
kill -9 <PID>

# 或者使用不同的端口
server := httpserver.New(&httpserver.Config{
    Host: "0.0.0.0",
    Port: 8081, // 使用不同端口
})
```

#### 2. 中间件顺序问题

```go
// ❌ 错误的中间件顺序
server.Use(authMiddleware())        // 需要用户信息
server.Use(httpserver.LoggerMiddleware()) // 但日志中间件在认证之前

// ✅ 正确的中间件顺序
server.Use(httpserver.LoggerMiddleware()) // 日志中间件最先
server.Use(authMiddleware())        // 认证中间件在日志之后
```

#### 3. 上下文传递问题

```go
// ❌ 错误的方式
func handler(c *gin.Context) {
    ctx := context.Background() // 丢失请求上下文
    // ...
}

// ✅ 正确的方式
func handler(c *gin.Context) {
    ctx := httpserver.ContextFromGin(c) // 获取请求上下文
    // ...
}
```

### 性能优化

```go
// 1. 启用Gin的发布模式
server := httpserver.New(&httpserver.Config{
    Mode: "release", // 禁用调试信息
})

// 2. 配置连接池
server.SetMaxConnections(1000)

// 3. 启用压缩
server.Use(gin.Recovery())

// 4. 配置静态文件缓存
server.Static("/static", "./static")

// 5. 使用连接池
server.SetConnState(func(conn net.Conn, state http.ConnState) {
    // 监控连接状态
})
```

## 📚 相关链接

- [Gin官方文档](https://gin-gonic.com/)
- [示例项目](./examples/http-server/)
- [返回首页](../README.md) 