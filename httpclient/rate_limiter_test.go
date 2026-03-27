package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRateLimiterIntegration 测试限流器集成
func TestRateLimiterIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	limiter := &mockRateLimiter{allow: false}

	client := NewClient(
		WithRateLimiter(limiter),
		WithTimeout(1*time.Second),
	)

	// 第一次调用，Allow 返回 false，触发 Wait
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, server.URL)
	if err == nil {
		t.Fatal("expected error due to rate limiter wait timeout")
	}

	if !limiter.waitCalled {
		t.Fatal("Wait should have been called")
	}
}

type mockRateLimiter struct {
	allow      bool
	waitCalled bool
}

func (m *mockRateLimiter) Allow() bool {
	return m.allow
}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	m.waitCalled = true
	// 模拟等待直到上下文取消
	<-ctx.Done()
	return ctx.Err()
}
