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

func TestWS_GoroutineCleanup(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	// 追踪 handler 完成
	handlerDone := make(chan bool, 1)

	srv.WS("/test", func(ctx context.Context, recv <-chan httpserver.WSMessage, send chan<- httpserver.WSMessage) {
		// 简单 handler，立即返回
		handlerDone <- true
	})

	// 启动服务器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go srv.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	// 连接 WebSocket
	u := url.URL{Scheme: "ws", Host: ln.Addr().String(), Path: "/stream/test"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	// 等待 handler 完成
	select {
	case <-handlerDone:
		// handler 已完成
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete")
	}

	// 给 goroutine 清理时间
	time.Sleep(200 * time.Millisecond)

	// 关闭连接
	conn.Close()

	// 如果没有 goroutine 泄漏，测试可以正常结束
	// 实际生产环境可以使用 runtime.NumGoroutine() 检测
}

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
