package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// RetryOptions 固定延迟重试配置
// MaxAttempts 小于等于 0 时按 1 次处理。
type RetryOptions struct {
	MaxAttempts int
	Delay       time.Duration
}

// BackoffOptions 指数退避重试配置。
type BackoffOptions struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	JitterEnabled bool
}

// DefaultBackoffOptions 返回默认退避配置。
func DefaultBackoffOptions() BackoffOptions {
	return BackoffOptions{
		MaxAttempts:   3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterEnabled: true,
	}
}

// Retry 按固定延迟执行重试。
func Retry(ctx context.Context, opts RetryOptions, fn func() error) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}

	var err error
	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}

		if attempt < opts.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("重试被取消: %w", ctx.Err())
			case <-time.After(opts.Delay):
			}
		}
	}

	return fmt.Errorf("重试 %d 次后仍失败: %w", opts.MaxAttempts, err)
}

// RetryWithBackoff 按指数退避执行重试。
func RetryWithBackoff(ctx context.Context, opts BackoffOptions, fn func() error) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.InitialDelay <= 0 {
		opts.InitialDelay = time.Second
	}
	if opts.BackoffFactor <= 0 {
		opts.BackoffFactor = 2.0
	}

	var err error
	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}

		if attempt < opts.MaxAttempts-1 {
			sleepDuration := calculateDelay(opts, attempt)
			select {
			case <-ctx.Done():
				return fmt.Errorf("重试被取消: %w", ctx.Err())
			case <-time.After(sleepDuration):
			}
		}
	}

	return fmt.Errorf("重试 %d 次后仍失败: %w", opts.MaxAttempts, err)
}

// calculateDelay 计算当前 attempt 的退避时长。
func calculateDelay(opts BackoffOptions, attempt int) time.Duration {
	delay := float64(opts.InitialDelay) * math.Pow(opts.BackoffFactor, float64(attempt))
	if opts.MaxDelay > 0 && time.Duration(delay) > opts.MaxDelay {
		delay = float64(opts.MaxDelay)
	}
	if opts.JitterEnabled {
		jitter := rand.Float64() * 0.1
		delay = delay * (1 + jitter)
	}
	return time.Duration(delay)
}
