package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestContextPropagation_VerifyContextReceived 验证服务端能收到 Context 中的值
func TestContextPropagation_VerifyContextReceived(t *testing.T) {
	var receivedContextValue string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求是否带有预期 header（通过中间件注入）
		receivedContextValue = r.Header.Get("X-Test-Value")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 创建带值的 Context
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")

	// 使用中间件将 Context 值注入请求头
	client := NewClient(
		WithMiddlewares(func(next http.RoundTripper) http.RoundTripper {
			return &testContextRoundTripper{
				next: next,
				key:  contextKey("test-key"),
			}
		}),
	)

	_, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedContextValue != "test-value" {
		t.Fatalf("expected context value 'test-value', got '%s'", receivedContextValue)
	}
}

type testContextRoundTripper struct {
	next http.RoundTripper
	key  interface{}
}

func (rt *testContextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if value := req.Context().Value(rt.key); value != nil {
		req.Header.Set("X-Test-Value", value.(string))
	}
	return rt.next.RoundTrip(req)
}

// TestContextCancellation_RequestTimeout 验证请求超时取消
func TestContextCancellation_RequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 延迟响应，超过客户端超时
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(WithTimeout(100 * time.Millisecond))

	ctx := context.Background()
	_, err := client.Get(ctx, server.URL)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestContextCancellation_ManualCancel 验证手动取消
func TestContextCancellation_ManualCancel(t *testing.T) {
	requestStarted := make(chan struct{})
	requestCancelled := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)

		// 等待 Context 取消或超时
		select {
		case <-r.Context().Done():
			close(requestCancelled)
		case <-time.After(5 * time.Second):
			t.Error("request was not cancelled in time")
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-requestStarted
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// 使用长超时，确保只有手动取消生效
	client := NewClient(WithTimeout(10 * time.Second))
	_, _ = client.Get(ctx, server.URL)

	select {
	case <-requestCancelled:
		// 成功取消
	case <-time.After(3 * time.Second):
		t.Fatal("request was not cancelled")
	}
}

// TestContextPropagation_GlobalFunctions 验证全局函数 Context 传播
func TestContextPropagation_GlobalFunctions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	defer ResetDefault()
	ResetDefault(WithBaseURL(server.URL))

	ctx := context.Background()

	// 测试所有全局函数
	tests := []struct {
		name string
		fn   func() (*Response, error)
	}{
		{
			name: "Get",
			fn:   func() (*Response, error) { return Get(ctx, "/test") },
		},
		{
			name: "Post",
			fn:   func() (*Response, error) { return Post(ctx, "/test", nil) },
		},
		{
			name: "Put",
			fn:   func() (*Response, error) { return Put(ctx, "/test", nil) },
		},
		{
			name: "Delete",
			fn:   func() (*Response, error) { return Delete(ctx, "/test") },
		},
		{
			name: "Patch",
			fn:   func() (*Response, error) { return Patch(ctx, "/test", nil) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.fn()
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
			if !resp.IsSuccess() {
				t.Fatalf("%s returned non-success status: %d", tt.name, resp.StatusCode)
			}
		})
	}
}

// TestContextDeadlineExceeded 验证 Context 超时错误
func TestContextDeadlineExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := NewClient()
	_, err := client.Get(ctx, server.URL)

	if err == nil {
		t.Fatal("expected error for timeout")
	}

	// 检查是否是 Context 超时错误
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", ctx.Err())
	}
}
