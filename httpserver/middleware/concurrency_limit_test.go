package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestConcurrencyLimitAllowsWithinLimit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name  string
		limit int
	}{
		{name: "single slot", limit: 1},
		{name: "multiple slots", limit: 2},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(ConcurrencyLimit(tc.limit))
			engine.GET("/ok", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ok", nil)
			engine.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
			}
		})
	}
}

func TestConcurrencyLimitRejectsWhenFull(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name  string
		limit int
	}{
		{name: "single slot", limit: 1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			release := make(chan struct{})
			entered := make(chan struct{}, 1)

			engine := gin.New()
			engine.Use(ConcurrencyLimit(tc.limit))
			engine.GET("/block", func(c *gin.Context) {
				select {
				case entered <- struct{}{}:
				default:
				}

				<-release
				c.Status(http.StatusOK)
			})
			engine.GET("/probe", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			firstDone := make(chan struct{})
			firstRecorder := httptest.NewRecorder()
			firstReq := httptest.NewRequest(http.MethodGet, "/block", nil)
			go func() {
				engine.ServeHTTP(firstRecorder, firstReq)
				close(firstDone)
			}()

			select {
			case <-entered:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("first request did not enter handler")
			}

			secondDone := make(chan struct{})
			secondRecorder := httptest.NewRecorder()
			secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
			go func() {
				engine.ServeHTTP(secondRecorder, secondReq)
				close(secondDone)
			}()

			select {
			case <-secondDone:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("second request was not rejected immediately")
			}

			if secondRecorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusServiceUnavailable)
			}

			close(release)

			select {
			case <-firstDone:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("first request did not finish")
			}

			if firstRecorder.Code != http.StatusOK {
				t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusOK)
			}
		})
	}
}

func TestConcurrencyLimitDefaultRejection(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)

	engine := gin.New()
	engine.Use(ConcurrencyLimit(1))
	engine.GET("/block", func(c *gin.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}

		<-release
		c.Status(http.StatusOK)
	})
	engine.GET("/probe", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	firstDone := make(chan struct{})
	go func() {
		firstRecorder := httptest.NewRecorder()
		firstReq := httptest.NewRequest(http.MethodGet, "/block", nil)
		engine.ServeHTTP(firstRecorder, firstReq)
		close(firstDone)
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not enter handler")
	}

	secondDone := make(chan struct{})
	secondRecorder := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
	go func() {
		engine.ServeHTTP(secondRecorder, secondReq)
		close(secondDone)
	}()

	select {
	case <-secondDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("second request was not rejected immediately")
	}

	if secondRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", secondRecorder.Code, http.StatusServiceUnavailable)
	}

	if secondRecorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty body", secondRecorder.Body.String())
	}

	close(release)

	select {
	case <-firstDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not finish")
	}
}

func TestConcurrencyLimitCustomRejection(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)

	engine := gin.New()
	engine.Use(ConcurrencyLimitWithConfig(ConcurrencyLimitConfig{
		Limit: 1,
		OnRejected: func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "busy",
			})
		},
	}))
	engine.GET("/block", func(c *gin.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}

		<-release
		c.Status(http.StatusOK)
	})
	engine.GET("/probe", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	firstDone := make(chan struct{})
	go func() {
		firstRecorder := httptest.NewRecorder()
		firstReq := httptest.NewRequest(http.MethodGet, "/block", nil)
		engine.ServeHTTP(firstRecorder, firstReq)
		close(firstDone)
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not enter handler")
	}

	secondDone := make(chan struct{})
	secondRecorder := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
	go func() {
		engine.ServeHTTP(secondRecorder, secondReq)
		close(secondDone)
	}()

	select {
	case <-secondDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("second request was not rejected immediately")
	}

	if secondRecorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", secondRecorder.Code, http.StatusTooManyRequests)
	}

	if got := secondRecorder.Body.String(); got != "{\"error\":\"busy\"}" {
		t.Fatalf("body = %q, want %q", got, "{\"error\":\"busy\"}")
	}

	close(release)

	select {
	case <-firstDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not finish")
	}
}

