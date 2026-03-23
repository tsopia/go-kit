package httpserver_test

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsopia/go-kit/httpserver"
)

func TestWS_Integration_Echo(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	// Echo 服务
	srv.WS("/echo", func(ctx context.Context, recv <-chan httpserver.WSMessage, send chan<- httpserver.WSMessage) {
		for msg := range recv {
			send <- msg
		}
	})

	// 启动服务器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go srv.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	// WebSocket 连接
	u := url.URL{Scheme: "ws", Host: ln.Addr().String(), Path: "/stream/echo"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// 发送消息
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}

	// 接收回显
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if msgType != websocket.TextMessage {
		t.Errorf("expected text message, got %d", msgType)
	}
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got '%s'", string(data))
	}
}
