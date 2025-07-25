package main

import (
	"log"
	"time"

	"github.com/tsopia/go-kit/pkg/httpserver"
	"github.com/tsopia/go-kit/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 定义各模块的接口

// UserService 用户服务接口
type UserService interface {
	ListUsers(c *gin.Context)
	CreateUser(c *gin.Context)
	GetUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	DeleteUser(c *gin.Context)
}

// ProductService 产品服务接口
type ProductService interface {
	ListProducts(c *gin.Context)
	CreateProduct(c *gin.Context)
	GetProduct(c *gin.Context)
	UpdateProduct(c *gin.Context)
	DeleteProduct(c *gin.Context)
}

// OrderService 订单服务接口
type OrderService interface {
	ListOrders(c *gin.Context)
	CreateOrder(c *gin.Context)
	GetOrder(c *gin.Context)
	UpdateOrderStatus(c *gin.Context)
	CancelOrder(c *gin.Context)
}

// AuthService 认证服务接口
type AuthService interface {
	Login(c *gin.Context)
	Logout(c *gin.Context)
	RefreshToken(c *gin.Context)
	ValidateToken(c *gin.Context)
}

// RouteRegistrar 路由注册器接口
type RouteRegistrar interface {
	RegisterRoutes(group *gin.RouterGroup)
}

// 实现各个服务

// userServiceImpl 用户服务实现
type userServiceImpl struct {
	logger *logger.Logger
}

func NewUserService() UserService {
	return &userServiceImpl{
		logger: logger.New(),
	}
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
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)
	log.Info("获取用户列表")

	users := []gin.H{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com", "role": "admin"},
		{"id": 2, "name": "李四", "email": "lisi@example.com", "role": "user"},
		{"id": 3, "name": "王五", "email": "wangwu@example.com", "role": "user"},
	}

	c.JSON(200, gin.H{
		"users":     users,
		"count":     len(users),
		"trace_id":  httpserver.GetTraceID(c),
		"timestamp": time.Now().Unix(),
	})
}

