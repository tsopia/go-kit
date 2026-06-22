package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
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
	regularGroupSpec   *routeGroupSpec // 有 Timeout 中间件
	streamingGroupSpec *routeGroupSpec // 无 Timeout 中间件，用于 SSE/WS
}

type routeGroupSpec struct {
	basePath      string
	localHandlers gin.HandlersChain
	frozen        *gin.RouterGroup
}

// newRouteGroupSpec 将一个 RouterGroup 转换为延迟重建规格。
//
// 设计目的：preset 在 srv.Use() 之前创建 regularGroup/streamingGroup，
// 后续用户调用 srv.Use(middleware.AccessLog()) 添加的中间件不会自动
// 继承到已创建的 group 上。routeGroupSpec 通过"延迟重建"解决此问题：
// 如果 group 的 Handlers 前缀与 engine.Handlers 完全一致，说明 group
// 没有添加额外的引擎级中间件，只需保存 basePath 和本地 handlers，
// 在注册路由时再从 engine 重建 group，从而继承后续添加的公共中间件。
//
// 函数指针比较使用 reflect.ValueOf().Pointer()。虽然 Go 规范未保证
// 函数值的指针稳定性，但在实践中对于同一函数变量，此比较是可靠的。
func newRouteGroupSpec(root gin.HandlersChain, group *gin.RouterGroup) *routeGroupSpec {
	if group == nil {
		return nil
	}

	if len(group.Handlers) < len(root) {
		return &routeGroupSpec{frozen: group}
	}

	for i := range root {
		if reflect.ValueOf(root[i]).Pointer() != reflect.ValueOf(group.Handlers[i]).Pointer() {
			return &routeGroupSpec{frozen: group}
		}
	}

	return &routeGroupSpec{
		basePath:      group.BasePath(),
		localHandlers: append(gin.HandlersChain(nil), group.Handlers[len(root):]...),
	}
}

func (s *Server) buildGroup(spec *routeGroupSpec) *gin.RouterGroup {
	if spec == nil {
		return &s.engine.RouterGroup
	}
	if spec.frozen != nil {
		return spec.frozen
	}

	group := s.engine.Group(spec.basePath)
	if len(spec.localHandlers) > 0 {
		group.Use(spec.localHandlers...)
	}
	return group
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

	// 如果启用了健康检查，自动注册健康检查路由。
	// 注意：这些路由直接注册在 engine 上，此时还没有任何中间件。
	// 后续 srv.Use() 添加的中间件不会影响这些路由（Gin 的快照语义）。
	// 如需健康检查路由也经过中间件链，请使用独立 HealthCheckPort。
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

// SSE 注册一个 Server-Sent Events 路由（GET 方法）。
// 自动设置 SSE 响应头，清除 WriteDeadline，使用 streamingGroup（无 Timeout 中间件）。
func (s *Server) SSE(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
	s.sseRegister("GET", relativePath, handler, opts...)
}

// SSEPost 注册一个 Server-Sent Events 路由（POST 方法）。
// 行为与 SSE 相同，但使用 POST 方法注册，支持通过 request body 传递较长输入。
func (s *Server) SSEPost(relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
	s.sseRegister("POST", relativePath, handler, opts...)
}

// sseRegister 是 SSE/SSEPost 的共用注册逻辑。
func (s *Server) sseRegister(method, relativePath string, handler SSEHandlerFunc, opts ...SSEOption) {
	cfg := &sseConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	ginHandler := func(c *gin.Context) {
		startedAt := time.Now()

		// 清除 WriteDeadline。SSE 是长连接，deadline 只做 best effort，
		// 不支持 deadline 的 writer 会返回错误，但不影响流式响应继续工作。
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
		stream := &sseSender{ginCtx: c, ctx: ctx, config: cfg, startedAt: startedAt}
		stream.logConnect()

		c.Set(utils.StreamingKey, "sse")
		if obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context()); ok && obs != nil {
			obs.OnConnect("sse")
			defer obs.OnDisconnect("sse")
		}

		// 启动心跳（如果配置了）
		stopHeartbeat := stream.startHeartbeat(ctx)

		// 调用 handler
		handler(stream)

		// handler 返回后主动停止 heartbeat，允许有限流自然结束
		stopHeartbeat()

		// 连接断开时打印日志
		stream.logDisconnect(ctx)
	}

	switch method {
	case "POST":
		s.getStreamingGroup().POST(relativePath, ginHandler)
	default:
		s.getStreamingGroup().GET(relativePath, ginHandler)
	}
}

