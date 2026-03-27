package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestClientConcurrency 安全并发压力测试
func TestClientConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClient(
		WithTimeout(5*time.Second),
		WithBaseURL(server.URL),
		WithPool(&PoolConfig{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
		}),
	)

	// 同时发起 100 个请求确保没有 race condition
	var wg sync.WaitGroup
	concurrencyLevel := 100
	errChan := make(chan error, concurrencyLevel)

	for i := 0; i < concurrencyLevel; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// 并发过程中修改 Client 的线程安全配置以测试数据竞争
			if id%10 == 0 {
				client.SetHeader("X-Dynamic-Header", "value")
			}

			resp, err := client.Get(context.Background(), "/")
			if err != nil {
				errChan <- err
				return
			}
			
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Fatalf("concurrency test failed with %d errors. first error: %v", len(errs), errs[0])
	}
}
