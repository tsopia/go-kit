package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTimeout(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Timeout(5 * time.Millisecond))
	started := make(chan struct{})
	contextDone := make(chan struct{})
	cleanupDone := make(chan struct{})
	allowReturn := make(chan struct{})
	returned := make(chan struct{})
	engine.GET("/slow", func(c *gin.Context) {
		defer close(returned)
		close(started)
		<-c.Request.Context().Done()
		close(contextDone)
		close(cleanupDone)
		<-allowReturn
	})

	w := &timeoutGuardRecorder{ResponseRecorder: httptest.NewRecorder(), returned: returned}
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handler did not start")
	}

	select {
	case <-contextDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handler did not observe context cancellation")
	}

	select {
	case <-cleanupDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handler did not finish cleanup")
	}

	close(allowReturn)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ServeHTTP did not return after the handler finished cleanup")
	}

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusGatewayTimeout)
	}
}

type timeoutGuardRecorder struct {
	*httptest.ResponseRecorder
	returned <-chan struct{}
}

func (r *timeoutGuardRecorder) WriteHeader(code int) {
	if code == http.StatusGatewayTimeout {
		select {
		case <-r.returned:
		default:
			panic("504 written before handler returned")
		}
	}
	r.ResponseRecorder.WriteHeader(code)
}
