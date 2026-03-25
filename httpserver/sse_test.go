package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
)

func TestSSESender_Event(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Event("update", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "event: update\ndata: {\"key\":\"value\"}\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Data(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Data("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "data: hello\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Comment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Comment("heartbeat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := ": heartbeat\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_DataEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Data("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "data: \n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_EventMultiline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Event("msg", "line1\nline2\nline3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "event: msg\ndata: line1\ndata: line2\ndata: line3\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestWithHeartbeat(t *testing.T) {
	opt := WithHeartbeat(30 * time.Second)
	if opt == nil {
		t.Fatal("WithHeartbeat should return non-nil option")
	}
}

func TestSSEOption_apply(t *testing.T) {
	cfg := &sseConfig{}
	opt := WithHeartbeat(30 * time.Second)
	opt.apply(cfg)
	if cfg.heartbeatInterval != 30*time.Second {
		t.Errorf("expected 30s, got %v", cfg.heartbeatInterval)
	}
}

func TestSSESender_heartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{
		ginCtx: c,
		config: &sseConfig{heartbeatInterval: 100 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	stopHeartbeat := sender.startHeartbeat(ctx)
	time.Sleep(300 * time.Millisecond) // 等待心跳发送
	stopHeartbeat()

	body := w.Body.String()
	if !strings.Contains(body, ": ping") {
		t.Errorf("expected ping in body, got: %s", body)
	}
}

func TestSSESender_startHeartbeat_noConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	stopHeartbeat := sender.startHeartbeat(context.Background())
	stopHeartbeat()
}

func TestSSESender_logDisconnect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		ctx       func() context.Context
		wantError bool
	}{
		{
			name: "nil context omits error field",
			ctx: func() context.Context {
				return nil
			},
			wantError: false,
		},
		{
			name: "cancelled context includes error field",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantError: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			logger := &capturedServerLogger{}
			req := httptest.NewRequest(http.MethodGet, "/events", nil)
			req = req.WithContext(httpmiddleware.WithStreamLogConfig(req.Context(), httpmiddleware.StreamLogConfig{
				Logger: logger.log,
			}))
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			sender := &sseSender{ginCtx: c, startedAt: time.Now()}
			sender.logDisconnect(tt.ctx())

			logEntry, ok := findServerLogByEvent(logger.snapshot(), "stream_disconnect")
			if !ok {
				t.Fatal("stream_disconnect log not found")
			}

			_, hasError := logEntry.fields["error"]
			if hasError != tt.wantError {
				t.Fatalf("has error field = %v, want %v", hasError, tt.wantError)
			}
		})
	}
}

// TestSSESender_ConcurrentWrite 验证并发写入不会导致 data race
func TestSSESender_ConcurrentWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{
		ginCtx: c,
		config: &sseConfig{heartbeatInterval: 50 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// 启动心跳
	stopHeartbeat := sender.startHeartbeat(ctx)

	// 同时从 handler 发送事件
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			_ = sender.Event("update", map[string]int{"count": i})
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// 等待所有 goroutine 完成
	wg.Wait()
	cancel() // 取消心跳
	stopHeartbeat()

	// 验证有内容写入
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestServer_SSE_withHeartbeat(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	server.SSE("/events", func(stream SSEStream) {
		<-stream.Context().Done()
	}, WithHeartbeat(100*time.Millisecond))

	// 验证路由已注册（streamingGroup 前缀为 /stream）
	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/stream/events" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SSE route not registered")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "single line no newline",
			input: "hello",
			want:  []string{"hello"},
		},
		{
			name:  "single line with trailing newline",
			input: "hello\n",
			want:  []string{"hello"},
		},
		{
			name:  "multiple lines",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "multiple lines with trailing newline",
			input: "line1\nline2\n",
			want:  []string{"line1", "line2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("splitLines(%q) = %v (len %d), want %v (len %d)", tc.input, got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}
