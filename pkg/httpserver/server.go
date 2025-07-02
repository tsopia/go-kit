package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-kit/pkg/constants"

	"github.com/gin-gonic/gin"
)

// Config 服务器配置
type Config struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderBytes  int
	ShutdownTimeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Host:            "0.0.0.0",
		Port:            8080,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1MB
		ShutdownTimeout: 10 * time.Second,
	}
}

// Server HTTP服务器 - 最小化封装
type Server struct {
	config *Config
	engine *gin.Engine
	server *http.Server
}

// NewServer 创建新的HTTP服务器
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	// 创建纯净的gin引擎，不添加任何中间件
	engine := gin.New()

	return &Server{
		config: config,
		engine: engine,
	}
}

// Engine 返回Gin引擎，用户完全控制
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// RegisterRoutes 使用回调函数注册路由（推荐方式）
func (s *Server) RegisterRoutes(routes func(r *gin.Engine)) {
	routes(s.engine)
}

// 路由注册便利方法（可选使用）

// GET 注册GET路由的便利方法
func (s *Server) GET(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.GET(relativePath, handlers...)
}

// POST 注册POST路由的便利方法
func (s *Server) POST(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.POST(relativePath, handlers...)
}

// PUT 注册PUT路由的便利方法
func (s *Server) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.PUT(relativePath, handlers...)
}

// DELETE 注册DELETE路由的便利方法
func (s *Server) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.DELETE(relativePath, handlers...)
}

// PATCH 注册PATCH路由的便利方法
func (s *Server) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.PATCH(relativePath, handlers...)
}

// HEAD 注册HEAD路由的便利方法
func (s *Server) HEAD(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.HEAD(relativePath, handlers...)
}

// OPTIONS 注册OPTIONS路由的便利方法
func (s *Server) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.OPTIONS(relativePath, handlers...)
}

// Any 注册所有HTTP方法的便利方法
func (s *Server) Any(relativePath string, handlers ...gin.HandlerFunc) {
	s.engine.Any(relativePath, handlers...)
}

// Group 创建路由组的便利方法
func (s *Server) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.engine.Group(relativePath, handlers...)
}

// Use 添加中间件的便利方法
func (s *Server) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

// Start 启动服务器（非阻塞）
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	// 启动服务器（非阻塞）
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("HTTP server failed to start: %v", err))
		}
	}()

	return nil
}

// Run 启动服务器（阻塞）
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	return s.server.ListenAndServe()
}

// RunTLS 启动HTTPS服务器（阻塞）
func (s *Server) RunTLS(certFile, keyFile string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// RunWithGracefulShutdown 启动服务器并自动处理优雅关闭（阻塞）
func (s *Server) RunWithGracefulShutdown() error {
	// 启动服务器（非阻塞）
	if err := s.Start(); err != nil {
		return err
	}

	// 监听关闭信号
	return s.WaitForShutdown()
}

// WaitForShutdown 等待关闭信号并执行优雅关闭
func (s *Server) WaitForShutdown() error {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	<-quit
	fmt.Println("收到关闭信号，开始优雅关闭服务器...")

	// 创建关闭context
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// 优雅关闭
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭失败: %w", err)
	}

	fmt.Println("服务器已优雅关闭")
	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()
	}

	return s.server.Shutdown(ctx)
}

// Addr 返回服务器地址
func (s *Server) Addr() string {
	if s.server != nil {
		return s.server.Addr
	}
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// IsRunning 检查服务器是否正在运行
func (s *Server) IsRunning() bool {
	return s.server != nil
}

// 中间件函数（可选使用）

// TraceIDMiddleware 添加 Trace ID 的中间件
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查请求头中是否已有 trace id
		traceID := c.GetHeader(constants.TraceIDHeader)
		if traceID == "" {
			// 生成新的 trace id
			traceID = constants.GenerateID()
		}

		// 设置到响应头
		c.Header(constants.TraceIDHeader, traceID)

		// 设置到 gin context 和 request context 中
		c.Set(constants.TraceIDKey, traceID)

		// 为了与 logger 包联动，也要设置到 request context 中
		ctx := constants.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequestIDMiddleware 添加 Request ID 的中间件（每个请求唯一）
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := constants.GenerateID()

		// 设置到响应头
		c.Header(constants.RequestIDHeader, requestID)

		// 设置到 gin context 和 request context 中
		c.Set(constants.RequestIDKey, requestID)

		// 为了与 logger 包联动，也要设置到 request context 中
		ctx := constants.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", fmt.Sprintf("Content-Type, Authorization, %s, %s", constants.TraceIDHeader, constants.RequestIDHeader))
		c.Header("Access-Control-Expose-Headers", fmt.Sprintf("%s, %s", constants.TraceIDHeader, constants.RequestIDHeader))

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// GetTraceID 从 context 中获取 trace id
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get(constants.TraceIDKey); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// GetRequestID 从 context 中获取 request id
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(constants.RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// ContextFromGin 从 Gin Context 提取 request context
// 这个 context 包含了 trace_id 和 request_id，可以用于创建 logger
// 示例用法:
//
//	ctx := httpserver.ContextFromGin(c)
//	logger := logger.FromContext(ctx)
//	logger.Info("处理用户请求") // 自动包含 trace_id 和 request_id
func ContextFromGin(c *gin.Context) context.Context {
	return c.Request.Context()
}
