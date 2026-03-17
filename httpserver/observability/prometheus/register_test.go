package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterMetricsRoute(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	collector := NewCollector()
	engine := gin.New()
	engine.Use(collector.Middleware())
	engine.GET("/users", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	Register(engine, Config{
		Path:      "/metrics",
		Collector: collector,
	})

	userResp := httptest.NewRecorder()
	userReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	engine.ServeHTTP(userResp, userReq)

	metricsResp := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	engine.ServeHTTP(metricsResp, metricsReq)

	if metricsResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", metricsResp.Code, http.StatusOK)
	}

	body := metricsResp.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics body missing http_requests_total: %s", body)
	}
	if !strings.Contains(body, "route=\"/users\"") {
		t.Fatalf("metrics body missing route label: %s", body)
	}
	if !strings.Contains(body, "status=\"204\"") {
		t.Fatalf("metrics body missing status label: %s", body)
	}
}
