package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

func TestRealIP_NoTrustedCIDRs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIP())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 没有配置信任 CIDR，应该使用 RemoteAddr
	if w.Body.String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", w.Body.String())
	}
}

func TestRealIP_WithTrustedCIDRSingleIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.1, 192.168.1.100")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 1.2.3.4 不在信任 CIDR 中，应该返回它
	if w.Body.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", w.Body.String())
	}
}

func TestRealIP_UntrustedRemoteAddr(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var untrustedCalled bool
	var untrustedAddr string

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
		OnUntrusted: func(c *gin.Context, remoteAddr string) {
			untrustedCalled = true
			untrustedAddr = remoteAddr
		},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	// RemoteAddr 不在信任 CIDR 中
	req.RemoteAddr = "10.0.0.1:12345"

	router.ServeHTTP(w, req)

	// RemoteAddr 不在信任列表，应该使用 RemoteAddr
	if w.Body.String() != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", w.Body.String())
	}
	if !untrustedCalled {
		t.Error("OnUntrusted callback was not called")
	}
	if untrustedAddr != "10.0.0.1" {
		t.Errorf("OnUntrusted addr = %s, want 10.0.0.1", untrustedAddr)
	}
}

func TestRealIP_AllProxiesTrusted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16", "10.0.0.0/8"},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// 所有 IP 都在信任 CIDR 中
	req.Header.Set("X-Forwarded-For", "10.1.2.3, 192.168.1.100")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 所有 IP 都信任，返回最后一个
	if w.Body.String() != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s", w.Body.String())
	}
}

func TestRealIP_XRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// 没有 X-Forwarded-For，但有 X-Real-IP
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 应该使用 X-Real-IP
	if w.Body.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", w.Body.String())
	}
}

func TestRealIP_XRealIPInTrustedCIDR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// X-Real-IP 在信任 CIDR 中（可能是配置错误），应该忽略
	req.Header.Set("X-Real-IP", "192.168.5.5")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// X-Real-IP 在信任 CIDR 中，应该使用 RemoteAddr
	if w.Body.String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", w.Body.String())
	}
}

func TestRealIP_NoProxyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// 没有代理头
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 没有代理头，应该使用 RemoteAddr
	if w.Body.String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", w.Body.String())
	}
}

func TestRealIP_ContextSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		// 验证 context 中是否设置了 client IP
		clientIP := utils.ClientIPFromContext(c.Request.Context())
		c.String(200, clientIP)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	if w.Body.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4 from context, got %s", w.Body.String())
	}
}

func TestRealIP_GinContextSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16"},
	}))
	router.GET("/test", func(c *gin.Context) {
		// 验证 gin context 中是否设置了 client IP
		if clientIP, exists := c.Get(utils.ClientIPKey); exists {
			c.String(200, clientIP.(string))
		} else {
			c.String(200, "not found")
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	if w.Body.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4 from gin context, got %s", w.Body.String())
	}
}

func TestRealIP_InvalidCIDR(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 包含无效 CIDR，应该被跳过
	router := gin.New()
	router.Use(RealIPWithConfig(RealIPConfig{
		TrustedCIDRs: []string{"192.168.0.0/16", "invalid-cidr", ""},
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	if w.Body.String() != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", w.Body.String())
	}
}

func TestClientIPFromContext_WithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 不使用 RealIP 中间件
	router.GET("/test", func(c *gin.Context) {
		c.String(200, clientIPFromContext(c))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	router.ServeHTTP(w, req)

	// 应该降级到 Gin 的 ClientIP()
	expected := "192.168.1.1"
	if w.Body.String() != expected {
		t.Errorf("expected %s from ClientIP(), got %s", expected, w.Body.String())
	}
}

func TestClientIPFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		remoteAddr string
		contextIP  string
		expected   string
	}{
		{
			name:       "with context IP",
			remoteAddr: "192.168.1.1:12345",
			contextIP:  "1.2.3.4",
			expected:   "1.2.3.4",
		},
		{
			name:       "without context IP",
			remoteAddr: "192.168.1.1:12345",
			contextIP:  "",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			if tt.contextIP != "" {
				req = req.WithContext(utils.WithClientIP(req.Context(), tt.contextIP))
			}

			got := ClientIPFromRequest(req)
			if got != tt.expected {
				t.Errorf("ClientIPFromRequest() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestParseCIDRs(t *testing.T) {
	tests := []struct {
		name     string
		cidrs    []string
		expected int
	}{
		{
			name:     "valid CIDRs",
			cidrs:    []string{"192.168.0.0/16", "10.0.0.0/8"},
			expected: 2,
		},
		{
			name:     "empty list",
			cidrs:    []string{},
			expected: 0,
		},
		{
			name:     "nil list",
			cidrs:    nil,
			expected: 0,
		},
		{
			name:     "with invalid CIDRs",
			cidrs:    []string{"192.168.0.0/16", "invalid", ""},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets := parseCIDRs(tt.cidrs)
			if len(nets) != tt.expected {
				t.Errorf("parseCIDRs() returned %d CIDRs, expected %d", len(nets), tt.expected)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "normal",
			input:    "a, b, c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    "  a  ,  b  ,  c  ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty parts",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "b"},
		},
		{
			name:     "only spaces",
			input:    "   ,   ,   ",
			sep:      ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input, tt.sep)
			if len(got) != len(tt.expected) {
				t.Errorf("splitAndTrim() = %v, expected %v", got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitAndTrim()[%d] = %s, expected %s", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
