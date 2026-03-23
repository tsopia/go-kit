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

// WSHandlerFunc 是 WebSocket handler 的函数签名
type WSHandlerFunc func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage)

// WSOption 是 WebSocket 配置选项
type WSOption interface {
	apply(*WSConfig)
}

type wsOptionFunc func(*WSConfig)

func (f wsOptionFunc) apply(cfg *WSConfig) {
	f(cfg)
}

// WithRecvBuffer 设置接收缓冲大小和策略
func WithRecvBuffer(size int, policy WSBufferPolicy) WSOption {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.RecvBufferSize = size
		cfg.RecvPolicy = policy
	})
}

// WithSendBuffer 设置发送缓冲大小和策略
func WithSendBuffer(size int, policy WSBufferPolicy) WSOption {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.SendBufferSize = size
		cfg.SendPolicy = policy
	})
}

// WithWSPingPeriod 设置 ping 发送周期
func WithWSPingPeriod(period time.Duration) WSOption {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.PingPeriod = period
	})
}

// WithWSPongTimeout 设置 pong 超时时间
func WithWSPongTimeout(timeout time.Duration) WSOption {
	return wsOptionFunc(func(cfg *WSConfig) {
		cfg.PongTimeout = timeout
	})
}
