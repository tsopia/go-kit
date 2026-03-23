package httpserver_test

import (
	"context"
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
	srv.SSE("/events", func(ctx context.Context, send httpserver.SSESender) {
		send.Event("test", "data")
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
