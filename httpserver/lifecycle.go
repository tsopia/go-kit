package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

func normalizeConfig(config *Config) *Config {
	if config == nil {
		return DefaultConfig()
	}

	normalized := *config
	normalized.Normalize()

	return &normalized
}

func (s *Server) validateConfig() error {
	if s == nil {
		return fmt.Errorf("server is nil")
	}

	return s.config.Validate()
}

func (s *Server) buildHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       s.config.ReadTimeout,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		MaxHeaderBytes:    s.config.MaxHeaderBytes,
	}
}

func (s *Server) emitHook(fn func(context.Context, LifecycleEvent), event LifecycleEvent) {
	if fn == nil {
		return
	}

	fn(context.Background(), event)
}

func (s *Server) lifecycleEvent(err error) LifecycleEvent {
	healthAddr := s.healthAddr
	if healthAddr == "" && s.config.EnableHealthCheck && s.config.HealthCheckPort == 0 {
		healthAddr = s.Addr()
	}

	return LifecycleEvent{
		Addr:       s.Addr(),
		HealthAddr: healthAddr,
		Err:        err,
	}
}

func (s *Server) reportServeError(err error) {
	if err == nil || err == http.ErrServerClosed {
		return
	}

	event := s.lifecycleEvent(err)
	s.emitHook(s.hooks.OnServeError, event)

	select {
	case s.serveErrCh <- err:
	default:
	}
}

func (s *Server) configuredAddr() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// startInternal 是统一启动入口，所有公开启动方法最终都走这里
// isBlocking: true 表示阻塞直到服务器关闭，false 表示非阻塞启动
// serveFn: 实际的服务启动函数，可以是 s.serveMainListener 或 serveTLS 等
func (s *Server) startInternal(ln net.Listener, isBlocking bool, serveFn func(net.Listener) error) error {
	// 从 New 或 Failed 状态转换到 Starting
	// 使用 setState 而不是 transitionTo 来允许从任意初始状态启动
	s.setState(StateStarting)

	s.prepareMainServer(ln)
	healthLn, err := s.prepareHealthServer()
	if err != nil {
		s.reportServeError(err)
		return err
	}

	s.emitHook(s.hooks.OnStarting, s.lifecycleEvent(nil))

	// 启动健康检查服务器（如果有独立端口）
	if healthLn != nil {
		go func() {
			_ = s.serveHealthListener(healthLn)
		}()
	}

	// 设置就绪状态
	if !s.manualReady {
		s.MarkReady()
	}

	s.emitHook(s.hooks.OnStarted, s.lifecycleEvent(nil))

	// 阻塞或非阻塞服务
	if isBlocking {
		return serveFn(ln)
	}

	go func() {
		_ = serveFn(ln)
	}()
	return nil
}

// startWithNewListener 创建新 listener 并启动
func (s *Server) startWithNewListener(isBlocking bool) error {
	if err := s.validateConfig(); err != nil {
		s.reportServeError(err)
		return err
	}

	ln, err := net.Listen("tcp", s.configuredAddr())
	if err != nil {
		s.reportServeError(err)
		return err
	}

	return s.startInternal(ln, isBlocking, s.serveMainListener)
}

// startWithNewListenerTLS 创建新 listener 并用 TLS 启动
func (s *Server) startWithNewListenerTLS(certFile, keyFile string) error {
	if err := s.validateConfig(); err != nil {
		s.reportServeError(err)
		return err
	}

	ln, err := net.Listen("tcp", s.configuredAddr())
	if err != nil {
		s.reportServeError(err)
		return err
	}

	return s.startInternal(ln, true, func(l net.Listener) error {
		err := s.server.ServeTLS(l, certFile, keyFile)
		if err != nil && err != http.ErrServerClosed {
			s.reportServeError(err)
			return err
		}
		return nil
	})
}

func (s *Server) prepareMainServer(ln net.Listener) {
	s.server = s.buildHTTPServer(ln.Addr().String(), s.engine)

	// 应用用户注册的 http.Server mutators
	for _, mutator := range s.serverMutators {
		mutator(s.server)
	}
}

func (s *Server) healthEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.healthHandler == nil {
			c.Status(http.StatusNotFound)
			return
		}

		s.healthHandler(c)
	}
}

func (s *Server) registerHealthRoute() {
	if !s.config.EnableHealthCheck || s.config.HealthCheckPort != 0 || s.healthRouteRegistered {
		return
	}

	s.engine.GET(s.config.HealthCheckPath, s.healthEndpoint())
	s.healthRouteRegistered = true
}

func (s *Server) prepareHealthServer() (net.Listener, error) {
	if !s.config.EnableHealthCheck || s.config.HealthCheckPort == 0 {
		s.healthServer = nil
		s.healthAddr = ""
		return nil, nil
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Host, s.config.HealthCheckPort))
	if err != nil {
		return nil, err
	}

	engine := gin.New()
	engine.GET(s.config.HealthCheckPath, s.healthEndpoint())
	engine.GET(s.config.ReadinessPath, s.readinessEndpoint())
	engine.GET(s.config.LivenessPath, s.livenessEndpoint())
	s.healthServer = s.buildHTTPServer(ln.Addr().String(), engine)
	s.healthAddr = ln.Addr().String()

	return ln, nil
}

func (s *Server) serveMainListener(ln net.Listener) error {
	err := s.server.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		s.reportServeError(err)
		return err
	}

	return nil
}

func (s *Server) serveHealthListener(ln net.Listener) error {
	err := s.healthServer.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		s.reportServeError(err)
		return err
	}

	return nil
}
