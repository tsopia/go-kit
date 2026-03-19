package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		config      SecurityHeadersConfig
		wantHeaders map[string]string
	}{
		{
			name:   "default basic headers",
			config: SecurityHeadersConfig{},
			wantHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":       "DENY",
				"Referrer-Policy":       "no-referrer",
			},
		},
		{
			name: "hsts header",
			config: SecurityHeadersConfig{
				HSTS: "max-age=31536000; includeSubDomains",
			},
			wantHeaders: map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			},
		},
		{
			name: "csp header",
			config: SecurityHeadersConfig{
				ContentSecurityPolicy: "default-src 'self'",
			},
			wantHeaders: map[string]string{
				"X-Content-Type-Options":  "nosniff",
				"Content-Security-Policy": "default-src 'self'",
			},
		},
		{
			name: "permissions policy header",
			config: SecurityHeadersConfig{
				PermissionsPolicy: "camera=(), microphone=()",
			},
			wantHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"Permissions-Policy":     "camera=(), microphone=()",
			},
		},
		{
			name: "all headers",
			config: SecurityHeadersConfig{
				HSTS:                  "max-age=31536000",
				ContentSecurityPolicy: "default-src 'self'",
				PermissionsPolicy:     "camera=()",
			},
			wantHeaders: map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"Referrer-Policy":           "no-referrer",
				"Strict-Transport-Security": "max-age=31536000",
				"Content-Security-Policy":   "default-src 'self'",
				"Permissions-Policy":        "camera=()",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(SecurityHeadersWithConfig(tc.config))
			engine.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			engine.ServeHTTP(w, req)

			for header, want := range tc.wantHeaders {
				got := w.Header().Get(header)
				if got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}

func TestSecurityHeadersDefaultCompatibility(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(SecurityHeaders())
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing X-Content-Type-Options")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing X-Frame-Options")
	}
	if w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("missing Referrer-Policy")
	}

	// HSTS/CSP/Permissions-Policy should NOT be set with default config
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Fatal("unexpected HSTS header with default config")
	}
}
