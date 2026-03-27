package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetryBackoffFactor 验证指数退避逻辑
func TestRetryBackoffFactor(t *testing.T) {
	var attempts int32
	var lastTime time.Time
	var delays []time.Duration
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := atomic.AddInt32(&attempts, 1)

		mu.Lock()
		now := time.Now()
		if currentAttempt > 1 {
			delays = append(delays, now.Sub(lastTime))
		}
		lastTime = now
		mu.Unlock()

		if currentAttempt <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		WithTimeout(5*time.Second),
		WithRetry(&RetryConfig{
			MaxRetries:      3,
			InitialDelay:    50 * time.Millisecond,
			MaxDelay:        500 * time.Millisecond,
			BackoffFactor:   2.0,
			RetryableStatus: []int{http.StatusInternalServerError},
		}),
	)

	_, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected success on 4th try, got error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(delays) != 3 {
		t.Fatalf("expected 3 retries (delays), got %d", len(delays))
	}

	// 第一次重试休眠约 50ms，第二次约 100ms，第三次约 200ms
	// 允许 30% 误差由于 Go 调度器精度
	expectedDelays := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	for i, d := range delays {
		expected := expectedDelays[i]
		min := time.Duration(float64(expected) * 0.7)
		if d < min {
			t.Errorf("delay %d too short: got %v, want at least %v", i, d, min)
		}
	}
}

// TestRetryCancellation 验证重试过程中的上下文取消逻辑
func TestRetryCancellation(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(
		WithRetry(&RetryConfig{
			MaxRetries:      5,
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        500 * time.Millisecond,
			BackoffFactor:   1.0,
			RetryableStatus: []int{http.StatusInternalServerError},
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())

	// 只允许发一次请求就取消 context
	go func() {
		time.Sleep(50 * time.Millisecond) // 等待第一次请求发出并在 sleep 阻塞阶段
		cancel()
	}()

	start := time.Now()
	_, err := client.Get(ctx, server.URL)
	duration := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts > 2 {
		t.Fatalf("expected context to stop retries early, but made %d requests", finalAttempts)
	}

	if duration > 300*time.Millisecond {
		t.Fatalf("context cancellation did not interrupt sleep fast enough, took %v", duration)
	}
}

func TestRetryClosesResponseBodyBeforeRetry(t *testing.T) {
	bodies := []*trackedBody{
		newTrackedBody("retry-1"),
		newTrackedBody("retry-2"),
		newTrackedBody("ok"),
	}
	statuses := []int{
		http.StatusServiceUnavailable,
		http.StatusServiceUnavailable,
		http.StatusOK,
	}

	var calls int32
	client := NewClient(
		WithHTTPClient(&http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				call := int(atomic.AddInt32(&calls, 1) - 1)
				if call >= len(bodies) {
					return nil, errors.New("unexpected extra retry")
				}

				return &http.Response{
					StatusCode: statuses[call],
					Status:     http.StatusText(statuses[call]),
					Header:     make(http.Header),
					Body:       bodies[call],
					Request:    req,
				}, nil
			}),
		}),
		WithRetry(&RetryConfig{
			MaxRetries:      2,
			InitialDelay:    time.Millisecond,
			MaxDelay:        time.Millisecond,
			BackoffFactor:   1,
			RetryableStatus: []int{http.StatusServiceUnavailable},
		}),
	)

	resp, err := client.Get(context.Background(), "http://example.com/test")
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected final status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	for i := 0; i < len(bodies); i++ {
		if got := atomic.LoadInt32(&bodies[i].closed); got != 1 {
			t.Fatalf("expected body %d to be closed once, got %d", i, got)
		}
	}
}

type trackedBody struct {
	data   []byte
	closed int32
}

func newTrackedBody(content string) *trackedBody {
	return &trackedBody{data: []byte(content)}
}

func (b *trackedBody) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}

	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *trackedBody) Close() error {
	atomic.AddInt32(&b.closed, 1)
	return nil
}
