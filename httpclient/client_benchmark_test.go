package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type benchRoundTripper func(*http.Request) (*http.Response, error)

func (f benchRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type nopLogger struct{}

func (nopLogger) Debug(context.Context, string, ...interface{}) {}
func (nopLogger) Info(context.Context, string, ...interface{})  {}
func (nopLogger) Warn(context.Context, string, ...interface{})  {}
func (nopLogger) Error(context.Context, string, ...interface{}) {}

func benchmarkClient(b *testing.B, opts ClientOptions) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	if opts.Logger == nil {
		opts.Logger = nopLogger{}
	}

	client := NewClientWithOptions(opts)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := client.NewRequest(http.MethodGet, server.URL).Do()
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		// The client reads and closes the response body internally; ensure we
		// don't retain references across iterations.
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
}

func BenchmarkClientBaseline(b *testing.B) {
	benchmarkClient(b, ClientOptions{Timeout: 5 * time.Second})
}

func BenchmarkClientWithMiddleware(b *testing.B) {
	benchmarkClient(b, ClientOptions{
		Timeout: 5 * time.Second,
		Middlewares: []Middleware{
			func(next http.RoundTripper) http.RoundTripper {
				return benchRoundTripper(func(req *http.Request) (*http.Response, error) {
					return next.RoundTrip(req)
				})
			},
		},
	})
}

func BenchmarkClientWithRetryAndBreaker(b *testing.B) {
	benchmarkClient(b, ClientOptions{
		Timeout:        5 * time.Second,
		Retry:          &RetryConfig{MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1},
		CircuitBreaker: &CircuitBreakerConfig{FailureThreshold: 3, SuccessThreshold: 1, Timeout: time.Second},
	})
}
