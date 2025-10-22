package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/logger"
)

func main() {
	fmt.Println("=== HTTP Server 健康检查功能演示 ===")
	fmt.Println()

	// 1. 基本健康检查演示
	fmt.Println("1. 基本健康检查演示:")
	fmt.Println("----------------------------------------")
	
	// 创建基本服务器（默认启用健康检查）
	basicServer := httpserver.NewServer(nil)
	
	// 添加一些业务路由
	basicServer.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "欢迎使用 Go-Kit HTTP Server",
			"version": "1.0.0",
		})
	})
	
	basicServer.GET("/api/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "demo-service",
			"version": "1.0.0",
			"uptime":  time.Since(time.Now()).String(),
		})
	})
	
	// 启动基本服务器
	go func() {
		if err := basicServer.Run(); err != nil {
			log.Printf("基本服务器启动失败: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	fmt.Printf("基本服务器启动在端口: %s\n", basicServer.Addr())
	fmt.Printf("健康检查端点: http://localhost:8080/health\n")
	
	// 2. 带管理器的健康检查演示
	fmt.Println("\n2. 带管理器的健康检查演示:")
	fmt.Println("----------------------------------------")
	
	// 创建健康检查管理器
	manager := httpserver.NewHealthCheckManager("1.0.0")
	
	// 添加模拟数据库检查器
	manager.AddChecker(httpserver.NewCustomHealthChecker("database", func(ctx context.Context) error {
		// 模拟数据库检查
		time.Sleep(100 * time.Millisecond)
		return nil // 模拟检查通过
	}))
	
	// 添加模拟Redis检查器
	manager.AddChecker(httpserver.NewCustomHealthChecker("redis", func(ctx context.Context) error {
		// 模拟Redis检查
		time.Sleep(50 * time.Millisecond)
		return nil // 模拟检查通过
	}))
	
	// 添加HTTP服务检查器（检查基本服务器）
	manager.AddChecker(httpserver.NewHTTPHealthChecker("basic_service", "http://localhost:8080/health", 3*time.Second))
	
	// 添加会失败的检查器（用于演示不健康状态）
	manager.AddChecker(httpserver.NewCustomHealthChecker("external_api", func(ctx context.Context) error {
		// 模拟外部API检查失败
		return fmt.Errorf("外部API服务不可用")
	}))
	
	// 创建带管理器的服务器
	advancedServer := httpserver.NewServer(&httpserver.Config{
		Host:            "0.0.0.0",
		Port:            8081,
		EnableHealthCheck: true,
		HealthCheckPath: "/health",
	})
	
	// 启用带管理器的健康检查
	advancedServer.EnableHealthCheckWithManager(manager)
	
	// 添加业务路由
	advancedServer.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "高级服务器 - 带健康检查管理器",
			"version": "1.0.0",
		})
	})
	
	advancedServer.GET("/api/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "running",
			"checks": []string{"database", "redis", "basic_service", "external_api"},
		})
	})
	
	// 启动高级服务器
	go func() {
		if err := advancedServer.Run(); err != nil {
			log.Printf("高级服务器启动失败: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	fmt.Printf("高级服务器启动在端口: %s\n", advancedServer.Addr())
	fmt.Printf("健康检查端点: http://localhost:8081/health\n")
	
	// 3. 自定义健康检查路径演示
	fmt.Println("\n3. 自定义健康检查路径演示:")
	fmt.Println("----------------------------------------")
	
	customServer := httpserver.NewServer(&httpserver.Config{
		Host:            "0.0.0.0",
		Port:            8082,
		EnableHealthCheck: true,
		HealthCheckPath: "/api/v1/health", // 自定义路径
	})
	
	customServer.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "自定义健康检查路径服务器",
			"health_path": "/api/v1/health",
		})
	})
	
	// 启动自定义服务器
	go func() {
		if err := customServer.Run(); err != nil {
			log.Printf("自定义服务器启动失败: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	fmt.Printf("自定义服务器启动在端口: %s\n", customServer.Addr())
	fmt.Printf("健康检查端点: http://localhost:8082/api/v1/health\n")
	
	// 4. 禁用健康检查演示
	fmt.Println("\n4. 禁用健康检查演示:")
	fmt.Println("----------------------------------------")
	
	disabledServer := httpserver.NewServer(&httpserver.Config{
		Host:            "0.0.0.0",
		Port:            8083,
		EnableHealthCheck: false, // 禁用健康检查
	})
	
	disabledServer.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "禁用健康检查的服务器",
			"health_enabled": false,
		})
	})
	
	// 启动禁用健康检查的服务器
	go func() {
		if err := disabledServer.Run(); err != nil {
			log.Printf("禁用健康检查服务器启动失败: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	fmt.Printf("禁用健康检查服务器启动在端口: %s\n", disabledServer.Addr())
	fmt.Printf("健康检查端点: http://localhost:8083/health (应该返回404)\n")
	
	// 5. 演示日志集成
	fmt.Println("\n5. 日志集成演示:")
	fmt.Println("----------------------------------------")
	
	// 设置logger
	logger.SetupDevelopment()
	
	// 创建带日志的服务器
	loggedServer := httpserver.NewServer(&httpserver.Config{
		Host:            "0.0.0.0",
		Port:            8084,
		EnableHealthCheck: true,
		HealthCheckPath: "/health",
	})
	
	// 添加中间件
	loggedServer.Use(httpserver.TraceIDMiddleware())
	loggedServer.Use(httpserver.RequestIDMiddleware())
	
	// 添加业务路由
	loggedServer.GET("/", func(c *gin.Context) {
		ctx := httpserver.ContextFromGin(c)
		log := logger.FromContext(ctx)
		
		log.Info("处理根路径请求")
		
		c.JSON(200, gin.H{
			"message": "带日志集成的服务器",
			"trace_id": httpserver.GetTraceID(c),
			"request_id": httpserver.GetRequestID(c),
		})
	})
	
	// 启动带日志的服务器
	go func() {
		if err := loggedServer.Run(); err != nil {
			log.Printf("带日志服务器启动失败: %v", err)
		}
	}()
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	fmt.Printf("带日志服务器启动在端口: %s\n", loggedServer.Addr())
	fmt.Printf("健康检查端点: http://localhost:8084/health\n")
	
	// 显示所有端点
	fmt.Println("\n=== 所有服务器端点 ===")
	fmt.Println("基本服务器:")
	fmt.Println("  - GET http://localhost:8080/")
	fmt.Println("  - GET http://localhost:8080/api/info")
	fmt.Println("  - GET http://localhost:8080/health")
	
	fmt.Println("\n高级服务器（带管理器）:")
	fmt.Println("  - GET http://localhost:8081/")
	fmt.Println("  - GET http://localhost:8081/api/status")
	fmt.Println("  - GET http://localhost:8081/health")
	
	fmt.Println("\n自定义路径服务器:")
	fmt.Println("  - GET http://localhost:8082/")
	fmt.Println("  - GET http://localhost:8082/api/v1/health")
	
	fmt.Println("\n禁用健康检查服务器:")
	fmt.Println("  - GET http://localhost:8083/")
	fmt.Println("  - GET http://localhost:8083/health (404)")
	
	fmt.Println("\n带日志服务器:")
	fmt.Println("  - GET http://localhost:8084/")
	fmt.Println("  - GET http://localhost:8084/health")
	
	fmt.Println("\n=== 测试命令 ===")
	fmt.Println("# 测试基本健康检查")
	fmt.Println("curl http://localhost:8080/health")
	
	fmt.Println("\n# 测试带管理器的健康检查")
	fmt.Println("curl http://localhost:8081/health")
	
	fmt.Println("\n# 测试自定义路径健康检查")
	fmt.Println("curl http://localhost:8082/api/v1/health")
	
	fmt.Println("\n# 测试禁用健康检查（应该返回404）")
	fmt.Println("curl http://localhost:8083/health")
	
	fmt.Println("\n# 测试带日志的请求")
	fmt.Println("curl http://localhost:8084/")
	
	fmt.Println("\n服务器运行中，按 Ctrl+C 停止...")
	
	// 保持程序运行
	select {}
}