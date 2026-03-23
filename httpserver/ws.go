package httpserver

import (
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
