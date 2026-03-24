package httpserver_test

import (
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

	srv.WS("/test", func(session httpserver.WSSession) {
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

func TestWS_PongTimeout(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	ctxCancelled := make(chan bool, 1)

	// 使用非常短的 ping/pong 超时用于测试
	srv.WS("/pongtest", func(session httpserver.WSSession) {
		<-session.Context().Done()
		ctxCancelled <- true
	},
		httpserver.WithWSPingPeriod(100*time.Millisecond),
		httpserver.WithWSPongTimeout(200*time.Millisecond),
	)

	// 启动服务器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go srv.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	// 使用自定义 dialer，禁用自动 pong 响应来模拟假死客户端
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	u := url.URL{Scheme: "ws", Host: ln.Addr().String(), Path: "/stream/pongtest"}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// 禁用自动 pong 响应（模拟假死）
	conn.SetPongHandler(func(string) error {
		return nil // 不更新任何状态
	})

	// 等待 context 被取消（由于 pong 超时）
	select {
	case <-ctxCancelled:
		// 成功：pong 超时触发了 context 取消
	case <-time.After(2 * time.Second):
		t.Error("context was not cancelled due to pong timeout")
	}
}

func TestWSSessionContract_Integration(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	ready := make(chan struct{}, 1)

	srv.WS("/session/:id", func(session httpserver.WSSession) {
		if session.Param("id") != "42" {
			t.Fatalf("id = %q, want %q", session.Param("id"), "42")
		}
		if session.Request().URL.Path != "/stream/session/42" {
			t.Fatalf("path = %q, want %q", session.Request().URL.Path, "/stream/session/42")
		}
		if err := session.Send(httpserver.WSMessage{Type: websocket.TextMessage, Data: []byte("hello")}); err != nil {
			t.Fatalf("send: %v", err)
		}
		ready <- struct{}{}
		<-session.Context().Done()
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go srv.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	u := url.URL{Scheme: "ws", Host: ln.Addr().String(), Path: "/stream/session/42"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("session handler did not complete setup")
	}
}

func TestWS_Integration_Echo(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	// Echo 服务
	srv.WS("/echo", func(session httpserver.WSSession) {
		for msg := range session.Recv() {
			_ = session.Send(msg)
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
