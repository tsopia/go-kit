package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tsopia/go-kit/constants"

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
	
	// 健康检查配置
	EnableHealthCheck bool   // 是否启用健康检查
	HealthCheckPath   string // 健康检查路径，默认为 /health
	HealthCheckPort   int    // 健康检查端口，默认为0（使用主端口）
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
		
		// 健康检查默认配置
		EnableHealthCheck: true,
		HealthCheckPath:   "/health",
		HealthCheckPort:   0, // 使用主端口
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

	server := &Server{
		config: config,
		engine: engine,
	}

	// 如果启用了健康检查，自动注册健康检查路由
	if config.EnableHealthCheck {
		server.EnableHealthCheck()
	}

	return server
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

// 健康检查相关结构体和函数

// HealthStatus 健康检查状态
type HealthStatus struct {
	Status    string                 `json:"status"`              // healthy, unhealthy
	Timestamp int64                  `json:"timestamp"`           // Unix时间戳
	Version   string                 `json:"version,omitempty"`   // 应用版本
	Uptime    int64                  `json:"uptime,omitempty"`    // 运行时间（秒）
	Checks    map[string]interface{} `json:"checks,omitempty"`    // 详细检查结果
	Error     string                 `json:"error,omitempty"`     // 错误信息
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	GetName() string
}

// HealthCheckManager 健康检查管理器
type HealthCheckManager struct {
	checkers []HealthChecker
	startTime time.Time
	version   string
}

// NewHealthCheckManager 创建健康检查管理器
func NewHealthCheckManager(version string) *HealthCheckManager {
	return &HealthCheckManager{
		checkers:  make([]HealthChecker, 0),
		startTime: time.Now(),
		version:   version,
	}
}

// AddChecker 添加健康检查器
func (hcm *HealthCheckManager) AddChecker(checker HealthChecker) {
	hcm.checkers = append(hcm.checkers, checker)
}

// CheckHealth 执行所有健康检查
func (hcm *HealthCheckManager) CheckHealth(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Version:   hcm.version,
		Uptime:    int64(time.Since(hcm.startTime).Seconds()),
		Checks:    make(map[string]interface{}),
	}

	// 执行所有检查器
	for _, checker := range hcm.checkers {
		checkName := checker.GetName()
		if err := checker.CheckHealth(ctx); err != nil {
			status.Status = "unhealthy"
			status.Checks[checkName] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			status.Checks[checkName] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	return status
}

// DefaultHealthHandler 默认健康检查处理器
func DefaultHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := &HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now().Unix(),
			Version:   "1.0.0",
		}

		c.JSON(http.StatusOK, status)
	}
}

// HealthHandlerWithManager 带管理器的健康检查处理器
func HealthHandlerWithManager(manager *HealthCheckManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := ContextFromGin(c)
		status := manager.CheckHealth(ctx)

		// 根据健康状态返回相应的HTTP状态码
		if status.Status == "healthy" {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	}
}

// EnableHealthCheck 启用健康检查
func (s *Server) EnableHealthCheck() {
	if s.config.EnableHealthCheck {
		// 使用默认健康检查处理器
		s.engine.GET(s.config.HealthCheckPath, DefaultHealthHandler())
	}
}

// EnableHealthCheckWithManager 启用带管理器的健康检查
func (s *Server) EnableHealthCheckWithManager(manager *HealthCheckManager) {
	// 无论是否启用默认健康检查，都注册带管理器的健康检查
	s.engine.GET(s.config.HealthCheckPath, HealthHandlerWithManager(manager))
}

// SetHealthCheckPath 设置健康检查路径
func (s *Server) SetHealthCheckPath(path string) {
	s.config.HealthCheckPath = path
}

// GetHealthCheckPath 获取健康检查路径
func (s *Server) GetHealthCheckPath() string {
	return s.config.HealthCheckPath
}

// 内置健康检查器

// DatabaseHealthChecker 数据库健康检查器
type DatabaseHealthChecker struct {
	name string
	db   interface {
		Ping() error
	}
}

// NewDatabaseHealthChecker 创建数据库健康检查器
func NewDatabaseHealthChecker(name string, db interface {
	Ping() error
}) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name: name,
		db:   db,
	}
}

// CheckHealth 检查数据库健康状态
func (dhc *DatabaseHealthChecker) CheckHealth(ctx context.Context) error {
	return dhc.db.Ping()
}

// GetName 获取检查器名称
func (dhc *DatabaseHealthChecker) GetName() string {
	return dhc.name
}

// HTTPHealthChecker HTTP服务健康检查器
type HTTPHealthChecker struct {
	name    string
	url     string
	timeout time.Duration
}

// NewHTTPHealthChecker 创建HTTP服务健康检查器
func NewHTTPHealthChecker(name, url string, timeout time.Duration) *HTTPHealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &HTTPHealthChecker{
		name:    name,
		url:     url,
		timeout: timeout,
	}
}

// CheckHealth 检查HTTP服务健康状态
func (hhc *HTTPHealthChecker) CheckHealth(ctx context.Context) error {
	client := &http.Client{
		Timeout: hhc.timeout,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		return err
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	
	return fmt.Errorf("HTTP服务返回状态码: %d", resp.StatusCode)
}

// GetName 获取检查器名称
func (hhc *HTTPHealthChecker) GetName() string {
	return hhc.name
}

// CustomHealthChecker 自定义健康检查器
type CustomHealthChecker struct {
	name    string
	checker func(ctx context.Context) error
}

// NewCustomHealthChecker 创建自定义健康检查器
func NewCustomHealthChecker(name string, checker func(ctx context.Context) error) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:    name,
		checker: checker,
	}
}

// CheckHealth 执行自定义健康检查
func (chc *CustomHealthChecker) CheckHealth(ctx context.Context) error {
	return chc.checker(ctx)
}

// GetName 获取检查器名称
func (chc *CustomHealthChecker) GetName() string {
	return chc.name
}
