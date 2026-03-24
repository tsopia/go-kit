package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server HTTP服务器 - 最小化封装
type Server struct {
	config                   *Config
	engine                   *gin.Engine
	server                   *http.Server
	healthServer             *http.Server
	serveErrCh               chan error
	hooks                    Hooks
	healthHandler            gin.HandlerFunc
	healthRouteRegistered    bool
	readinessRouteRegistered bool
	livenessRouteRegistered  bool
	healthAddr               string
	manualReady              bool
	stateMu                  sync.RWMutex
	state                    State
	serverMutators           []HTTPServerMutator
	// 路由分组
	regularGroup   *gin.RouterGroup // 有 Timeout 中间件
	streamingGroup *gin.RouterGroup // 无 Timeout 中间件，用于 SSE/WS
}

// NewServer 创建新的HTTP服务器
func NewServer(config *Config, opts ...Option) *Server {
	config = normalizeConfig(config)

	// 创建纯净的gin引擎，不添加任何中间件
	engine := gin.New()

	server := &Server{
		config:        config,
		engine:        engine,
		serveErrCh:    make(chan error, 4),
		healthHandler: DefaultHealthHandler(),
		state:         StateNew,
	}

	// 如果启用了健康检查，自动注册健康检查路由
	if config.EnableHealthCheck {
		server.EnableHealthCheck()
		server.registerProbeRoutes()
	}

	server.applyOptions(opts...)

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
	s.getRegularGroup().GET(relativePath, handlers...)
}

// POST 注册POST路由的便利方法
func (s *Server) POST(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().POST(relativePath, handlers...)
}

// PUT 注册PUT路由的便利方法
func (s *Server) PUT(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().PUT(relativePath, handlers...)
}

// DELETE 注册DELETE路由的便利方法
func (s *Server) DELETE(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().DELETE(relativePath, handlers...)
}

// PATCH 注册PATCH路由的便利方法
func (s *Server) PATCH(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().PATCH(relativePath, handlers...)
}

// HEAD 注册HEAD路由的便利方法
func (s *Server) HEAD(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().HEAD(relativePath, handlers...)
}

// OPTIONS 注册OPTIONS路由的便利方法
func (s *Server) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().OPTIONS(relativePath, handlers...)
}

// Any 注册所有HTTP方法的便利方法
func (s *Server) Any(relativePath string, handlers ...gin.HandlerFunc) {
	s.getRegularGroup().Any(relativePath, handlers...)
}

// Group 创建路由组的便利方法
func (s *Server) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.getRegularGroup().Group(relativePath, handlers...)
}

// Use 添加中间件的便利方法
func (s *Server) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

// SSE 注册一个 Server-Sent Events 路由。
// 自动设置 SSE 响应头，清除 WriteDeadline，使用 streamingGroup（无 Timeout 中间件）。
func (s *Server) SSE(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
	// 应用选项
	cfg := &sseConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	s.getStreamingGroup().GET(relativePath, func(c *gin.Context) {
		// 清除 WriteDeadline
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetWriteDeadline(time.Time{})

		// 设置 SSE 响应头
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)

		// 立即 flush header
		c.Writer.Flush()

		// 为 handler 构建独立 ctx，便于未来扩展内部取消
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		// 创建 stream
		stream := &sseSender{ginCtx: c, ctx: ctx, config: cfg}

		// 启动心跳（如果配置了）
		stopHeartbeat := stream.startHeartbeat(ctx)

		// 调用 handler
		handler(stream)

		// handler 返回后主动停止 heartbeat，允许有限流自然结束
		stopHeartbeat()

		// 连接断开时打印日志
		stream.logDisconnect(c.Request.Context())
	})
}

