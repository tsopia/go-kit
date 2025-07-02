package main

import (
	"fmt"
	"log"
	"time"

	"go-kit/pkg/httpserver"
	"go-kit/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("=== 回调函数式路由注册演示 ===")

	server := httpserver.NewServer(&httpserver.Config{
		Host: "0.0.0.0",
		Port: 9080,
	})

	// 添加全局中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	server.Use(httpserver.TraceIDMiddleware())
	server.Use(httpserver.RequestIDMiddleware())
	server.Use(httpserver.CORSMiddleware())

	// 使用回调函数注册路由（推荐方式）
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

			// 订单相关路由
			orders := v1.Group("/orders")
			{
				orders.GET("", listOrdersHandler)
				orders.POST("", createOrderHandler)
				orders.GET("/:id", getOrderHandler)
				orders.PATCH("/:id/status", updateOrderStatusHandler)
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

		// WebSocket 路由
		r.GET("/ws", websocketHandler)

		// 健康检查路由（不同级别）
		health := r.Group("/health")
		{
			health.GET("/live", livenessHandler)
			health.GET("/ready", readinessHandler)
			health.GET("/metrics", metricsHandler)
		}
	})

	fmt.Println("使用回调函数式路由注册完成")
	fmt.Println("注册的路由包括:")
	fmt.Println("- GET /health - 基础健康检查")
	fmt.Println("- GET /ping - 简单ping")
	fmt.Println("- GET /users - 用户列表")
	fmt.Println("- POST /users - 创建用户")
	fmt.Println("- GET /api/v1/products - 产品列表")
	fmt.Println("- GET /api/v1/admin/stats - 管理员统计（需认证）")
	fmt.Println("- GET /ws - WebSocket连接")
	fmt.Println("- GET /health/live - 存活检查")

	// 启动服务器
	fmt.Printf("服务器启动在: http://localhost:9080\n")
	fmt.Println("使用 Ctrl+C 优雅关闭服务器")

	if err := server.RunWithGracefulShutdown(); err != nil {
		log.Fatal("服务器运行失败:", err)
	}
}

// 路由处理器实现
func healthCheckHandler(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)
	log.Info("健康检查请求")

	c.JSON(200, gin.H{
		"status":     "healthy",
		"timestamp":  time.Now().Unix(),
		"trace_id":   httpserver.GetTraceID(c),
		"request_id": httpserver.GetRequestID(c),
	})
}

func pingHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "pong"})
}

func listUsersHandler(c *gin.Context) {
	users := []gin.H{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com"},
		{"id": 2, "name": "李四", "email": "lisi@example.com"},
	}
	c.JSON(200, gin.H{"users": users, "trace_id": httpserver.GetTraceID(c)})
}

func createUserHandler(c *gin.Context) {
	var user struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"message":  "用户创建成功",
		"user":     user,
		"trace_id": httpserver.GetTraceID(c),
	})
}

func getUserHandler(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"user": gin.H{
			"id":    userID,
			"name":  fmt.Sprintf("用户%s", userID),
			"email": fmt.Sprintf("user%s@example.com", userID),
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func updateUserHandler(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "用户更新成功",
		"user_id":  userID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

func deleteUserHandler(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "用户删除成功",
		"user_id":  userID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// 产品相关处理器
func listProductsHandler(c *gin.Context) {
	products := []gin.H{
		{"id": 1, "name": "iPhone 15", "price": 999},
		{"id": 2, "name": "MacBook Pro", "price": 1999},
	}
	c.JSON(200, gin.H{"products": products})
}

func createProductHandler(c *gin.Context) {
	c.JSON(201, gin.H{"message": "产品创建成功"})
}

func getProductHandler(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{
		"product": gin.H{
			"id":    productID,
			"name":  fmt.Sprintf("产品%s", productID),
			"price": 99.99,
		},
	})
}

func updateProductHandler(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{"message": "产品更新成功", "product_id": productID})
}

func deleteProductHandler(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{"message": "产品删除成功", "product_id": productID})
}

// 订单相关处理器
func listOrdersHandler(c *gin.Context) {
	c.JSON(200, gin.H{"orders": []gin.H{}})
}

func createOrderHandler(c *gin.Context) {
	c.JSON(201, gin.H{"message": "订单创建成功"})
}

func getOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	c.JSON(200, gin.H{"order": gin.H{"id": orderID, "status": "pending"}})
}

func updateOrderStatusHandler(c *gin.Context) {
	orderID := c.Param("id")
	c.JSON(200, gin.H{"message": "订单状态更新成功", "order_id": orderID})
}

// 管理员相关处理器
func getStatsHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"stats": gin.H{
			"users":    100,
			"products": 50,
			"orders":   200,
		},
	})
}

func adminListUsersHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "管理员查看用户列表"})
}

func adminDeleteUserHandler(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{"message": "管理员删除用户成功", "user_id": userID})
}

// WebSocket处理器
func websocketHandler(c *gin.Context) {
	c.JSON(200, gin.H{"message": "WebSocket endpoint"})
}

// 健康检查处理器
func livenessHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "alive"})
}

func readinessHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ready"})
}

func metricsHandler(c *gin.Context) {
	c.JSON(200, gin.H{"metrics": gin.H{"requests": 1000, "errors": 5}})
}

// 管理员认证中间件
func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里应该实现真正的认证逻辑
		token := c.GetHeader("Authorization")
		if token != "Bearer admin-token" {
			c.JSON(401, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}
