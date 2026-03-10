package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type testRouteModule struct{}

func (testRouteModule) RegisterRoutes(r gin.IRoutes) {
	r.GET("/module/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func TestRegisterModules(t *testing.T) {
	srv := NewServer(nil)
	srv.RegisterModules(testRouteModule{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/module/ping", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNewServerWithModulesOption(t *testing.T) {
	srv := NewServer(nil, WithModules(testRouteModule{}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/module/ping", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
