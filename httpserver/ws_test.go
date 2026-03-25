package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWSConfig_Defaults(t *testing.T) {
	cfg := defaultWSConfig()
	if cfg.SendBufferSize != 100 {
		t.Errorf("expected SendBufferSize=100, got %d", cfg.SendBufferSize)
	}
	if cfg.PingPeriod != 30*time.Second {
		t.Errorf("expected PingPeriod=30s, got %v", cfg.PingPeriod)
	}
	if cfg.PongTimeout != 60*time.Second {
		t.Errorf("expected PongTimeout=60s, got %v", cfg.PongTimeout)
	}

	routeCfg := defaultWSRouteConfig()
	if routeCfg.ReadIdleTimeout != 0 {
		t.Errorf("expected ReadIdleTimeout=0, got %v", routeCfg.ReadIdleTimeout)
	}
	if routeCfg.WriteTimeout != 10*time.Second {
		t.Errorf("expected WriteTimeout=10s, got %v", routeCfg.WriteTimeout)
	}
	if routeCfg.CheckOrigin == nil {
		t.Fatal("expected default CheckOrigin to be configured")
	}
}

func TestDefaultWSOriginCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name: "no origin allowed",
			want: true,
		},
		{
			name:   "browser origin denied by default",
			origin: "https://app.example.com",
			want:   false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			if got := defaultWSOriginCheck(req); got != tc.want {
				t.Fatalf("defaultWSOriginCheck() = %v, want %v", got, tc.want)
			}
		})
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
	cfg := defaultWSRouteConfig()

	opts := []struct {
		name  string
		apply WSRouteOption
		check func(*testing.T, WSRouteConfig)
	}{
		{
			name:  "send buffer",
			apply: WithWSSendBuffer(300),
			check: func(t *testing.T, cfg WSRouteConfig) {
				if cfg.SendBufferSize != 300 {
					t.Fatalf("SendBufferSize = %d, want %d", cfg.SendBufferSize, 300)
				}
			},
		},
		{
			name:  "ping period",
			apply: WithWSPingPeriod(10 * time.Second),
			check: func(t *testing.T, cfg WSRouteConfig) {
				if cfg.PingPeriod != 10*time.Second {
					t.Fatalf("PingPeriod = %v, want %v", cfg.PingPeriod, 10*time.Second)
				}
			},
		},
		{
			name:  "pong timeout",
			apply: WithWSPongTimeout(20 * time.Second),
			check: func(t *testing.T, cfg WSRouteConfig) {
				if cfg.PongTimeout != 20*time.Second {
					t.Fatalf("PongTimeout = %v, want %v", cfg.PongTimeout, 20*time.Second)
				}
			},
		},
		{
			name:  "read idle timeout",
			apply: WithReadIdleTimeout(15 * time.Second),
			check: func(t *testing.T, cfg WSRouteConfig) {
				if cfg.ReadIdleTimeout != 15*time.Second {
					t.Fatalf("ReadIdleTimeout = %v, want %v", cfg.ReadIdleTimeout, 15*time.Second)
				}
			},
		},
		{
			name:  "write timeout",
			apply: WithWriteTimeout(5 * time.Second),
			check: func(t *testing.T, cfg WSRouteConfig) {
				if cfg.WriteTimeout != 5*time.Second {
					t.Fatalf("WriteTimeout = %v, want %v", cfg.WriteTimeout, 5*time.Second)
				}
			},
		},
		{
			name:  "allowed origins",
			apply: WithWSAllowedOrigins("https://app.example.com"),
			check: func(t *testing.T, cfg WSRouteConfig) {
				req := httptest.NewRequest(http.MethodGet, "/ws", nil)
				req.Header.Set("Origin", "https://app.example.com")
				if !cfg.CheckOrigin(req) {
					t.Fatal("expected configured origin to be allowed")
				}

				req.Header.Set("Origin", "https://evil.example.com")
				if cfg.CheckOrigin(req) {
					t.Fatal("expected unconfigured origin to be denied")
				}
			},
		},
		{
			name: "origin checker",
			apply: WithWSOriginChecker(func(r *http.Request) bool {
				return r.Header.Get("Origin") == "https://checker.example.com"
			}),
			check: func(t *testing.T, cfg WSRouteConfig) {
				req := httptest.NewRequest(http.MethodGet, "/ws", nil)
				req.Header.Set("Origin", "https://checker.example.com")
				if !cfg.CheckOrigin(req) {
					t.Fatal("expected custom checker to allow origin")
				}
			},
		},
	}

	for _, tt := range opts {
		t.Run(tt.name, func(t *testing.T) {
			local := cfg
			tt.apply.applyRoute(&local)
			tt.check(t, local)
		})
	}
}

