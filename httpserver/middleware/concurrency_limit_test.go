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
