package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WSBufferPolicy 定义缓冲满时的处理策略
type WSBufferPolicy int

const (
	Block      WSBufferPolicy = iota // 阻塞等待
	DropNewest                       // 丢弃最新消息
	DropOldest                       // 丢弃最旧消息
	Disconnect                       // 断开连接
)

// WSConfig 是 WebSocket 配置
type WSConfig struct {
	RecvBufferSize int
	SendBufferSize int
	RecvPolicy     WSBufferPolicy
	// SendPolicy 当前未生效，预留用于未来版本
	// 目前 send channel 的阻塞行为由用户 handler 控制
	SendPolicy     WSBufferPolicy
	PingPeriod     time.Duration
	PongTimeout    time.Duration
}

// WSRouteConfig 是 WebSocket 路由级配置
type WSRouteConfig struct {
	WSConfig
	ReadTimeout  time.Duration // 读取消息超时，0 = 无超时
	WriteTimeout time.Duration // 发送消息超时，0 = 无超时
}

func defaultWSConfig() WSConfig {
	return WSConfig{
		RecvBufferSize: 100,
		SendBufferSize: 100,
		RecvPolicy:     DropNewest,
		SendPolicy:     DropOldest,
		PingPeriod:     30 * time.Second,
		PongTimeout:    60 * time.Second,
	}
}

func defaultWSRouteConfig() WSRouteConfig {
	return WSRouteConfig{
		WSConfig:     defaultWSConfig(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// WSUpgrader 是 WebSocket upgrader，可由用户自定义
var WSUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应配置
	},
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
	Recv() <-chan WSMessage
	Send(msg WSMessage) error
	TrySend(msg WSMessage) bool
	Close(code int, reason string) error
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

// WithReadTimeout 设置读取消息超时
func WithReadTimeout(d time.Duration) WSRouteOption {
	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.ReadTimeout = d
	})
}

// WithWriteTimeout 设置发送消息超时
func WithWriteTimeout(d time.Duration) WSRouteOption {
	return wsRouteOptionFunc(func(cfg *WSRouteConfig) {
		cfg.WriteTimeout = d
	})
}

// WithRecvBuffer 设置接收缓冲大小和策略
func WithRecvBuffer(size int, policy WSBufferPolicy) wsOptionFunc {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.RecvBufferSize = size
		cfg.RecvPolicy = policy
	})
}

// WithSendBuffer 设置发送缓冲大小和策略
func WithSendBuffer(size int, policy WSBufferPolicy) wsOptionFunc {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.SendBufferSize = size
		cfg.SendPolicy = policy
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
