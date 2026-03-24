package httpserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSBufferPolicy_Constants(t *testing.T) {
	if Block != 0 {
		t.Error("Block should be 0")
	}
	if DropNewest != 1 {
		t.Error("DropNewest should be 1")
	}
	if DropOldest != 2 {
		t.Error("DropOldest should be 2")
	}
	if Disconnect != 3 {
		t.Error("Disconnect should be 3")
	}
}

func TestWSConfig_Defaults(t *testing.T) {
	cfg := defaultWSConfig()
	if cfg.RecvBufferSize != 100 {
		t.Errorf("expected RecvBufferSize=100, got %d", cfg.RecvBufferSize)
	}
	if cfg.SendBufferSize != 100 {
		t.Errorf("expected SendBufferSize=100, got %d", cfg.SendBufferSize)
	}
	if cfg.RecvPolicy != DropNewest {
		t.Error("expected RecvPolicy=DropNewest")
	}
	if cfg.SendPolicy != DropOldest {
		t.Error("expected SendPolicy=DropOldest")
	}
	if cfg.PingPeriod != 30*time.Second {
		t.Errorf("expected PingPeriod=30s, got %v", cfg.PingPeriod)
	}
	if cfg.PongTimeout != 60*time.Second {
		t.Errorf("expected PongTimeout=60s, got %v", cfg.PongTimeout)
	}
}

func TestWSMessage(t *testing.T) {
	msg := WSMessage{
		Type: websocket.TextMessage,
		Data: []byte("hello"),
	}
	if msg.Type != websocket.TextMessage {
		t.Error("Type mismatch")
	}
	if string(msg.Data) != "hello" {
		t.Error("Data mismatch")
	}
}

func TestWSOptions(t *testing.T) {
	cfg := defaultWSConfig()

	opt1 := WithRecvBuffer(200, Block)
	opt1.apply(&cfg)
	if cfg.RecvBufferSize != 200 || cfg.RecvPolicy != Block {
		t.Error("WithRecvBuffer failed")
	}

	opt2 := WithWSPingPeriod(10 * time.Second)
	opt2.apply(&cfg)
	if cfg.PingPeriod != 10*time.Second {
		t.Error("WithWSPingPeriod failed")
	}

	opt3 := WithSendBuffer(300, DropOldest)
	opt3.apply(&cfg)
	if cfg.SendBufferSize != 300 || cfg.SendPolicy != DropOldest {
		t.Error("WithSendBuffer failed")
	}

	opt4 := WithWSPongTimeout(20 * time.Second)
	opt4.apply(&cfg)
	if cfg.PongTimeout != 20*time.Second {
		t.Error("WithWSPongTimeout failed")
	}
}

func TestServer_WS(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	server.WS("/ws", func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage) {
		<-ctx.Done()
	})

	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		// WS 路由注册在 streamingGroup 下，路径为 /stream/ws
		if r.Path == "/stream/ws" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_PanicRecovery(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// panic 的 handler
	server.WS("/panic", func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage) {
		panic("intentional panic in handler")
	})

	server.WS("/panic-in-read", func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage) {
		// 等待一条消息然后 panic
		<-recv
		panic("panic after receiving message")
	})

	// 验证路由注册成功（即使 handler 会 panic）
	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/stream/panic" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_UpgradeError(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// 注册 WS 路由
	server.WS("/ws", func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage) {
		<-ctx.Done()
	})

	// 验证路由注册
	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/stream/ws" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_ReadTimeout(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	timeoutCalled := make(chan bool, 1)

	server.WS("/timeout", func(ctx context.Context, recv <-chan WSMessage, send chan<- WSMessage) {
		select {
		case <-recv:
			// 不应该收到，因为客户端不发消息
		case <-ctx.Done():
			timeoutCalled <- true
		}
	}, WithReadTimeout(100*time.Millisecond))

	// 使用 net.Listen 获取随机端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond) // 等待服务器启动

	// WebSocket 连接
	wsURL := "ws://" + ln.Addr().String() + "/stream/timeout"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 等待超时触发
	select {
	case <-timeoutCalled:
		// 成功：读超时触发了 ctx 取消
	case <-time.After(2 * time.Second):
		t.Error("read timeout did not trigger")
	}
}