func TestConcurrencyLimitOnRejectedFallsBackToDefaultWhenUnwritten(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	rejectedCalled := make(chan struct{}, 1)

	engine := gin.New()
	engine.Use(ConcurrencyLimitWithConfig(ConcurrencyLimitConfig{
		Limit: 1,
		OnRejected: func(c *gin.Context) {
			select {
			case rejectedCalled <- struct{}{}:
			default:
			}
		},
	}))
	engine.GET("/block", func(c *gin.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}

		<-release
		c.Status(http.StatusOK)
	})
	engine.GET("/probe", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	firstDone := make(chan struct{})
	go func() {
		firstRecorder := httptest.NewRecorder()
		firstReq := httptest.NewRequest(http.MethodGet, "/block", nil)
		engine.ServeHTTP(firstRecorder, firstReq)
		close(firstDone)
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not enter handler")
	}

	secondDone := make(chan struct{})
	secondRecorder := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
	go func() {
		engine.ServeHTTP(secondRecorder, secondReq)
		close(secondDone)
	}()

	select {
	case <-rejectedCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("OnRejected was not called")
	}

	select {
	case <-secondDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("second request was not rejected immediately")
	}

	if secondRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", secondRecorder.Code, http.StatusServiceUnavailable)
	}

	if secondRecorder.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty body", secondRecorder.Body.String())
	}

	close(release)

	select {
	case <-firstDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not finish")
	}
}

func TestConcurrencyLimitPanicsOnInvalidLimit(t *testing.T) {
	testCases := []struct {
		name  string
		limit int
	}{
		{name: "zero", limit: 0},
		{name: "negative", limit: -1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()

			_ = ConcurrencyLimitWithConfig(ConcurrencyLimitConfig{
				Limit: tc.limit,
			})
		})
	}
}

func TestConcurrencyLimitReleasesSlotAfterCompletion(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name  string
		limit int
	}{
		{name: "single slot", limit: 1},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			release := make(chan struct{})
			entered := make(chan struct{}, 1)

			engine := gin.New()
			engine.Use(ConcurrencyLimit(tc.limit))
			engine.GET("/block", func(c *gin.Context) {
				select {
				case entered <- struct{}{}:
				default:
				}

				<-release
				c.Status(http.StatusOK)
			})
			engine.GET("/probe", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			firstRecorder := httptest.NewRecorder()
			firstReq := httptest.NewRequest(http.MethodGet, "/block", nil)
			firstDone := make(chan struct{})
			go func() {
				engine.ServeHTTP(firstRecorder, firstReq)
				close(firstDone)
			}()

			select {
			case <-entered:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("first request did not enter handler")
			}

			close(release)

			select {
			case <-firstDone:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("first request did not finish")
			}

			if firstRecorder.Code != http.StatusOK {
				t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusOK)
			}

			secondRecorder := httptest.NewRecorder()
			secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
			engine.ServeHTTP(secondRecorder, secondReq)

			if secondRecorder.Code != http.StatusOK {
				t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusOK)
			}
		})
	}
}

func TestConcurrencyLimitReleasesSlotAfterPanic(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	entered := make(chan struct{}, 1)

	engine := gin.New()
	engine.Use(Recovery())
	engine.Use(ConcurrencyLimit(1))
	engine.GET("/panic", func(c *gin.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}

		panic("boom")
	})
	engine.GET("/probe", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	firstDone := make(chan struct{})
	firstRecorder := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/panic", nil)
	go func() {
		engine.ServeHTTP(firstRecorder, firstReq)
		close(firstDone)
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not enter handler")
	}

	select {
	case <-firstDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not finish")
	}

	if firstRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusInternalServerError)
	}

	secondRecorder := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
	engine.ServeHTTP(secondRecorder, secondReq)

	if secondRecorder.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusNoContent)
	}
}

func TestConcurrencyLimitReleasesSlotAfterTimeout(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	finished := make(chan struct{})

	engine := gin.New()
	engine.Use(Recovery())
	engine.Use(ConcurrencyLimit(1))
	engine.Use(Timeout(5 * time.Millisecond))
	engine.GET("/slow", func(c *gin.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}

		<-release
		close(finished)
	})
	engine.GET("/probe", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	firstDone := make(chan struct{})
	firstRecorder := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/slow", nil)
	go func() {
		engine.ServeHTTP(firstRecorder, firstReq)
		close(firstDone)
	}()

	select {
	case <-entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not enter handler")
	}

	select {
	case <-firstDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first request did not finish")
	}

	if firstRecorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusGatewayTimeout)
	}

	secondRecorder := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/probe", nil)
	engine.ServeHTTP(secondRecorder, secondReq)

	if secondRecorder.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusNoContent)
	}

	close(release)

	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("first handler did not finish")
	}
}
