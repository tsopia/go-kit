package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecovery(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		handler    gin.HandlerFunc
		wantStatus int
	}{
		{
			name: "panic returns 500",
			handler: func(c *gin.Context) {
				panic("boom")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "no panic passes through",
			handler: func(c *gin.Context) {
				c.Status(http.StatusOK)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Recovery())
			engine.GET("/test", tc.handler)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			engine.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestRecoveryWithConfigLogsPanicValueAndStack(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	var captured map[string]any

	engine := gin.New()
	engine.Use(RecoveryWithConfig(RecoveryConfig{
		Logger: func(ctx context.Context, level string, event string, fields map[string]any) {
			mu.Lock()
			defer mu.Unlock()
			captured = fields
		},
	}))
	engine.GET("/panic", func(c *gin.Context) {
		panic("test-panic-value")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	mu.Lock()
	defer mu.Unlock()

	if captured == nil {
		t.Fatal("logger was not called")
	}

	panicValue, ok := captured["panic"]
	if !ok {
		t.Fatal("missing panic field")
	}
	if panicValue != "test-panic-value" {
		t.Fatalf("panic = %v, want %q", panicValue, "test-panic-value")
	}

	stack, ok := captured["stack"].(string)
	if !ok || stack == "" {
		t.Fatal("missing or empty stack field")
	}
	if !strings.Contains(stack, "goroutine") {
		t.Fatalf("stack does not look like a stack trace: %q", stack[:100])
	}

	if captured["method"] != http.MethodGet {
		t.Fatalf("method = %v, want %q", captured["method"], http.MethodGet)
	}
	if captured["path"] != "/panic" {
		t.Fatalf("path = %v, want %q", captured["path"], "/panic")
	}
}

func TestRecoveryWithConfigCallsOnPanic(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	var callbackRecovered any
	var callbackStack []byte

	engine := gin.New()
	engine.Use(RecoveryWithConfig(RecoveryConfig{
		OnPanic: func(c *gin.Context, recovered any, stack []byte) {
			mu.Lock()
			defer mu.Unlock()
			callbackRecovered = recovered
			callbackStack = stack
		},
	}))
	engine.GET("/panic", func(c *gin.Context) {
		panic("callback-test")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	mu.Lock()
	defer mu.Unlock()

	if callbackRecovered != "callback-test" {
		t.Fatalf("OnPanic recovered = %v, want %q", callbackRecovered, "callback-test")
	}
	if len(callbackStack) == 0 {
		t.Fatal("OnPanic stack is empty")
	}
}

func TestRecoveryDefaultConfigDoesNotPanic(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Recovery())
	engine.GET("/panic", func(c *gin.Context) {
		panic("default-logger-test")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	// 不应 panic
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecoveryWithConfigResponder(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RecoveryWithConfig(RecoveryConfig{
		Responder: func(c *gin.Context, recovered any) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "internal_error",
			})
		},
	}))
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(w.Body.String(), "internal_error") {
		t.Fatalf("body = %q, want internal_error", w.Body.String())
	}
}