func (s *userServiceImpl) CreateUser(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	c.JSON(201, gin.H{
		"message": "用户创建成功",
		"user": gin.H{
			"id":    999,
			"name":  req.Name,
			"email": req.Email,
			"role":  req.Role,
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *userServiceImpl) GetUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"user": gin.H{
			"id":    userID,
			"name":  "用户" + userID,
			"email": "user" + userID + "@example.com",
			"role":  "user",
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *userServiceImpl) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "用户更新成功",
		"user_id":  userID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *userServiceImpl) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "用户删除成功",
		"user_id":  userID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// productServiceImpl 产品服务实现
type productServiceImpl struct {
	logger *logger.Logger
}

func NewProductService() ProductService {
	return &productServiceImpl{
		logger: logger.New(),
	}
}

func (s *productServiceImpl) RegisterRoutes(group *gin.RouterGroup) {
	products := group.Group("/products")
	{
		products.GET("", s.ListProducts)
		products.POST("", s.CreateProduct)
		products.GET("/:id", s.GetProduct)
		products.PUT("/:id", s.UpdateProduct)
		products.DELETE("/:id", s.DeleteProduct)
	}
}

func (s *productServiceImpl) ListProducts(c *gin.Context) {
	ctx := httpserver.ContextFromGin(c)
	log := logger.FromContext(ctx)
	log.Info("获取产品列表")

	products := []gin.H{
		{"id": 1, "name": "iPhone 15", "price": 999.99, "category": "手机"},
		{"id": 2, "name": "MacBook Pro", "price": 1999.99, "category": "电脑"},
		{"id": 3, "name": "AirPods Pro", "price": 249.99, "category": "耳机"},
	}

	c.JSON(200, gin.H{
		"products":  products,
		"count":     len(products),
		"trace_id":  httpserver.GetTraceID(c),
		"timestamp": time.Now().Unix(),
	})
}

func (s *productServiceImpl) CreateProduct(c *gin.Context) {
	var req struct {
		Name     string  `json:"name" binding:"required"`
		Price    float64 `json:"price" binding:"required,gt=0"`
		Category string  `json:"category"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"message": "产品创建成功",
		"product": gin.H{
			"id":       888,
			"name":     req.Name,
			"price":    req.Price,
			"category": req.Category,
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *productServiceImpl) GetProduct(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{
		"product": gin.H{
			"id":       productID,
			"name":     "产品" + productID,
			"price":    99.99,
			"category": "默认分类",
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *productServiceImpl) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{
		"message":    "产品更新成功",
		"product_id": productID,
		"trace_id":   httpserver.GetTraceID(c),
	})
}

func (s *productServiceImpl) DeleteProduct(c *gin.Context) {
	productID := c.Param("id")
	c.JSON(200, gin.H{
		"message":    "产品删除成功",
		"product_id": productID,
		"trace_id":   httpserver.GetTraceID(c),
	})
}

// orderServiceImpl 订单服务实现
type orderServiceImpl struct {
	logger *logger.Logger
}

func NewOrderService() OrderService {
	return &orderServiceImpl{
		logger: logger.New(),
	}
}

func (s *orderServiceImpl) RegisterRoutes(group *gin.RouterGroup) {
	orders := group.Group("/orders")
	{
		orders.GET("", s.ListOrders)
		orders.POST("", s.CreateOrder)
		orders.GET("/:id", s.GetOrder)
		orders.PATCH("/:id/status", s.UpdateOrderStatus)
		orders.DELETE("/:id", s.CancelOrder)
	}
}

func (s *orderServiceImpl) ListOrders(c *gin.Context) {
	orders := []gin.H{
		{"id": 1, "user_id": 1, "total": 999.99, "status": "completed"},
		{"id": 2, "user_id": 2, "total": 1999.99, "status": "pending"},
	}

	c.JSON(200, gin.H{
		"orders":    orders,
		"count":     len(orders),
		"trace_id":  httpserver.GetTraceID(c),
		"timestamp": time.Now().Unix(),
	})
}

func (s *orderServiceImpl) CreateOrder(c *gin.Context) {
	c.JSON(201, gin.H{
		"message":  "订单创建成功",
		"order_id": 777,
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *orderServiceImpl) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	c.JSON(200, gin.H{
		"order": gin.H{
			"id":      orderID,
			"user_id": 1,
			"total":   999.99,
			"status":  "completed",
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *orderServiceImpl) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "订单状态更新成功",
		"order_id": orderID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *orderServiceImpl) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	c.JSON(200, gin.H{
		"message":  "订单取消成功",
		"order_id": orderID,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// authServiceImpl 认证服务实现
type authServiceImpl struct {
	logger *logger.Logger
}

func NewAuthService() AuthService {
	return &authServiceImpl{
		logger: logger.New(),
	}
}

func (s *authServiceImpl) RegisterRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")
	{
		auth.POST("/login", s.Login)
		auth.POST("/logout", s.Logout)
		auth.POST("/refresh", s.RefreshToken)
		auth.GET("/validate", s.ValidateToken)
	}
}

func (s *authServiceImpl) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数验证失败", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":      "登录成功",
		"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"expires_in":   3600,
		"user": gin.H{
			"id":       1,
			"username": req.Username,
			"role":     "user",
		},
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *authServiceImpl) Logout(c *gin.Context) {
	c.JSON(200, gin.H{
		"message":  "登出成功",
		"trace_id": httpserver.GetTraceID(c),
	})
}

func (s *authServiceImpl) RefreshToken(c *gin.Context) {
	c.JSON(200, gin.H{
		"message":      "令牌刷新成功",
		"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"expires_in":   3600,
		"trace_id":     httpserver.GetTraceID(c),
	})
}

func (s *authServiceImpl) ValidateToken(c *gin.Context) {
	c.JSON(200, gin.H{
		"valid":    true,
		"user_id":  1,
		"trace_id": httpserver.GetTraceID(c),
	})
}

// 服务注册器 - 统一管理所有服务的路由注册
type ServiceRegistry struct {
	services []RouteRegistrar
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make([]RouteRegistrar, 0),
	}
}

func (r *ServiceRegistry) Register(service RouteRegistrar) {
	r.services = append(r.services, service)
}

func (r *ServiceRegistry) RegisterAllRoutes(group *gin.RouterGroup) {
	for _, service := range r.services {
		service.RegisterRoutes(group)
	}
}

func main() {
	log.Println("=== 模块化服务架构演示 ===")
	log.Println("每个服务实现自己的接口，统一注册到路由组")


	// 3. 创建HTTP服务器
	server := httpserver.NewServer(&httpserver.Config{
		Host: "0.0.0.0",
		Port: 8080,
	})

	// 添加全局中间件
	server.Use(gin.Logger())
	server.Use(gin.Recovery())
	server.Use(httpserver.TraceIDMiddleware())
	server.Use(httpserver.RequestIDMiddleware())
	server.Use(httpserver.CORSMiddleware())

	// 4. 使用回调函数注册所有服务的路由
	server.RegisterRoutes(registerRoutes)

	log.Println("✅ 所有服务路由注册完成")
	log.Println("📡 API接口列表:")
	log.Println("   健康检查: GET /health")
	log.Println("   用户服务: /api/v1/users/*")
	log.Println("   产品服务: /api/v1/products/*")
	log.Println("   订单服务: /api/v1/orders/*")
	log.Println("   认证服务: /api/v1/auth/*")
	log.Println("   管理后台: /admin/api/v1/*")
	log.Printf("🚀 服务器启动: http://localhost:8080\n")
	log.Println("💡 使用 Ctrl+C 优雅关闭服务器")

	// 5. 启动服务器并自动处理优雅关闭
	if err := server.RunWithGracefulShutdown(); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

// 管理员认证中间件
func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token != "Bearer admin-token" {
			c.JSON(401, gin.H{
				"error":    "需要管理员权限",
				"trace_id": httpserver.GetTraceID(c),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
func registerRoutes(r *gin.Engine) {

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
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "healthy",
			"timestamp":  time.Now().Unix(),
			"services":   []string{"user", "product", "order", "auth"},
			"trace_id":   httpserver.GetTraceID(c),
			"request_id": httpserver.GetRequestID(c),
		})
	})

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 统一注册所有服务的路由
		registry.RegisterAllRoutes(v1)
	}

	// 管理后台路由组（可以添加认证中间件）
	admin := r.Group("/admin/api/v1")
	admin.Use(adminAuthMiddleware())
	{
		// 管理员专用接口
		admin.GET("/stats", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"stats": gin.H{
					"total_users":     100,
					"total_products":  50,
					"total_orders":    200,
					"active_sessions": 25,
				},
				"trace_id": httpserver.GetTraceID(c),
			})
		})
	}
}