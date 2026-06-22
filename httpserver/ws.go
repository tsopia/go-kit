package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WSConfig 是 WebSocket 配置
type WSConfig struct {
	SendBufferSize int
	PingPeriod     time.Duration
	PongTimeout    time.Duration
}

// WSRouteConfig 是 WebSocket 路由级配置
type WSRouteConfig struct {
	WSConfig
	ReadIdleTimeout time.Duration // 0 = 使用 PongTimeout
	WriteTimeout    time.Duration // 发送消息超时，0 = 无超时
	CheckOrigin     func(*http.Request) bool
}

func defaultWSConfig() WSConfig {
	return WSConfig{
		SendBufferSize: 100,
		PingPeriod:     30 * time.Second,
		PongTimeout:    60 * time.Second,
	}
}

func defaultWSRouteConfig() WSRouteConfig {
	return WSRouteConfig{
		WSConfig:     defaultWSConfig(),
		WriteTimeout: 10 * time.Second,
		CheckOrigin:  defaultWSOriginCheck,
	}
}

func defaultWSOriginCheck(r *http.Request) bool {
	if r == nil {
		return true
	}
	return r.Header.Get("Origin") == ""
}

func recoverWSPumpPanic(pump string, path string, cancel context.CancelFunc) {
	if r := recover(); r != nil {
		slog.Error("websocket "+pump+" goroutine panic",
			"path", path,
			"recover", r,
		)
		if cancel != nil {
			cancel()
		}
	}
}

// WSUpgrader 是 WebSocket upgrader，可由用户自定义
var WSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     defaultWSOriginCheck,
}

// WSMessage 是 WebSocket 消息
type WSMessage struct {
	Type int    // websocket.TextMessage 或 BinaryMessage
	Data []byte // 消息内容
}

// WSSession 描述 WebSocket handler 可见的连接能力。
type WSSession interface {
	Context() context.Context
	Request() *http.Request
	Param(name string) string
	Get(key string) (any, bool)
	GetString(key string) (string, bool)
	Recv() <-chan WSMessage
	Send(msg WSMessage) error
	TrySend(msg WSMessage) bool
	Close(code int, reason string) error
	CloseGracefully(ctx context.Context, code int, reason string) error
}

// WSHandlerFunc 是 WebSocket handler 的函数签名
type WSHandlerFunc func(session WSSession)

// WSOption 是 WebSocket 配置选项
type WSOption interface {
	apply(*WSConfig)
}

type wsOptionFunc func(*WSConfig)

func (f wsOptionFunc) apply(cfg *WSConfig) {
	f(cfg)
}

func (f wsOptionFunc) applyRoute(cfg *WSRouteConfig) {
	f(&cfg.WSConfig)
}

// WSRouteOption 是 WebSocket 路由级配置选项
type WSRouteOption interface {
	applyRoute(*WSRouteConfig)
}

type wsRouteOptionFunc func(*WSRouteConfig)

func (f wsRouteOptionFunc) applyRoute(cfg *WSRouteConfig) {
	f(cfg)
}

// WithReadIdleTimeout 设置读空闲超时。
func WithReadIdleTimeout(d time.Duration) WSRouteOption {
	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.ReadIdleTimeout = d
	})
}

// WithWriteTimeout 设置发送消息超时
func WithWriteTimeout(d time.Duration) WSRouteOption {
	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.WriteTimeout = d
	})
}

// WithWSAllowedOrigins 显式允许指定浏览器 Origin。
// 不带 Origin 的请求仍会放行，便于非浏览器客户端使用。
func WithWSAllowedOrigins(origins ...string) WSRouteOption {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}

	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.CheckOrigin = func(r *http.Request) bool {
			if r == nil {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			_, ok := allowed[origin]
			return ok
		}
	})
}

// WithWSOriginChecker 自定义 WebSocket Origin 校验逻辑。
func WithWSOriginChecker(fn func(*http.Request) bool) WSRouteOption {
	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.CheckOrigin = fn
	})
}

// WithWSSendBuffer 设置发送队列大小。
func WithWSSendBuffer(size int) wsOptionFunc {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.SendBufferSize = size
	})
}

// WithWSPingPeriod 设置 ping 发送周期
func WithWSPingPeriod(period time.Duration) wsOptionFunc {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.PingPeriod = period
	})
}

// WithWSPongTimeout 设置 pong 超时时间
func WithWSPongTimeout(timeout time.Duration) wsOptionFunc {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.PongTimeout = timeout
	})
}
