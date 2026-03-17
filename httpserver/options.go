package httpserver

import "context"

// LifecycleEvent 描述服务器生命周期事件。
type LifecycleEvent struct {
	Addr       string
	HealthAddr string
	Err        error
}

// Hooks 描述服务器生命周期的可选回调。
type Hooks struct {
	OnStarting         func(context.Context, LifecycleEvent)
	OnStarted          func(context.Context, LifecycleEvent)
	OnServeError       func(context.Context, LifecycleEvent)
	OnShuttingDown     func(context.Context, LifecycleEvent)
	OnShutdownComplete func(context.Context, LifecycleEvent)
}

// Option 描述 Server 的可选配置项。
type Option func(*Server)

// WithHooks 为服务器注入生命周期 hooks。
func WithHooks(h Hooks) Option {
	return func(s *Server) {
		s.hooks = h
	}
}

// WithModules 在构造时批量注册路由模块。
func WithModules(modules ...RouteModule) Option {
	return func(s *Server) {
		s.RegisterModules(modules...)
	}
}

// WithManualReadiness 禁用自动 ready 切换。
func WithManualReadiness() Option {
	return func(s *Server) {
		s.manualReady = true
	}
}

func (s *Server) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
}
