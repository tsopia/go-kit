package httpserver

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
)

type countingObserver struct {
	mu          sync.Mutex
	connects    map[string]int
	disconnects map[string]int
}

func newCountingObserver() *countingObserver {
	return &countingObserver{connects: map[string]int{}, disconnects: map[string]int{}}
}

func (o *countingObserver) OnConnect(transport string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.connects[transport]++
}

func (o *countingObserver) OnDisconnect(transport string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.disconnects[transport]++
}

func (o *countingObserver) snapshot(transport string) (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.connects[transport], o.disconnects[transport]
}

// injectObserver 是一个把 observer 注入 request context 的测试中间件。
func injectObserver(obs httpmiddleware.StreamObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(
			httpmiddleware.WithStreamObserver(c.Request.Context(), obs),
		)
		c.Next()
	}
}

func startTestServer(t *testing.T, srv *Server) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(100 * time.Millisecond)
	return ln
}

func TestSSE_ObserverConnectDisconnect(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	obs := newCountingObserver()
	srv.Use(injectObserver(obs))
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	srv.SSE("/events", func(s SSEStream) {
		_ = s.Data("hi")
	})

	ln := startTestServer(t, srv)
	defer func() { _ = ln.Close() }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/stream/events")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()

	time.Sleep(150 * time.Millisecond)
	c, d := obs.snapshot("sse")
	if c != 1 || d != 1 {
		t.Errorf("sse connect/disconnect = %d/%d, want 1/1", c, d)
	}
}

func TestWS_ObserverConnectDisconnect(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	obs := newCountingObserver()
	srv.Use(injectObserver(obs))
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	srv.WS("/ws", func(session WSSession) {
		// 立即返回，触发断开
	})

	ln := startTestServer(t, srv)
	defer func() { _ = ln.Close() }()

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/stream/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	time.Sleep(200 * time.Millisecond)
	c, d := obs.snapshot("ws")
	if c != 1 || d != 1 {
		t.Errorf("ws connect/disconnect = %d/%d, want 1/1", c, d)
	}
}
