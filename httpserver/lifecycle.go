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

func (s *Server) prepareMainServer(ln net.Listener) {
	s.server = s.buildHTTPServer(ln.Addr().String(), s.engine)
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
