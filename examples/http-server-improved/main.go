package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tsopia/go-kit/pkg/httpserver"
	"github.com/tsopia/go-kit/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("=== HTTP 服务器改进功能演示 ===")

	// 创建日志记录器
	loggerInstance := logger.New()

	// 演示1：使用便利的路由注册方法
	fmt.Println("1. 演示便利的路由注册方法")
	server := httpserver.NewServer(nil)

	// 添加一些必要的中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	server.Use(httpserver.TraceIDMiddleware())
	server.Use(httpserver.RequestIDMiddleware())
	server.Use(httpserver.CORSMiddleware())

	// 使用便利方法注册路由
	server.GET("/health", healthHandler)
	server.POST("/users", createUserHandler)
	server.PUT("/users/:id", updateUserHandler)
	server.DELETE("/users/:id", deleteUserHandler)
	server.PATCH("/users/:id/status", updateUserStatusHandler)

	// 创建路由组
	api := server.Group("/api/v1")
	{
		api.GET("/users", listUsersHandler)
		api.GET("/users/:id", getUserHandler)
	}

	// 演示2：自动优雅关闭（推荐用法）
	fmt.Println("2. 演示自动优雅关闭功能")
	fmt.Println("服务器启动中...")
	fmt.Printf("- 访问地址: http://localhost:8080\n")
	fmt.Println("- 使用 Ctrl+C 或发送 SIGTERM 信号来优雅关闭服务器")
	fmt.Println("\n可用端点:")
	fmt.Println("- GET /health - 健康检查")
	fmt.Println("- POST /users - 创建用户")
	fmt.Println("- PUT /users/123 - 更新用户")
	fmt.Println("- DELETE /users/123 - 删除用户")
	fmt.Println("- PATCH /users/123/status - 更新用户状态")
	fmt.Println("- GET /api/v1/users - 用户列表")
	fmt.Println("- GET /api/v1/users/123 - 获取用户详情")

	// 运行服务器并自动处理优雅关闭
	if err := server.RunWithGracefulShutdown(); err != nil {
		loggerInstance.Error("服务器运行出错", "error", err)
	}

	fmt.Println("程序退出")
}

// 手动优雅关闭的演示方法
func demonstrateManualShutdown() {
	fmt.Println("3. 演示手动控制优雅关闭")

	server := httpserver.NewServer(&httpserver.Config{Port: 9000})
	server.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// 启动服务器（非阻塞）
	if err := server.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	fmt.Println("服务器已启动在 :9000")
	fmt.Println("模拟运行3秒后关闭...")

	// 模拟运行一段时间
	time.Sleep(3 * time.Second)

	// 手动优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %v", err)
	} else {
		fmt.Println("服务器已优雅关闭")
	}
}

// 对比新旧API的演示
func demonstrateAPIComparison() {
	fmt.Println("4. API使用方式对比")

	server := httpserver.NewServer(nil)

	fmt.Println("原始方式（仍然支持）：")
	fmt.Println("  engine := server.Engine()")
	fmt.Println("  engine.GET(\"/old-way\", handler)")

	// 原始方式仍然完全支持
	engine := server.Engine()
	engine.GET("/old-way", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "old way"})
	})

	fmt.Println("新的便利方式：")
	fmt.Println("  server.GET(\"/new-way\", handler)")

	// 新的便利方式
	server.GET("/new-way", func(c *gin.Context) {
		c.JSON(200, gin.H{"method": "new way"})
	})
}

// healthHandler 健康检查处理器
func healthHandler(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	log.Info("处理健康检查请求")

	traceID := httpserver.GetTraceID(c)
	requestID := httpserver.GetRequestID(c)

	c.JSON(200, gin.H{
		"status":     "ok",
		"timestamp":  time.Now().Unix(),
		"trace_id":   traceID,
		"request_id": requestID,
		"message":    "服务运行正常",
	})
}

// createUserHandler 创建用户处理器
func createUserHandler(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	var user struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}

	log.Info("开始创建用户")

	if err := c.ShouldBindJSON(&user); err != nil {
		log.Error("用户参数验证失败", "error", err.Error())
		c.JSON(400, gin.H{
			"error":      "请求参数无效",
			"details":    err.Error(),
			"trace_id":   httpserver.GetTraceID(c),
			"request_id": httpserver.GetRequestID(c),
		})
		return
	}

	// 模拟创建用户
	newUser := gin.H{
		"id":    123,
		"name":  user.Name,
		"email": user.Email,
	}

	log.Info("用户创建成功", "user_id", 123, "user_name", user.Name)

	c.JSON(201, gin.H{
		"message":    "用户创建成功",
		"user":       newUser,
		"trace_id":   httpserver.GetTraceID(c),
		"request_id": httpserver.GetRequestID(c),
	})
}

// updateUserHandler 更新用户处理器
func updateUserHandler(c *gin.Context) {
	userID := c.Param("id")
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	log.Info("更新用户信息", "user_id", userID)

	var updates struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&updates); err != nil {
		log.Error("更新参数验证失败", "error", err.Error())
		c.JSON(400, gin.H{"error": "请求参数无效", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":  "用户更新成功",
		"user_id":  userID,
		"updates":  updates,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// deleteUserHandler 删除用户处理器
func deleteUserHandler(c *gin.Context) {
	userID := c.Param("id")
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	log.Info("删除用户", "user_id", userID)

	c.JSON(200, gin.H{
		"message":  "用户删除成功",
		"user_id":  userID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// updateUserStatusHandler 更新用户状态处理器
func updateUserStatusHandler(c *gin.Context) {
	userID := c.Param("id")
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	var status struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&status); err != nil {
		log.Error("状态参数验证失败", "error", err.Error())
		c.JSON(400, gin.H{"error": "请求参数无效", "details": err.Error()})
		return
	}

	log.Info("更新用户状态", "user_id", userID, "status", status.Status)

	c.JSON(200, gin.H{
		"message":  "用户状态更新成功",
		"user_id":  userID,
		"status":   status.Status,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// listUsersHandler 用户列表处理器
func listUsersHandler(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	log.Info("获取用户列表")

	users := []gin.H{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com", "status": "active"},
		{"id": 2, "name": "李四", "email": "lisi@example.com", "status": "active"},
		{"id": 3, "name": "王五", "email": "wangwu@example.com", "status": "inactive"},
	}

	c.JSON(200, gin.H{
		"users":    users,
		"count":    len(users),
		"trace_id": httpserver.GetTraceID(c),
	})
}

// getUserHandler 获取单个用户处理器
func getUserHandler(c *gin.Context) {
	userID := c.Param("id")
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)

	log.Info("获取用户详情", "user_id", userID)

	// 模拟查找用户
	user := gin.H{
		"id":     userID,
		"name":   fmt.Sprintf("用户%s", userID),
		"email":  fmt.Sprintf("user%s@example.com", userID),
		"status": "active",
	}

	c.JSON(200, gin.H{
		"user":     user,
		"trace_id": httpserver.GetTraceID(c),
	})
}
