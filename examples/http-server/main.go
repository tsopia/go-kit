package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-kit/pkg/httpserver"
	"go-kit/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("=== 最小化封装的 HTTP 服务器示例 ===")

	// 创建日志记录器
	loggerInstance := logger.New()

	// 示例1：使用默认配置
	fmt.Println("1. 使用默认配置创建服务器")
	server := httpserver.NewServer(nil) // 使用默认配置

	// 获取 Gin 引擎，用户完全控制
	engine := server.Engine()

	// 用户自己决定要添加哪些中间件
	engine.Use(gin.Logger())                     // 可选：添加日志中间件
	engine.Use(gin.Recovery())                   // 可选：添加恢复中间件
	engine.Use(httpserver.TraceIDMiddleware())   // 可选：添加 Trace ID
	engine.Use(httpserver.RequestIDMiddleware()) // 可选：添加 Request ID
	engine.Use(httpserver.CORSMiddleware())      // 可选：添加 CORS

	// 注册路由
	engine.GET("/health", healthHandler)
	engine.GET("/trace", traceHandler)
	engine.GET("/user/:id", userHandler)

	// API 路由组
	api := engine.Group("/api/v1")
	{
		api.GET("/users", listUsersHandler)
		api.POST("/users", createUserHandler)
		api.GET("/users/:id", getUserHandler)
	}

	// 示例2：自定义配置的服务器
	fmt.Println("2. 使用自定义配置创建服务器")
	config := &httpserver.Config{
		Host:            "127.0.0.1",
		Port:            9000,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}

	customServer := httpserver.NewServer(config)
	customEngine := customServer.Engine()

	// 用户完全控制 - 这个服务器只添加 Trace ID，不添加其他中间件
	gin.SetMode(gin.DebugMode) // 用户自己控制 gin 模式
	customEngine.Use(httpserver.TraceIDMiddleware())
	customEngine.GET("/custom", customHandler)

	// 示例3：极简服务器（无任何中间件）
	fmt.Println("3. 极简服务器（无中间件）")
	minimalServer := httpserver.NewServer(&httpserver.Config{Port: 9001})
	minimalEngine := minimalServer.Engine()

	// 不添加任何中间件，完全纯净
	minimalEngine.GET("/minimal", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "极简响应，无任何中间件"})
	})

	// 启动服务器（非阻塞）
	fmt.Println("启动服务器...")
	if err := server.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	if err := customServer.Start(); err != nil {
		log.Fatalf("启动自定义服务器失败: %v", err)
	}

	if err := minimalServer.Start(); err != nil {
		log.Fatalf("启动极简服务器失败: %v", err)
	}

	fmt.Println("服务器已启动:")
	fmt.Printf("- 默认服务器: http://localhost:8080 (完整中间件)\n")
	fmt.Printf("- 自定义服务器: http://localhost:9000 (只有 Trace ID)\n")
	fmt.Printf("- 极简服务器: http://localhost:9001 (无中间件)\n")
	fmt.Println("\n测试端点:")
	fmt.Println("- GET /health - 健康检查")
	fmt.Println("- GET /trace - Trace ID 演示")
	fmt.Println("- GET /user/123 - 用户信息")
	fmt.Println("- GET /api/v1/users - 用户列表")
	fmt.Println("- POST /api/v1/users - 创建用户")
	fmt.Println("- GET /custom - 自定义服务器")
	fmt.Println("- GET /minimal - 极简服务器")

	// 模拟运行
	time.Sleep(2 * time.Second)

	// 优雅关闭
	fmt.Println("\n关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		loggerInstance.Error("服务器关闭失败", "error", err)
	}

	if err := customServer.Shutdown(ctx); err != nil {
		loggerInstance.Error("自定义服务器关闭失败", "error", err)
	}

	if err := minimalServer.Shutdown(ctx); err != nil {
		loggerInstance.Error("极简服务器关闭失败", "error", err)
	}

	fmt.Println("服务器已关闭")
}

