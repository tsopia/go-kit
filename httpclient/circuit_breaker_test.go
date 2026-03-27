package httpclient

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		MaxRequests:      2,
		Timeout:          100 * time.Millisecond,
	}

	cb := newCircuitBreaker(config).(*statefulCircuitBreaker)

	// 初始状态应为 Closed
	if state := cb.State(); state != string(stateClosed) {
		t.Fatalf("expected state %s, got %s", stateClosed, state)
	}

	ctx := context.Background()
	dummyErr := errors.New("dummy error")

	// 模拟连续失败 3 次，应该触发熔断
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error {
			return dummyErr
		})
		if !errors.Is(err, dummyErr) {
			t.Fatalf("expected dummy error, got %v", err)
		}
	}

	// 达到阈值后，状态应该是 Open
	if state := cb.State(); state != string(stateOpen) {
		t.Fatalf("expected state %s, got %s", stateOpen, state)
	}

	// 在 Open 状态下请求应该直接被拒绝
	err := cb.Execute(ctx, func() error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// 等待超时时间过去，状态应该变为 HalfOpen
	time.Sleep(150 * time.Millisecond)

	// HalfOpen 允许有限请求通过
	// 半开状态下我们直接验证发送 2 个成功的请求，应当能正确触发恢复到 Closed 状态
	for i := 0; i < 2; i++ {
		e := cb.Execute(ctx, func() error {
			time.Sleep(1 * time.Millisecond) // 少许处理时间
			return nil
		})
		if e != nil {
			t.Fatalf("unexpected error in half open: %v", e)
		}
	}

	// 由于上面刚好有 SuccessThreshold(2) 次成功请求，状态应该重置为 Closed
	if state := cb.State(); state != string(stateClosed) {
		t.Fatalf("expected state %s, got %s", stateClosed, state)
	}
}

func TestCircuitBreaker_HalfOpenFails(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		MaxRequests:      2,
		Timeout:          50 * time.Millisecond,
	}

	cb := newCircuitBreaker(config).(*statefulCircuitBreaker)
	ctx := context.Background()
	dummyErr := errors.New("dummy error")

	// 触发熔断
	_ = cb.Execute(ctx, func() error { return dummyErr })

	// 等待半开
	time.Sleep(100 * time.Millisecond)

	if state := cb.State(); state != string(stateHalfOpen) {
		t.Fatalf("expected state %s, got %s", stateHalfOpen, state)
	}

	// 在半开状态下，一次失败就应该重新熔断
	_ = cb.Execute(ctx, func() error { return dummyErr })

	if state := cb.State(); state != string(stateOpen) {
		t.Fatalf("expected state %s, got %s", stateOpen, state)
	}
}

func TestCircuitBreaker_HalfOpenLimitsConcurrentRequests(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		MaxRequests:      1,
		Timeout:          20 * time.Millisecond,
	}

	cb := newCircuitBreaker(config).(*statefulCircuitBreaker)
	ctx := context.Background()
	dummyErr := errors.New("dummy error")

	if err := cb.Execute(ctx, func() error { return dummyErr }); !errors.Is(err, dummyErr) {
		t.Fatalf("expected dummy error, got %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	if state := cb.State(); state != string(stateHalfOpen) {
		t.Fatalf("expected state %s, got %s", stateHalfOpen, state)
	}

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstErrCh := make(chan error, 1)

	go func() {
		firstErrCh <- cb.Execute(ctx, func() error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
	}()

	<-firstStarted

	secondExecuted := false
	err := cb.Execute(ctx, func() error {
		secondExecuted = true
		return nil
	})
	if !errors.Is(err, ErrCircuitHalfOpenLimited) {
		close(releaseFirst)
		<-firstErrCh
		t.Fatalf("expected ErrCircuitHalfOpenLimited, got %v", err)
	}
	if secondExecuted {
		close(releaseFirst)
		<-firstErrCh
		t.Fatal("second half-open probe should not be executed while first probe is still in flight")
	}

	close(releaseFirst)
	if err := <-firstErrCh; err != nil {
		t.Fatalf("expected first probe to succeed, got %v", err)
	}
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	config := CircuitBreakerConfig{}
	cb := newCircuitBreaker(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立刻取消

	executed := false
	err := cb.Execute(ctx, func() error {
		executed = true
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if executed {
		t.Fatal("fn should not be executed if context is canceled")
	}
}
