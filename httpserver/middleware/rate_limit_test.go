package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		rps        float64
		burst      int
		requests   int
		wantPassed int
	}{
		{
			name:       "allows within rate",
			rps:        100,
			burst:      1,
			requests:   1,
			wantPassed: 1,
		},
		{
			name:       "burst allows short spike",
			rps:        1,
			burst:      5,
			requests:   5,
			wantPassed: 5,
		},
		{
			name:       "rejects when exceeded",
			rps:        1,
			burst:      1,
			requests:   3,
			wantPassed: 1,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(RateLimitWithConfig(RateLimitConfig{
				Rate:  tc.rps,
				Burst: tc.burst,
			}))
			engine.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			passed := 0
			for i := 0; i < tc.requests; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				engine.ServeHTTP(w, req)
				if w.Code == http.StatusOK {
					passed++
				}
			}

			if passed != tc.wantPassed {
				t.Fatalf("passed = %d, want %d", passed, tc.wantPassed)
			}
		})
	}
}

func TestRateLimitRejectsWithRetryAfter(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RateLimitWithConfig(RateLimitConfig{
		Rate:  1,
		Burst: 1,
	}))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Consume the token
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w1, req1)

	// Second request should be rejected
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}

	retryAfter := w2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestRateLimitCustomOnRejected(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RateLimitWithConfig(RateLimitConfig{
		Rate:  1,
		Burst: 1,
		OnRejected: func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limited",
			})
		},
	}))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Consume the token
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w1, req1)

	// Second request should be rejected with custom response
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}

	body := w2.Body.String()
	if body != "{\"error\":\"rate limited\"}" {
		t.Fatalf("body = %q, want custom response", body)
	}
}

func TestRateLimitNoopOnInvalidRate(t *testing.T) {
	testCases := []struct {
		name string
		rate float64
	}{
		{name: "zero", rate: 0},
		{name: "negative", rate: -1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(RateLimitWithConfig(RateLimitConfig{Rate: tc.rate}))
			engine.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			engine.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestRateLimitRetryAfterSetWhenOnRejectedDoesNotWrite(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RateLimitWithConfig(RateLimitConfig{
		Rate:  1,
		Burst: 1,
		OnRejected: func(c *gin.Context) {
			// 不写响应，应回退到默认 429 + Retry-After
		},
	}))
	engine.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 消耗 token
	w1 := httptest.NewRecorder()
	engine.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/test", nil))

	// 被拒绝的请求
	w2 := httptest.NewRecorder()
	engine.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/test", nil))

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header should be set when OnRejected does not write response")
	}
}
