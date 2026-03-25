package preset

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/httpserver"
	"github.com/tsopia/go-kit/utils"
)

func TestNewProductionServerAppliesExpectedMiddlewareBehavior(t *testing.T) {
	t.Parallel()

	srv := NewProductionServer(&httpserver.Config{
		EnableHealthCheck: false,
		ReadTimeout:       5 * time.Millisecond,
		HandlerTimeout:    5 * time.Millisecond,
	})

	srv.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	srv.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})
	srv.GET("/slow", func(c *gin.Context) {
		time.Sleep(20 * time.Millisecond)
		c.Status(http.StatusNoContent)
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		assert     func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "ok includes headers",
			path:       "/ok",
			wantStatus: http.StatusNoContent,
			assert: func(t *testing.T, resp *httptest.ResponseRecorder) {
				t.Helper()

				if resp.Header().Get(utils.RequestIDHeader) == "" {
					t.Fatal("missing request id header")
				}
				if resp.Header().Get(utils.TraceIDHeader) == "" {
					t.Fatal("missing trace id header")
				}
				if resp.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Fatalf("unexpected X-Content-Type-Options: %q", resp.Header().Get("X-Content-Type-Options"))
				}
			},
		},
		{
			name:       "panic is recovered",
			path:       "/panic",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "slow request times out",
			path:       "/slow",
			wantStatus: http.StatusGatewayTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			srv.Engine().ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}

			if tt.assert != nil {
				tt.assert(t, resp)
			}
		})
	}
}

func TestNewProductionServerLateUseAppliesToFutureHelperRoutes(t *testing.T) {
	t.Parallel()

	srv := NewProductionServer(&httpserver.Config{
		EnableHealthCheck: false,
		HandlerTimeout:    5 * time.Millisecond,
		WriteTimeout:      50 * time.Millisecond,
	})

	srv.Use(func(c *gin.Context) {
		c.Header("X-Late-Use", "1")
		c.Next()
	})

	srv.GET("/late", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	srv.SSE("/events", func(stream httpserver.SSEStream) {
		_ = stream.Event("ok", "1")
	})

	regularResp := httptest.NewRecorder()
	regularReq := httptest.NewRequest(http.MethodGet, "/late", nil)
	srv.Engine().ServeHTTP(regularResp, regularReq)

	if regularResp.Code != http.StatusNoContent {
		t.Fatalf("regular status = %d, want %d", regularResp.Code, http.StatusNoContent)
	}
	if got := regularResp.Header().Get("X-Late-Use"); got != "1" {
		t.Fatalf("regular X-Late-Use = %q, want %q", got, "1")
	}

	streamingResp := httptest.NewRecorder()
	streamingReq := httptest.NewRequest(http.MethodGet, "/events", nil)
	srv.Engine().ServeHTTP(streamingResp, streamingReq)

	if streamingResp.Code != http.StatusOK {
		t.Fatalf("streaming status = %d, want %d", streamingResp.Code, http.StatusOK)
	}
	if got := streamingResp.Header().Get("X-Late-Use"); got != "1" {
		t.Fatalf("streaming X-Late-Use = %q, want %q", got, "1")
	}
}