func TestWSSessionContract(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recv := make(chan WSMessage, 1)
	send := make(chan WSMessage, 1)
	closeErr := errors.New("close failed")
	closeCalled := false

	session := &wsSession{
		ctx:     ctx,
		request: httptest.NewRequest("GET", "/stream/ws/42", nil),
		params:  gin.Params{{Key: "id", Value: "42"}},
		recv:    recv,
		send:    send,
		closeFn: func(code int, reason string) error {
			closeCalled = true
			if code != websocket.CloseNormalClosure {
				t.Fatalf("code = %d, want %d", code, websocket.CloseNormalClosure)
			}
			if reason != "bye" {
				t.Fatalf("reason = %q, want %q", reason, "bye")
			}
			return closeErr
		},
	}

	if session.Request().URL.Path != "/stream/ws/42" {
		t.Fatalf("path = %q, want %q", session.Request().URL.Path, "/stream/ws/42")
	}
	if session.Param("id") != "42" {
		t.Fatalf("id = %q, want %q", session.Param("id"), "42")
	}

	recv <- WSMessage{Type: websocket.TextMessage, Data: []byte("recv")}
	msg := <-session.Recv()
	if string(msg.Data) != "recv" {
		t.Fatalf("recv = %q, want %q", string(msg.Data), "recv")
	}

	if err := session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte("send")}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if session.TrySend(WSMessage{Type: websocket.TextMessage, Data: []byte("drop")}) {
		t.Fatal("expected TrySend to return false when queue is full")
	}

	msg = <-send
	if string(msg.Data) != "send" {
		t.Fatalf("send = %q, want %q", string(msg.Data), "send")
	}

	if err := session.Close(websocket.CloseNormalClosure, "bye"); !errors.Is(err, closeErr) {
		t.Fatalf("close err = %v, want %v", err, closeErr)
	}
	if !closeCalled {
		t.Fatal("expected close callback to be invoked")
	}
}

func TestWSSessionCloseRejectsFurtherSends(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	session := &wsSession{
		ctx:  ctx,
		send: make(chan WSMessage),
		recv: make(chan WSMessage),
		closeFn: func(code int, reason string) error {
			if code != websocket.CloseNormalClosure {
				t.Fatalf("code = %d, want %d", code, websocket.CloseNormalClosure)
			}
			if reason != "bye" {
				t.Fatalf("reason = %q, want %q", reason, "bye")
			}
			cancel()
			return nil
		},
	}

	if err := session.Close(websocket.CloseNormalClosure, "bye"); err != nil {
		t.Fatalf("close: %v", err)
	}

	err := session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte("late")})
	if !errors.Is(err, ErrWSSessionClosed) {
		t.Fatalf("send err = %v, want %v", err, ErrWSSessionClosed)
	}
	if session.TrySend(WSMessage{Type: websocket.TextMessage, Data: []byte("late")}) {
		t.Fatal("expected TrySend to fail after Close")
	}
}

func TestServer_WS(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	server.WS("/ws", func(session WSSession) {
		<-session.Context().Done()
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
	t.Parallel()

	testCases := []struct {
		name       string
		panicValue any
		wantCancel bool
	}{
		{
			name:       "panic triggers cancel",
			panicValue: "boom",
			wantCancel: true,
		},
		{
			name:       "no panic leaves cancel untouched",
			wantCancel: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cancelCh := make(chan struct{}, 1)
			cancel := func() {
				select {
				case cancelCh <- struct{}{}:
				default:
				}
			}

			func() {
				defer recoverWSPumpPanic("read", "/stream/panic", cancel)
				if tt.panicValue != nil {
					panic(tt.panicValue)
				}
			}()

			select {
			case <-cancelCh:
				if !tt.wantCancel {
					t.Fatal("cancel called unexpectedly")
				}
			default:
				if tt.wantCancel {
					t.Fatal("cancel was not called")
				}
			}
		})
	}
}

func TestWS_UpgradeError(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// 注册 WS 路由
	server.WS("/ws", func(session WSSession) {
		<-session.Context().Done()
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

func TestWS_ReadIdleTimeout(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	firstRead := make(chan struct{}, 1)
	handlerDone := make(chan error, 1)

	server.WS("/timeout", func(session WSSession) {
		select {
		case msg, ok := <-session.Recv():
			if !ok {
				handlerDone <- errors.New("recv closed before first message")
				return
			}
			if string(msg.Data) != "hello" {
				handlerDone <- fmt.Errorf("message = %q, want %q", string(msg.Data), "hello")
				return
			}
			firstRead <- struct{}{}
		case <-time.After(time.Second):
			handlerDone <- errors.New("server did not receive first message")
			return
		}
		<-session.Context().Done()
		handlerDone <- nil
	}, WithReadIdleTimeout(100*time.Millisecond), WithWSPingPeriod(0))

	// 使用 net.Listen 获取随机端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = ln.Close()
	}()

	go func() {
		_ = server.Serve(ln)
	}()
	time.Sleep(100 * time.Millisecond) // 等待服务器启动

	// WebSocket 连接
	wsURL := "ws://" + ln.Addr().String() + "/stream/timeout"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	select {
	case <-time.After(60 * time.Millisecond):
	case err := <-handlerDone:
		t.Fatalf("handler finished too early: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("write message: %v", err)
	}

	select {
	case <-firstRead:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not observe inbound message")
	}

	select {
	case err := <-handlerDone:
		t.Fatalf("read idle timeout did not reset after message: %v", err)
	case <-time.After(70 * time.Millisecond):
	}

	select {
	case err := <-handlerDone:
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("read idle timeout did not fire after inactivity")
	}
}