// WS 注册一个 WebSocket 路由。
// 自动处理 Upgrade、ping/pong、缓冲策略和断开日志。
func (s *Server) WS(relativePath string, handler WSHandlerFunc, opts ...WSRouteOption) {
	cfg := defaultWSRouteConfig()
	for _, opt := range opts {
		opt.applyRoute(&cfg)
	}

	s.getStreamingGroup().GET(relativePath, func(c *gin.Context) {
		// 1. Upgrade 连接
		conn, err := WSUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Debug("websocket upgrade failed",
				"path", c.Request.URL.Path,
				"error", err,
				"remote_addr", c.Request.RemoteAddr,
			)
			return
		}
		defer conn.Close()

		// 2. 创建带超时的 context
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		// 3. 启动读超时监控（如果配置了）
		if cfg.ReadTimeout > 0 {
			go func() {
				timer := time.NewTimer(cfg.ReadTimeout)
				defer timer.Stop()
				select {
				case <-timer.C:
					cancel() // 读超时触发关闭
				case <-ctx.Done():
					// 正常关闭
				}
			}()
		}

		// 4. 配置 pong handler，跟踪最后收到 pong 的时间
		var (
			lastPong   = time.Now()
			lastPongMu sync.Mutex
		)
		conn.SetPongHandler(func(string) error {
			lastPongMu.Lock()
			lastPong = time.Now()
			lastPongMu.Unlock()
			return nil
		})

		// 5. 创建带策略的发送器
		sender := newWSSender(cfg.SendBufferSize, cfg.SendPolicy)

		// 6. 启动发送器 goroutine（应用 SendPolicy）
		go sender.Run(cancel)

		// 7. 创建接收 channel
		recv := make(chan WSMessage, cfg.RecvBufferSize)

		// 8. 启动读 goroutine（客户端 → recv）
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("websocket read goroutine panic",
						"path", c.Request.URL.Path,
						"recover", r,
					)
					cancel()
				}
			}()
			defer close(recv)
			for {
				msgType, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				msg := WSMessage{Type: msgType, Data: data}

				select {
				case recv <- msg:
				default:
					switch cfg.RecvPolicy {
					case Block:
						recv <- msg
					case DropNewest:
						// 丢弃
					case DropOldest:
						select {
						case <-recv:
						default:
						}
						select {
						case recv <- msg:
						default:
						}
					case Disconnect:
						cancel()
						return
					}
				}
			}
		}()

		// 9. 启动 ping goroutine，带 Pong 超时检测
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("websocket ping goroutine panic",
						"path", c.Request.URL.Path,
						"recover", r,
					)
					cancel()
				}
			}()
			ticker := time.NewTicker(cfg.PingPeriod)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// 检查是否超过 PongTimeout 没有收到 pong
					lastPongMu.Lock()
					sinceLastPong := time.Since(lastPong)
					lastPongMu.Unlock()

					if sinceLastPong > cfg.PongTimeout {
						// 超过 pong 超时时间，断开连接
						cancel()
						return
					}

					deadline := time.Now().Add(cfg.PongTimeout)
					conn.SetWriteDeadline(deadline)
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						cancel()
						return
					}
					conn.SetWriteDeadline(time.Time{})
				}
			}
		}()

		// 10. 启动写 goroutine（sender.Channel() → 客户端）
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("websocket write goroutine panic",
						"path", c.Request.URL.Path,
						"recover", r,
					)
				}
				cancel()
			}()
			for msg := range sender.Channel() {
				if cfg.WriteTimeout > 0 {
					conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
				}
				if err := conn.WriteMessage(msg.Type, msg.Data); err != nil {
					return
				}
				if cfg.WriteTimeout > 0 {
					conn.SetWriteDeadline(time.Time{})
				}
			}
		}()

		// 8. 调用用户 handler，传入代理 channel
		handler(ctx, recv, sender.Proxy())

		// 9. 关闭代理 channel 触发发送器退出
		close(sender.Proxy())

		// 10. 断开日志
		if err := ctx.Err(); err != nil {
			slog.Info("websocket client disconnected",
				"path", c.Request.URL.Path,
				"error", err,
			)
		}
	})
}

// StreamingGroup 创建一个流式路由组，用于 WebSocket 等长连接场景。
// 该组不会挂载 Timeout 中间件。
func (s *Server) StreamingGroup(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.getStreamingGroup().Group(relativePath, handlers...)
}

// SetGroups 设置普通和流式路由组，由 preset 调用。
func (s *Server) SetGroups(regular, streaming *gin.RouterGroup) {
	s.regularGroup = regular
	s.streamingGroup = streaming
}

// getRegularGroup 返回普通路由组。
func (s *Server) getRegularGroup() *gin.RouterGroup {
	if s.regularGroup != nil {
		return s.regularGroup
	}
	return &s.engine.RouterGroup
}

// getStreamingGroup 返回流式路由组。
func (s *Server) getStreamingGroup() *gin.RouterGroup {
	if s.streamingGroup != nil {
		return s.streamingGroup
	}
	return &s.engine.RouterGroup
}

// Errors 返回服务器运行期错误通道。
func (s *Server) Errors() <-chan error {
	return s.serveErrCh
}

// Serve 使用现成的 listener 启动服务器（阻塞）。
func (s *Server) Serve(ln net.Listener) error {
	if ln == nil {
		return fmt.Errorf("listener is nil")
	}
	if err := s.validateConfig(); err != nil {
		s.reportServeError(err)
		return err
	}
	return s.startInternal(ln, true, s.serveMainListener)
}

// Start 启动服务器（非阻塞）
func (s *Server) Start() error {
	return s.startWithNewListener(false)
}

// Run 启动服务器（阻塞）
func (s *Server) Run() error {
	return s.startWithNewListener(true)
}

