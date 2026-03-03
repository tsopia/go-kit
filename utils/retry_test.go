package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	callCount := 0
	err := Retry(context.Background(), RetryOptions{MaxAttempts: 3, Delay: 10 * time.Millisecond}, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestRetry_FinalFailure(t *testing.T) {
	callCount := 0
	expectedErr := errors.New("always fail")

	err := Retry(context.Background(), RetryOptions{MaxAttempts: 3, Delay: 10 * time.Millisecond}, func() error {
		callCount++
		return expectedErr
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	callCount := 0
	err := Retry(context.Background(), RetryOptions{MaxAttempts: 5, Delay: 10 * time.Millisecond}, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	err := Retry(ctx, RetryOptions{MaxAttempts: 5, Delay: 100 * time.Millisecond}, func() error {
		callCount++
		if callCount == 1 {
			cancel()
		}
		return errors.New("failure")
	})

	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call before cancellation, got %d", callCount)
	}
}

func TestRetryWithBackoff_DefaultOptions(t *testing.T) {
	opts := DefaultBackoffOptions()

	if opts.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts=3, got %d", opts.MaxAttempts)
	}
	if opts.InitialDelay != time.Second {
		t.Fatalf("expected InitialDelay=1s, got %v", opts.InitialDelay)
	}
	if opts.MaxDelay != 30*time.Second {
		t.Fatalf("expected MaxDelay=30s, got %v", opts.MaxDelay)
	}
	if opts.BackoffFactor != 2.0 {
		t.Fatalf("expected BackoffFactor=2.0, got %f", opts.BackoffFactor)
	}
	if !opts.JitterEnabled {
		t.Fatal("expected JitterEnabled=true")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	callCount := 0
	opts := BackoffOptions{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond, BackoffFactor: 2.0}

	err := RetryWithBackoff(context.Background(), opts, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestCalculateDelay_BackoffAndCap(t *testing.T) {
	opts := BackoffOptions{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      250 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterEnabled: false,
	}

	if got := calculateDelay(opts, 0); got != 100*time.Millisecond {
		t.Fatalf("attempt 0 delay mismatch: got %v", got)
	}
	if got := calculateDelay(opts, 1); got != 200*time.Millisecond {
		t.Fatalf("attempt 1 delay mismatch: got %v", got)
	}
	if got := calculateDelay(opts, 2); got != 250*time.Millisecond {
		t.Fatalf("attempt 2 delay mismatch with cap: got %v", got)
	}
}