// healthHandler 健康检查处理器
func healthHandler(c *gin.Context) {
	// 使用新的便利函数创建带有 trace_id 和 request_id 的 logger
	ctx := httpserver.ContextFromGin(c)
	logger := logger.FromContext(ctx)

	// 日志会自动包含 trace_id 和 request_id
	logger.Info("处理健康检查请求")

	traceID := httpserver.GetTraceID(c)
	requestID := httpserver.GetRequestID(c)

	logger.Info("健康检查完成", "status", "ok")

	c.JSON(200, gin.H{
		"status":     "ok",
		"timestamp":  time.Now().Unix(),
		"trace_id":   traceID,
		"request_id": requestID,
	})
}

// traceHandler 演示 Trace ID 的处理器
func traceHandler(c *gin.Context) {
	// 创建带有追踪信息的 logger
	ctx := httpserver.ContextFromGin(c)
	logger := logger.FromContext(ctx)

	traceID := httpserver.GetTraceID(c)
	requestID := httpserver.GetRequestID(c)

	// 使用 logger 记录，会自动包含 trace_id 和 request_id
	logger.Info("开始处理 Trace ID 演示请求")
	logger.Debug("请求详情", "method", c.Request.Method, "path", c.Request.URL.Path)

	c.JSON(200, gin.H{
		"message":    "Trace ID 演示",
		"trace_id":   traceID,
		"request_id": requestID,
		"tip":        "查看日志输出，每条日志都包含了 trace_id 和 request_id",
	})

	logger.Info("Trace ID 演示请求处理完成")
}

// userHandler 用户处理器
func userHandler(c *gin.Context) {
	userID := c.Param("id")
	traceID := httpserver.GetTraceID(c)

	log.Printf("获取用户信息 - UserID: %s, TraceID: %s", userID, traceID)

	c.JSON(200, gin.H{
		"user_id":  userID,
		"name":     fmt.Sprintf("用户%s", userID),
		"trace_id": traceID,
	})
}

// listUsersHandler 用户列表处理器
func listUsersHandler(c *gin.Context) {
	traceID := httpserver.GetTraceID(c)

	users := []gin.H{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com"},
		{"id": 2, "name": "李四", "email": "lisi@example.com"},
		{"id": 3, "name": "王五", "email": "wangwu@example.com"},
	}

	log.Printf("获取用户列表 - TraceID: %s, Count: %d", traceID, len(users))

	c.JSON(200, gin.H{
		"users":    users,
		"count":    len(users),
		"trace_id": traceID,
	})
}

// createUserHandler 创建用户处理器
func createUserHandler(c *gin.Context) {
	// 创建带有追踪信息的 logger
	ctx := httpserver.ContextFromGin(c)
	logger := logger.FromContext(ctx)
	
	traceID := httpserver.GetTraceID(c)
	requestID := httpserver.GetRequestID(c)

	var user struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	logger.Info("开始创建用户", "remote_addr", c.ClientIP())

	if err := c.ShouldBindJSON(&user); err != nil {
		// 错误日志也会自动包含 trace_id 和 request_id
		logger.Error("用户参数验证失败", "error", err.Error())
		c.JSON(400, gin.H{
			"error":      "请求参数无效",
			"details":    err.Error(),
			"trace_id":   traceID,
			"request_id": requestID,
		})
		return
	}

	// 模拟保存用户
	newUser := gin.H{
		"id":    999,
		"name":  user.Name,
		"email": user.Email,
	}

	logger.Info("用户创建成功", "user_id", 999, "user_name", user.Name, "user_email", user.Email)

	c.JSON(201, gin.H{
		"message":    "用户创建成功",
		"user":       newUser,
		"trace_id":   traceID,
		"request_id": requestID,
	})
}

// getUserHandler 获取单个用户处理器
func getUserHandler(c *gin.Context) {
	userID := c.Param("id")
	traceID := httpserver.GetTraceID(c)

	// 模拟查找用户
	user := gin.H{
		"id":    userID,
		"name":  fmt.Sprintf("用户%s", userID),
		"email": fmt.Sprintf("user%s@example.com", userID),
	}

	log.Printf("获取用户详情 - UserID: %s, TraceID: %s", userID, traceID)

	c.JSON(200, gin.H{
		"user":     user,
		"trace_id": traceID,
	})
}

// customHandler 自定义处理器
func customHandler(c *gin.Context) {
	traceID := httpserver.GetTraceID(c)

	c.JSON(200, gin.H{
		"message":  "这是自定义服务器的响应",
		"server":   "custom",
		"trace_id": traceID,
	})
}
