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
