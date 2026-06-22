package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"
)

func TestMiddleware_SkipsStreamingRequestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := NewCollector()
	engine := gin.New()
	engine.Use(collector.Middleware())
	engine.GET("/stream", func(c *gin.Context) {
		c.Set(utils.StreamingKey, "sse")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	if strings.Contains(collector.render(), `route="/stream"`) {
		t.Error("streaming request should not appear in request metrics")
	}
}

func TestMiddleware_InjectsStreamObserver(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := NewCollector()
	engine := gin.New()
	engine.Use(collector.Middleware())
	engine.GET("/stream", func(c *gin.Context) {
		obs, ok := httpmiddleware.StreamObserverFromContext(c.Request.Context())
		if !ok {
			t.Error("expected StreamObserver in context")
			return
		}
		obs.OnConnect("ws")
	})

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	body := collector.render()
	if !strings.Contains(body, `streaming_active_connections{transport="ws"} 1`) {
		t.Errorf("render missing active ws gauge:\n%s", body)
	}
}

func TestCollector_StreamGaugeIncDec(t *testing.T) {
	collector := NewCollector()
	collector.IncStream("sse")
	collector.IncStream("sse")
	collector.DecStream("sse")

	if !strings.Contains(collector.render(), `streaming_active_connections{transport="sse"} 1`) {
		t.Errorf("expected sse gauge = 1:\n%s", collector.render())
	}
}
