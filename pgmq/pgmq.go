package pgmq

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// SimpleLogger 基础日志接口
type SimpleLogger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

type stdLogger struct{}

func (l stdLogger) Info(msg string, fields ...interface{})  { log.Printf("INFO %s %v", msg, fields) }
func (l stdLogger) Warn(msg string, fields ...interface{})  { log.Printf("WARN %s %v", msg, fields) }
func (l stdLogger) Error(msg string, fields ...interface{}) { log.Printf("ERROR %s %v", msg, fields) }

// Metrics 可插拔指标接口
type Metrics interface {
	IncProcessCount(queue string, status string)
	ObserveLatency(queue string, duration time.Duration)
}

type noopMetrics struct{}

func (m noopMetrics) IncProcessCount(queue string, status string)         {}
func (m noopMetrics) ObserveLatency(queue string, duration time.Duration) {}

// Message PGMQ 消息
type Message[T any] struct {
	ID         int64
	ReadCount  int32
	EnqueuedAt time.Time
	VisibleAt  time.Time
	Payload    json.RawMessage
	Body       T
}

// Queue 队列封装
type Queue[T any] struct {
	db      DB
	name    string
	config  QueueConfig
	logger  SimpleLogger
	metrics Metrics
}

// NewQueue 创建队列
func NewQueue[T any](ctx context.Context, db DB, name string, opts ...Option) (*Queue[T], error) {
	if db == nil {
		return nil, ErrMissingDB
	}
	if err := validateQueueName(name); err != nil {
		return nil, err
	}

	cfg := QueueConfig{
		CheckExtension: true,
		EnsureQueue:    true,
	}
	options := &QueueOptions{}
	for _, opt := range opts {
		opt(&cfg, options)
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	queue := &Queue[T]{
		db:      db,
		name:    name,
		config:  cfg,
		logger:  options.logger,
		metrics: options.metrics,
	}

	if queue.logger == nil {
		queue.logger = stdLogger{}
	}
	if queue.metrics == nil {
		queue.metrics = noopMetrics{}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if cfg.CheckExtension {
		if err := queue.checkExtension(ctx); err != nil {
			return nil, err
		}
	}

	if cfg.EnsureQueue {
		if err := queue.ensureQueue(ctx); err != nil {
			return nil, err
		}
	}

	return queue, nil
}

// Send 发送消息
func (q *Queue[T]) Send(ctx context.Context, payload T, delay time.Duration) (int64, error) {
	if delay < 0 {
		return 0, ErrInvalidDelay
	}

	data, err := encodePayload(payload)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("SELECT %s.send($1, $2, $3)", q.config.Schema)
	var messageID int64
	if err := q.db.QueryRowContext(ctx, query, q.name, data, int(delay.Seconds())).Scan(&messageID); err != nil {
		return 0, fmt.Errorf("send 失败: %w", err)
	}
	return messageID, nil
}

// Read 读取消息
func (q *Queue[T]) Read(ctx context.Context, opts ReadOptions) ([]Message[T], error) {
	options, err := normalizeReadOptions(q.config, opts)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT msg_id, read_ct, enqueued_at, vt, message FROM %s.read($1, $2, $3)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, int(options.VisibilityTimeout.Seconds()), options.Quantity)
	if err != nil {
		return nil, fmt.Errorf("read 失败: %w", err)
	}
	defer rows.Close()

	messages := make([]Message[T], 0)
	for rows.Next() {
		msg, err := scanMessage[T](rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取消息失败: %w", err)
	}
	return messages, nil
}

// Pop 读取并删除消息
func (q *Queue[T]) Pop(ctx context.Context) (*Message[T], error) {
	query := fmt.Sprintf("SELECT msg_id, read_ct, enqueued_at, vt, message FROM %s.pop($1)", q.config.Schema)
	row := q.db.QueryRowContext(ctx, query, q.name)
	msg, err := scanMessageRow[T](row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

// Archive 归档消息
func (q *Queue[T]) Archive(ctx context.Context, messageID int64) error {
	query := fmt.Sprintf("SELECT %s.archive($1, $2)", q.config.Schema)
	var ok bool
	if err := q.db.QueryRowContext(ctx, query, q.name, messageID).Scan(&ok); err != nil {
		return fmt.Errorf("archive 失败: %w", err)
	}
	return nil
}

// Delete 删除消息
func (q *Queue[T]) Delete(ctx context.Context, messageID int64) error {
	query := fmt.Sprintf("SELECT %s.delete($1, $2)", q.config.Schema)
	var ok bool
	if err := q.db.QueryRowContext(ctx, query, q.name, messageID).Scan(&ok); err != nil {
		return fmt.Errorf("delete 失败: %w", err)
	}
	return nil
}

// SetVisibilityTimeout 设置消息可见性超时
func (q *Queue[T]) SetVisibilityTimeout(ctx context.Context, messageID int64, delay time.Duration) error {
	if delay < 0 {
		return ErrInvalidDelay
	}
	query := fmt.Sprintf("SELECT %s.set_vt($1, $2, $3)", q.config.Schema)
	var ok bool
	if err := q.db.QueryRowContext(ctx, query, q.name, messageID, int(delay.Seconds())).Scan(&ok); err != nil {
		return fmt.Errorf("set_vt 失败: %w", err)
	}
	return nil
}

// Drop 删除队列
func (q *Queue[T]) Drop(ctx context.Context) error {
	query := fmt.Sprintf("SELECT %s.drop_queue($1, true)", q.config.Schema)
	var dropped bool
	if err := q.db.QueryRowContext(ctx, query, q.name).Scan(&dropped); err != nil {
		return fmt.Errorf("drop_queue 失败: %w", err)
	}
	return nil
}

// Consume 启动消费者
func (q *Queue[T]) Consume(ctx context.Context, handler func(context.Context, *Message[T]) error) error {
	if handler == nil {
		return ErrInvalidConfig
	}
	if ctx == nil {
		ctx = context.Background()
	}

	workerCount := q.config.Consumer.Concurrency
	errCh := make(chan error, workerCount)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for {
				if workerCtx.Err() != nil {
					return
				}
				messages, err := q.Read(workerCtx, ReadOptions{Quantity: 1, VisibilityTimeout: q.config.Visibility})
				if err != nil {
					errCh <- err
					return
				}
				if len(messages) == 0 {
					select {
					case <-time.After(200 * time.Millisecond):
					case <-workerCtx.Done():
						return
					}
					continue
				}

				for i := range messages {
					msg := messages[i]
					start := time.Now()
					err := handler(workerCtx, &msg)
					q.metrics.ObserveLatency(q.name, time.Since(start))
					if err == nil {
						if err := q.Archive(workerCtx, msg.ID); err != nil {
							errCh <- err
							return
						}
						q.metrics.IncProcessCount(q.name, "success")
						continue
					}
					if err := q.handleFailure(workerCtx, &msg); err != nil {
						errCh <- err
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			cancel()
			return err
		}
	}

	return nil
}

func (q *Queue[T]) handleFailure(ctx context.Context, msg *Message[T]) error {
	if msg.ReadCount < int32(q.config.Retry.MaxRetries) {
		delay := computeBackoffDelay(q.config.Retry, msg.ReadCount)
		if q.config.Retry.Jitter {
			jitter := rand.Float64() + 0.5
			delay = time.Duration(float64(delay) * jitter)
		}
		if err := q.SetVisibilityTimeout(ctx, msg.ID, delay); err != nil {
			return err
		}
		q.metrics.IncProcessCount(q.name, "retry")
		return nil
	}

	dlq := q.name + q.config.DLQSuffix
	if q.config.EnsureQueue {
		if err := q.ensureQueueWithName(ctx, dlq); err != nil {
			return err
		}
	}
	if _, err := q.SendRaw(ctx, dlq, msg.Payload, 0); err != nil {
		return err
	}
	if err := q.Delete(ctx, msg.ID); err != nil {
		return err
	}
	q.metrics.IncProcessCount(q.name, "dlq")
	return nil
}

// SendRaw 直接发送 JSON 消息
func (q *Queue[T]) SendRaw(ctx context.Context, queue string, payload json.RawMessage, delay time.Duration) (int64, error) {
	if delay < 0 {
		return 0, ErrInvalidDelay
	}
	if err := validateQueueName(queue); err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT %s.send($1, $2, $3)", q.config.Schema)
	var messageID int64
	if err := q.db.QueryRowContext(ctx, query, queue, payload, int(delay.Seconds())).Scan(&messageID); err != nil {
		return 0, fmt.Errorf("send 失败: %w", err)
	}
	return messageID, nil
}

func (q *Queue[T]) checkExtension(ctx context.Context) error {
	row := q.db.QueryRowContext(ctx, "SELECT exists(SELECT 1 FROM pg_extension WHERE extname = $1)", ExtensionName)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("扩展检查失败: %w", err)
	}
	if !exists {
		return ErrExtensionMissing
	}
	return nil
}

func (q *Queue[T]) ensureQueue(ctx context.Context) error {
	return q.ensureQueueWithName(ctx, q.name)
}

func (q *Queue[T]) ensureQueueWithName(ctx context.Context, name string) error {
	query := fmt.Sprintf("SELECT %s.create($1)", q.config.Schema)
	var created bool
	if err := q.db.QueryRowContext(ctx, query, name).Scan(&created); err != nil {
		return fmt.Errorf("创建队列失败: %w", err)
	}
	return nil
}

func encodePayload[T any](payload T) (json.RawMessage, error) {
	if raw, ok := any(payload).(json.RawMessage); ok {
		return raw, nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("消息序列化失败: %w", err)
	}
	return data, nil
}

func scanMessage[T any](rows *sql.Rows) (Message[T], error) {
	var msg Message[T]
	if err := rows.Scan(&msg.ID, &msg.ReadCount, &msg.EnqueuedAt, &msg.VisibleAt, &msg.Payload); err != nil {
		return Message[T]{}, fmt.Errorf("解析消息失败: %w", err)
	}
	if err := json.Unmarshal(msg.Payload, &msg.Body); err != nil {
		return Message[T]{}, fmt.Errorf("%w: %v", ErrDecodeMessage, err)
	}
	return msg, nil
}

func scanMessageRow[T any](row *sql.Row) (Message[T], error) {
	var msg Message[T]
	if err := row.Scan(&msg.ID, &msg.ReadCount, &msg.EnqueuedAt, &msg.VisibleAt, &msg.Payload); err != nil {
		return Message[T]{}, err
	}
	if err := json.Unmarshal(msg.Payload, &msg.Body); err != nil {
		return Message[T]{}, fmt.Errorf("%w: %v", ErrDecodeMessage, err)
	}
	return msg, nil
}
