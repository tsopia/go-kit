package httpserver_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/httpserver/preset"
)

func TestPreset_StreamingSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &httpserver.Config{
		Port:           0, // 随机端口
		HandlerTimeout: 5 * time.Second,
		WriteTimeout:   10 * time.Second,
	}

	srv := preset.NewProductionServer(config)

	// 注册普通路由
	srv.GET("/api/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// 注册 SSE 路由
	srv.SSE("/events", func(stream httpserver.SSEStream) {
		_ = stream.Event("test", "data")
	})

	// 测试普通路由
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	srv.Engine().ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("ping status = %d", w1.Code)
	}

	// 测试 SSE 路由
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.Engine().ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("sse status = %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "event: test") {
		t.Errorf("sse response missing event")
	}
}

func TestSSE_Integration_WithHeartbeat(t *testing.T) {
	cfg := &httpserver.Config{Port: 18081}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	srv.SSE("/heartbeat-test", func(stream httpserver.SSEStream) {
		// 等待一段时间让心跳发送
		select {
		case <-stream.Context().Done():
			return
		case <-time.After(300 * time.Millisecond):
		}
	}, httpserver.WithHeartbeat(50*time.Millisecond))

	// 启动服务器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	go func() {
		_ = srv.Serve(ln)
	}()
	time.Sleep(100 * time.Millisecond)

	// 连接 SSE
	url := "http://" + ln.Addr().String() + "/stream/heartbeat-test"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 读取响应内容
	var body strings.Builder
	buf := make([]byte, 1024)
	deadline := time.AfterFunc(400*time.Millisecond, func() {
		_ = resp.Body.Close()
	})
	defer deadline.Stop()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	// 验证心跳存在
	content := body.String()
	if !strings.Contains(content, ": ping") {
		t.Errorf("expected ping in response, got: %s", content)
	}
}

func TestSSEHeartbeatDoesNotBlockFiniteStream(t *testing.T) {
	cfg := &httpserver.Config{Port: 0}
	srv := httpserver.NewServer(cfg)
	srv.SetGroups(
		srv.Engine().Group("/api"),
		srv.Engine().Group("/stream"),
	)

	srv.SSE("/finite", func(stream httpserver.SSEStream) {
		_ = stream.Event("done", "ok")
	}, httpserver.WithHeartbeat(50*time.Millisecond))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	go func() {
		_ = srv.Serve(ln)
	}()
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + ln.Addr().String() + "/stream/finite")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	done := make(chan error, 1)
	go func() {
		_, readErr := io.ReadAll(resp.Body)
		done <- readErr
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("response did not finish after handler returned")
	}
}