// RunTLS 启动HTTPS服务器（阻塞）
func (s *Server) RunTLS(certFile, keyFile string) error {
	return s.startWithNewListenerTLS(certFile, keyFile)
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

// RunWithContext 启动服务器，当 ctx 取消时自动优雅关闭（阻塞）。
// 适用于 errgroup 等并发控制场景。
func (s *Server) RunWithContext(ctx context.Context) error {
	if err := s.Start(); err != nil {
		return err
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.Shutdown(shutdownCtx)
}

// WaitForShutdown 等待关闭信号并执行优雅关闭
func (s *Server) WaitForShutdown() error {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	// 阻塞等待信号
	<-quit

	// 先标记为 draining，让 readiness 立即返回 503
	if err := s.MarkDraining(); err != nil {
		return fmt.Errorf("mark draining: %w", err)
	}
	s.emitHook(s.hooks.OnShuttingDown, s.lifecycleEvent(nil))

	// 等待 DrainTimeout，让负载均衡器有时间将流量切走
	if s.config.DrainTimeout > 0 {
		time.Sleep(s.config.DrainTimeout)
	}

	// 创建关闭 context（ShutdownTimeout 用于 http.Server.Shutdown 的超时）
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// 优雅关闭
	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭失败: %w", err)
	}

	s.emitHook(s.hooks.OnShutdownComplete, s.lifecycleEvent(nil))
	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil && s.healthServer == nil {
		if s.State() == StateStopped {
			return nil
		}
		if err := s.tryTransitionTo(StateStopped); err != nil {
			return fmt.Errorf("transition to stopped: %w", err)
		}
		return nil
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()
	}

	if err := s.tryTransitionTo(StateStopping); err != nil {
		return fmt.Errorf("transition to stopping: %w", err)
	}

	if s.healthServer != nil {
		if err := s.healthServer.Shutdown(ctx); err != nil {
			_ = s.tryTransitionTo(StateFailed)
			return fmt.Errorf("shutdown health server: %w", err)
		}
		s.healthServer = nil
		s.healthAddr = ""
	}

	if s.server == nil {
		if err := s.tryTransitionTo(StateStopped); err != nil {
			return fmt.Errorf("transition to stopped: %w", err)
		}
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		_ = s.tryTransitionTo(StateFailed)
		return fmt.Errorf("shutdown server: %w", err)
	}

	// 关闭成功后清理 server 引用
	s.server = nil
	if err := s.tryTransitionTo(StateStopped); err != nil {
		return fmt.Errorf("transition to stopped: %w", err)
	}
	return nil
}

// Addr 返回服务器地址
func (s *Server) Addr() string {
	if s.server != nil {
		return s.server.Addr
	}
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// HealthAddr 返回健康检查服务器地址。
// 如果健康检查未启用，返回空字符串。
// 如果健康检查使用独立端口，返回独立地址；否则返回主服务器地址。
func (s *Server) HealthAddr() string {
	if !s.config.EnableHealthCheck {
		return ""
	}

	// 如果有独立健康检查端口，返回保存的地址
	if s.config.HealthCheckPort != 0 {
		return s.healthAddr
	}

	// 否则返回主服务器地址
	return s.Addr()
}

// IsRunning 检查服务器是否正在运行
// 基于 State() 判断：Ready 或 Draining 状态时返回 true
func (s *Server) IsRunning() bool {
	switch s.State() {
	case StateReady, StateDraining:
		return true
	default:
		return false
	}
}

// 中间件函数（可选使用）

// TraceIDMiddleware 添加 Trace ID 的中间件
func TraceIDMiddleware() gin.HandlerFunc {
	return httpmiddleware.TraceID()
}

// RequestIDMiddleware 添加 Request ID 的中间件（每个请求唯一）
func RequestIDMiddleware() gin.HandlerFunc {
	return httpmiddleware.RequestID()
}

// CORSMiddleware CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return httpmiddleware.CORS(httpmiddleware.CORSConfig{})
}

// RealIPMiddleware 可信客户端 IP 解析中间件
// 默认配置：不信任任何代理，直接使用 RemoteAddr
func RealIPMiddleware() gin.HandlerFunc {
	return httpmiddleware.RealIP()
}

// RealIPMiddlewareWithConfig 使用自定义配置的 RealIP 中间件
func RealIPMiddlewareWithConfig(config httpmiddleware.RealIPConfig) gin.HandlerFunc {
	return httpmiddleware.RealIPWithConfig(config)
}

// GetTraceID 从 context 中获取 trace id
func GetTraceID(c *gin.Context) string {
	if traceID, exists := c.Get(utils.TraceIDKey); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// GetRequestID 从 context 中获取 request id
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(utils.RequestIDKey); exists {
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
	Status    string                 `json:"status"`            // healthy, unhealthy
	Timestamp int64                  `json:"timestamp"`         // Unix时间戳
	Version   string                 `json:"version,omitempty"` // 应用版本
	Uptime    int64                  `json:"uptime,omitempty"`  // 运行时间（秒）
	Checks    map[string]interface{} `json:"checks,omitempty"`  // 详细检查结果
	Error     string                 `json:"error,omitempty"`   // 错误信息
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	GetName() string
}

// HealthCheckManager 健康检查管理器
type HealthCheckManager struct {
	checkers     []HealthChecker
	startTime    time.Time
	version      string
	checkTimeout time.Duration
}

// HealthCheckOption 描述 HealthCheckManager 的可选配置。
type HealthCheckOption func(*HealthCheckManager)

// WithCheckTimeout 设置单个 checker 的超时时间。
func WithCheckTimeout(timeout time.Duration) HealthCheckOption {
	return func(hcm *HealthCheckManager) {
		hcm.checkTimeout = timeout
	}
}

const defaultCheckTimeout = 5 * time.Second

// NewHealthCheckManager 创建健康检查管理器
func NewHealthCheckManager(version string, opts ...HealthCheckOption) *HealthCheckManager {
	hcm := &HealthCheckManager{
		checkers:     make([]HealthChecker, 0),
		startTime:    time.Now(),
		version:      version,
		checkTimeout: defaultCheckTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(hcm)
		}
	}
	return hcm
}

// AddChecker 添加健康检查器
func (hcm *HealthCheckManager) AddChecker(checker HealthChecker) {
	hcm.checkers = append(hcm.checkers, checker)
}

type checkResult struct {
	name       string
	err        error
	durationMs int64
}

// CheckHealth 并发执行所有健康检查
func (hcm *HealthCheckManager) CheckHealth(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Version:   hcm.version,
		Uptime:    int64(time.Since(hcm.startTime).Seconds()),
		Checks:    make(map[string]interface{}),
	}

	if len(hcm.checkers) == 0 {
		return status
	}

	results := make(chan checkResult, len(hcm.checkers))

	for _, checker := range hcm.checkers {
		go func(c HealthChecker) {
			checkCtx, cancel := context.WithTimeout(context.Background(), hcm.checkTimeout)
			defer cancel()
			start := time.Now()
			err := c.CheckHealth(checkCtx)
			results <- checkResult{
				name:       c.GetName(),
				err:        err,
				durationMs: time.Since(start).Milliseconds(),
			}
		}(checker)
	}

	for range hcm.checkers {
		r := <-results
		if r.err != nil {
			status.Status = "unhealthy"
			status.Checks[r.name] = map[string]interface{}{
				"status":      "unhealthy",
				"error":       r.err.Error(),
				"duration_ms": r.durationMs,
			}
		} else {
			status.Checks[r.name] = map[string]interface{}{
				"status":      "healthy",
				"duration_ms": r.durationMs,
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

// EnableHealthCheck 启用健康检查并注册默认健康检查路由。
func (s *Server) EnableHealthCheck() {
	s.config.EnableHealthCheck = true
	s.healthHandler = DefaultHealthHandler()
	s.registerHealthRoute()
	s.registerProbeRoutes()
}

// EnableHealthCheckWithManager 启用带管理器的健康检查并注册探针路由。
func (s *Server) EnableHealthCheckWithManager(manager *HealthCheckManager) {
	s.config.EnableHealthCheck = true
	s.healthHandler = HealthHandlerWithManager(manager)
	s.registerHealthRoute()
	s.registerProbeRoutes()
}

// SetHealthCheckPath 设置健康检查路径。
// 必须在健康检查路由注册或服务器启动前调用，否则返回错误。
func (s *Server) SetHealthCheckPath(path string) error {
	if s.healthRouteRegistered || s.readinessRouteRegistered || s.livenessRouteRegistered || s.server != nil || s.healthServer != nil {
		return fmt.Errorf("health check path must be configured before route registration or startup")
	}
	s.config.HealthCheckPath = path
	return nil
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
	name   string
	url    string
	client *http.Client
}

// NewHTTPHealthChecker 创建HTTP服务健康检查器
func NewHTTPHealthChecker(name, url string, timeout time.Duration) *HTTPHealthChecker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &HTTPHealthChecker{
		name:   name,
		url:    url,
		client: &http.Client{Timeout: timeout},
	}
}

// CheckHealth 检查HTTP服务健康状态
func (hhc *HTTPHealthChecker) CheckHealth(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hhc.url, nil)
	if err != nil {
		return err
	}

	resp, err := hhc.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

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
