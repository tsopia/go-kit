package httpserver

import (
	"context"
	"net/http"
)

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

// HTTPServerMutator 是修改 http.Server 的函数类型。
type HTTPServerMutator func(*http.Server)

// WithHTTPServerMutator 注册一个在 http.Server 创建后、启动前调用的 mutator。
// 用于需要修改底层 http.Server 高级配置的场景。
func WithHTTPServerMutator(mutator HTTPServerMutator) Option {
	return func(s *Server) {
		s.serverMutators = append(s.serverMutators, mutator)
	}
}

func (s *Server) applyOptions(opts ...Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
}
