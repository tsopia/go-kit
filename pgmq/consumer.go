package pgmq

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

const (
	StatusSuccess = "success"
	StatusRetry   = "retry"
	StatusDLQ     = "dlq"
)

// HandlerFunc processes a message.
type HandlerFunc[T any] func(context.Context, *Message[T]) error

// ConsumerOption overrides consumer behavior.
type ConsumerOption func(*ConsumerConfig)

// WithConsumerVisibilityTimeout overrides visibility timeout for consumer reads.
func WithConsumerVisibilityTimeout(vt time.Duration) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.VisibilityTimeout = vt
	}
}

// WithConsumerPollInterval sets the polling interval when no messages are found.
func WithConsumerPollInterval(interval time.Duration) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.PollInterval = interval
	}
}

// WithConsumerMaxConcurrency sets max concurrent handlers.
func WithConsumerMaxConcurrency(n int) ConsumerOption {
	return func(cfg *ConsumerConfig) {
		cfg.MaxConcurrency = n
	}
}

// Consumer manages the worker lifecycle.
type Consumer[T any] struct {
	queue      *Queue[T]
	handler    HandlerFunc[T]
	loopCtx    context.Context
	handlerCtx context.Context
	cancel     context.CancelFunc
	cfg        ConsumerConfig
	sem        chan struct{}
	wg         sync.WaitGroup
	errOnce    sync.Once
	errCh      chan error
}

// StartConsumer starts polling the queue with concurrency control.
func (q *Queue[T]) StartConsumer(ctx context.Context, handler HandlerFunc[T], opts ...ConsumerOption) (*Consumer[T], error) {
	if handler == nil {
		return nil, errors.New("pgmq: handler is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg := q.config.Consumer
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	cfg = normalizeConsumerConfig(cfg)

	loopCtx, cancel := context.WithCancel(ctx)
	c := &Consumer[T]{
		queue:      q,
		handler:    handler,
		loopCtx:    loopCtx,
		handlerCtx: ctx,
		cancel:     cancel,
		cfg:        cfg,
		sem:        make(chan struct{}, cfg.MaxConcurrency),
		errCh:      make(chan error, 1),
	}

	c.wg.Add(1)
	go c.loop()

	return c, nil
}

// Consume starts a consumer and blocks until it stops.
func (q *Queue[T]) Consume(ctx context.Context, handler HandlerFunc[T]) error {
	consumer, err := q.StartConsumer(ctx, handler)
	if err != nil {
		return err
	}
	return consumer.Wait(ctx)
}

// Wait blocks until the consumer stops or returns an error.
func (c *Consumer[T]) Wait(ctx context.Context) error {
	if c == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	if ctx == nil {
		<-done
		return c.firstError()
	}

	select {
	case <-done:
		return c.firstError()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops polling new messages and waits for in-flight handlers.
func (c *Consumer[T]) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.cancel()
	return c.Wait(ctx)
}

func (c *Consumer[T]) loop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.loopCtx.Done():
			return
		default:
		}

		messages, err := c.queue.Read(c.loopCtx, ReadOptions{Quantity: 1, VisibilityTimeout: c.cfg.VisibilityTimeout})
		if err != nil {
			c.reportErr(err)
			return
		}
		if len(messages) == 0 {
			if !sleepWithContext(c.loopCtx, c.cfg.PollInterval) {
				return
			}
			continue
		}

		select {
		case c.sem <- struct{}{}:
		case <-c.loopCtx.Done():
			return
		}

		c.wg.Add(1)
		msg := messages[0]
		go func(m Message[T]) {
			defer c.wg.Done()
			defer func() { <-c.sem }()
			c.processMessage(&m)
		}(msg)
	}
}

func (c *Consumer[T]) processMessage(msg *Message[T]) {
	start := time.Now()
	err := c.handler(c.handlerCtx, msg)
	c.queue.metrics.ObserveLatency(c.queue.name, time.Since(start))

	if err == nil {
		if err := c.queue.Archive(c.handlerCtx, msg.MsgID); err == nil {
			c.queue.metrics.IncProcessCount(c.queue.name, StatusSuccess)
			return
		}
	}

	if err := c.handleFailure(msg); err != nil {
		c.reportErr(err)
	}
}

func (c *Consumer[T]) handleFailure(msg *Message[T]) error {
	if msg.ReadCount < int64(c.queue.config.Retry.MaxRetries) {
		delay := computeBackoffDelay(c.queue.config.Retry, int32(msg.ReadCount))
		if c.queue.config.Retry.Jitter {
			jitter := rand.Float64() + 0.5
			delay = time.Duration(float64(delay) * jitter)
		}
		if _, err := c.queue.SetVisibilityTimeout(c.handlerCtx, msg.MsgID, delay); err != nil {
			return err
		}
		c.queue.metrics.IncProcessCount(c.queue.name, StatusRetry)
		return nil
	}

	dlq := c.queue.name + c.queue.config.DLQSuffix
	if c.queue.config.EnsureQueue {
		if err := c.queue.ensureQueueWithName(c.handlerCtx, dlq); err != nil {
			return err
		}
	}
	if _, err := c.queue.SendRawWithDelay(c.handlerCtx, dlq, msg.Raw, 0); err != nil {
		return err
	}
	if err := c.queue.Delete(c.handlerCtx, msg.MsgID); err != nil {
		return err
	}
	c.queue.metrics.IncProcessCount(c.queue.name, StatusDLQ)
	return nil
}

func (c *Consumer[T]) reportErr(err error) {
	if err == nil {
		return
	}
	c.errOnce.Do(func() {
		c.errCh <- err
		c.cancel()
	})
}

func (c *Consumer[T]) firstError() error {
	select {
	case err := <-c.errCh:
		return err
	default:
		return nil
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		d = DefaultPollInterval
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