// WS 注册一个 WebSocket 路由。
// 自动处理 Upgrade、ping/pong、缓冲策略和断开日志。
func (s *Server) WS(relativePath string, handler WSHandlerFunc, opts ...WSRouteOption) {
	cfg := defaultWSRouteConfig()
	for _, opt := range opts {
		opt.applyRoute(&cfg)
	}

	s.getStreamingGroup().GET(relativePath, func(c *gin.Context) {
		startedAt := time.Now()

		// 1. Upgrade 连接
		upgrader := WSUpgrader
		if cfg.CheckOrigin != nil {
			upgrader.CheckOrigin = cfg.CheckOrigin
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logStreamEvent(c, "warn", "ws_upgrade_failed", "ws", time.Time{}, err)
			return
		}
		logStreamEvent(c, "info", "stream_connect", "ws", time.Time{}, nil)

		c.Set(utils.StreamingKey, "ws")
		if obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context()); ok && obs != nil {
			obs.OnConnect("ws")
			defer obs.OnDisconnect("ws")
		}

		defer func() {
			if err := conn.Close(); err != nil {
				slog.Debug("websocket close failed",
					"path", c.Request.URL.Path,
					"error", err,
				)
			}
		}()

		// 2. 创建 session context
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		readIdleTimeout := cfg.ReadIdleTimeout
		if readIdleTimeout <= 0 {
			readIdleTimeout = cfg.PongTimeout
		}

		controlDeadline := func() time.Time {
			switch {
			case cfg.WriteTimeout > 0:
				return time.Now().Add(cfg.WriteTimeout)
			case cfg.PongTimeout > 0:
				return time.Now().Add(cfg.PongTimeout)
			default:
				return time.Time{}
			}
		}

		if readIdleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(readIdleTimeout))
		}
		conn.SetPongHandler(func(string) error {
			if readIdleTimeout <= 0 {
				return nil
			}
			return conn.SetReadDeadline(time.Now().Add(readIdleTimeout))
		})

		send := make(chan WSMessage, cfg.SendBufferSize)
		recv := make(chan WSMessage, 1)
		writeDone := make(chan struct{})

		closeConn := func(code int, reason string) error {
			cancel()
			err := conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(code, reason),
				controlDeadline(),
			)
			if closeErr := conn.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
			return err
		}

		session := &wsSession{
			ctx:     ctx,
			request: c.Request,
			params:  c.Params,
			keys:    cloneGinKeys(c.Keys),
			recv:    recv,
			send:    send,
			closeFn: closeConn,
			gracefulCloseFn: func(closeCtx context.Context, code int, reason string) error {
				if closeCtx == nil {
					closeCtx = context.Background()
				}
				close(send)
				select {
				case <-writeDone:
					return closeConn(code, reason)
				case <-closeCtx.Done():
					if err := closeConn(code, reason); err != nil {
						return err
					}
					return closeCtx.Err()
				}
			},
		}

		var pumps sync.WaitGroup

		pumps.Add(1)
		go func() {
			defer recoverWSPumpPanic("read", c.Request.URL.Path, cancel)
			defer pumps.Done()
			defer close(recv)
			defer cancel()
			for {
				msgType, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if readIdleTimeout > 0 {
					_ = conn.SetReadDeadline(time.Now().Add(readIdleTimeout))
				}
				select {
				case recv <- WSMessage{Type: msgType, Data: data}:
				case <-ctx.Done():
					return
				}
			}
		}()

		pumps.Add(1)
		go func() {
			defer recoverWSPumpPanic("write", c.Request.URL.Path, cancel)
			defer pumps.Done()
			defer cancel()
			defer close(writeDone)
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-send:
					if !ok {
						return
					}
					if cfg.WriteTimeout > 0 {
						// Best effort: closed/broken connections会在后续 WriteMessage 暴露错误。
						_ = conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
					}
					if err := conn.WriteMessage(msg.Type, msg.Data); err != nil {
						return
					}
					if cfg.WriteTimeout > 0 {
						// Best effort: 重置 deadline 失败不影响后续连接关闭与错误传播。
						_ = conn.SetWriteDeadline(time.Time{})
					}
				}
			}
		}()

		if cfg.PingPeriod > 0 {
			pumps.Add(1)
			go func() {
				defer recoverWSPumpPanic("ping", c.Request.URL.Path, cancel)
				defer pumps.Done()
				ticker := time.NewTicker(cfg.PingPeriod)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := conn.WriteControl(websocket.PingMessage, nil, controlDeadline()); err != nil {
							cancel()
							return
						}
					}
				}
			}()
		}

		handler(session)

		_ = session.Close(websocket.CloseNormalClosure, "")
		pumps.Wait()
		logStreamEvent(c, "info", "stream_disconnect", "ws", startedAt, ctx.Err())
	})
}

// StreamingGroup 创建一个流式路由组，用于 WebSocket 等长连接场景。
// 该组不会挂载 Timeout 中间件。
func (s *Server) StreamingGroup(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.getStreamingGroup().Group(relativePath, handlers...)
}

// SetGroups 设置普通和流式路由组，由 preset 调用。
func (s *Server) SetGroups(regular, streaming *gin.RouterGroup) {
	rootHandlers := append(gin.HandlersChain(nil), s.engine.Handlers...)
	s.regularGroupSpec = newRouteGroupSpec(rootHandlers, regular)
	s.streamingGroupSpec = newRouteGroupSpec(rootHandlers, streaming)
}

// getRegularGroup 返回普通路由组。
func (s *Server) getRegularGroup() *gin.RouterGroup {
	return s.buildGroup(s.regularGroupSpec)
}

// getStreamingGroup 返回流式路由组。
func (s *Server) getStreamingGroup() *gin.RouterGroup {
	return s.buildGroup(s.streamingGroupSpec)
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
	return s.gracefulShutdownSequence()
}

// WaitForShutdown 等待关闭信号并执行优雅关闭
func (s *Server) WaitForShutdown() error {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	// 阻塞等待信号
	<-quit

	return s.gracefulShutdownSequence()
}

func (s *Server) gracefulShutdownSequence() error {
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
	ctx, cancel := s.shutdownContext()
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
		ctx, cancel = s.shutdownContext()
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

// cloneGinKeys 复制 gin.Context.Keys 快照，供 WS pump goroutine 安全读取。
// gin.Context.Keys 的 key 类型为 any，此处只保留 string key。
func cloneGinKeys(src map[any]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if sk, ok := k.(string); ok {
			dst[sk] = v
		}
	}
	return dst
}
