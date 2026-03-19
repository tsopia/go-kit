package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		config        CORSConfig
		origin        string
		method        string
		wantStatus    int
		wantOrigin    string
		wantMaxAge    string
		wantCredHeader string
		wantNoOrigin  bool
	}{
		{
			name:       "default allows all origins",
			config:     CORSConfig{},
			origin:     "https://example.com",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantOrigin: "*",
		},
		{
			name:       "single allowed origin",
			config:     CORSConfig{AllowOrigins: []string{"https://example.com"}},
			origin:     "https://example.com",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantOrigin: "https://example.com",
		},
		{
			name:       "multiple allowed origins matches first",
			config:     CORSConfig{AllowOrigins: []string{"https://a.com", "https://b.com"}},
			origin:     "https://b.com",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantOrigin: "https://b.com",
		},
		{
			name:         "non matching origin",
			config:       CORSConfig{AllowOrigins: []string{"https://allowed.com"}},
			origin:       "https://evil.com",
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantNoOrigin: true,
		},
		{
			name: "dynamic origin func",
			config: CORSConfig{
				AllowOriginFunc: func(origin string) bool {
					return origin == "https://dynamic.com"
				},
			},
			origin:     "https://dynamic.com",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantOrigin: "https://dynamic.com",
		},
		{
			name: "credentials with specific origin",
			config: CORSConfig{
				AllowOrigins:     []string{"https://trusted.com"},
				AllowCredentials: true,
			},
			origin:         "https://trusted.com",
			method:         http.MethodGet,
			wantStatus:     http.StatusOK,
			wantOrigin:     "https://trusted.com",
			wantCredHeader: "true",
		},
		{
			name: "credentials strips wildcard",
			config: CORSConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
			},
			origin:       "https://any.com",
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantNoOrigin: true,
		},
		{
			name: "preflight returns no content",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com"},
			},
			origin:     "https://example.com",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
			wantOrigin: "https://example.com",
		},
		{
			name: "preflight max age",
			config: CORSConfig{
				AllowOrigins: []string{"https://example.com"},
				MaxAge:       3600 * time.Second,
			},
			origin:     "https://example.com",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
			wantOrigin: "https://example.com",
			wantMaxAge: "3600",
		},
		{
			name:         "no origin header skips cors",
			config:       CORSConfig{},
			origin:       "",
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantNoOrigin: true,
		},
		{
			name: "AllowOriginFunc rejects - does NOT fallback to AllowOrigins",
			config: CORSConfig{
				AllowOrigins: []string{"https://allowed.com"},
				AllowOriginFunc: func(origin string) bool {
					return false // 明确拒绝所有
				},
			},
			origin:       "https://allowed.com", // 在 AllowOrigins 里，但 func 拒绝
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantNoOrigin: true, // func 拒绝后不应 fallback
		},
		{
			name: "AllowOriginFunc rejects non-matching origin - no fallback to wildcard",
			config: CORSConfig{
				// AllowOrigins 未设置时 normalizeCORSConfig 不填充默认值（因为 AllowOriginFunc != nil）
				AllowOriginFunc: func(origin string) bool {
					return origin == "https://explicit.com"
				},
			},
			origin:       "https://other.com",
			method:       http.MethodGet,
			wantStatus:   http.StatusOK,
			wantNoOrigin: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(CORS(tc.config))
			engine.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			// 对 OPTIONS 方法也需要注册
			engine.OPTIONS("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/test", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			engine.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tc.wantNoOrigin {
				if gotOrigin != "" {
					t.Fatalf("expected no Access-Control-Allow-Origin header, got %q", gotOrigin)
				}
			} else {
				if gotOrigin != tc.wantOrigin {
					t.Fatalf("Access-Control-Allow-Origin = %q, want %q", gotOrigin, tc.wantOrigin)
				}
			}

			if tc.wantMaxAge != "" {
				gotMaxAge := w.Header().Get("Access-Control-Max-Age")
				if gotMaxAge != tc.wantMaxAge {
					t.Fatalf("Access-Control-Max-Age = %q, want %q", gotMaxAge, tc.wantMaxAge)
				}
			}

			if tc.wantCredHeader != "" {
				gotCred := w.Header().Get("Access-Control-Allow-Credentials")
				if gotCred != tc.wantCredHeader {
					t.Fatalf("Access-Control-Allow-Credentials = %q, want %q", gotCred, tc.wantCredHeader)
				}
			}
		})
	}
}
