package pgmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Queue 队列封装
type Queue[T any] struct {
	db      DB
	name    string
	config  QueueConfig
	logger  SimpleLogger
	metrics Metrics
}

// CreateExtension 创建 pgmq 扩展
func CreateExtension(ctx context.Context, db DB) error {
	if db == nil {
		return ErrMissingDB
	}
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pgmq CASCADE"); err != nil {
		return fmt.Errorf("创建扩展失败: %w", err)
	}
	return nil
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

// Name 队列名称
func (q *Queue[T]) Name() string {
	return q.name
}

// Send 发送消息
func (q *Queue[T]) Send(ctx context.Context, payload T) (int64, error) {
	return q.SendWithDelay(ctx, payload, 0)
}

// SendWithDelay 发送消息（延迟秒）
func (q *Queue[T]) SendWithDelay(ctx context.Context, payload T, delay time.Duration) (int64, error) {
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

// SendWithDelayTimestamp 发送消息（指定时间）
func (q *Queue[T]) SendWithDelayTimestamp(ctx context.Context, payload T, delay time.Time) (int64, error) {
	data, err := encodePayload(payload)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT %s.send($1, $2, $3::timestamptz)", q.config.Schema)
	var messageID int64
	if err := q.db.QueryRowContext(ctx, query, q.name, data, delay).Scan(&messageID); err != nil {
		return 0, fmt.Errorf("send_at 失败: %w", err)
	}
	return messageID, nil
}

// SendBatch 批量发送消息
func (q *Queue[T]) SendBatch(ctx context.Context, payloads []T) ([]int64, error) {
	return q.SendBatchWithDelay(ctx, payloads, 0)
}

// SendBatchWithDelay 批量发送消息（延迟秒）
func (q *Queue[T]) SendBatchWithDelay(ctx context.Context, payloads []T, delay time.Duration) ([]int64, error) {
	if delay < 0 {
		return nil, ErrInvalidDelay
	}
	data, err := encodePayloads(payloads)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s.send_batch($1, $2::jsonb[], $3::int)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, data, int(delay.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("send_batch 失败: %w", err)
	}
	defer rows.Close()

	messageIDs := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("send_batch 失败: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("send_batch 失败: %w", err)
	}
	return messageIDs, nil
}

// SendBatchWithDelayTimestamp 批量发送消息（指定时间）
func (q *Queue[T]) SendBatchWithDelayTimestamp(ctx context.Context, payloads []T, delay time.Time) ([]int64, error) {
	data, err := encodePayloads(payloads)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s.send_batch($1, $2::jsonb[], $3::timestamptz)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, data, delay)
	if err != nil {
		return nil, fmt.Errorf("send_batch_at 失败: %w", err)
	}
	defer rows.Close()

	messageIDs := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("send_batch_at 失败: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("send_batch_at 失败: %w", err)
	}
	return messageIDs, nil
}

// Read 读取消息
func (q *Queue[T]) Read(ctx context.Context, opts ReadOptions) ([]Message[T], error) {
	options, err := normalizeReadOptions(q.config, opts)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s.read($1, $2, $3)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, int(options.VisibilityTimeout.Seconds()), options.Quantity)
	if err != nil {
		return nil, fmt.Errorf("read 失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取消息失败: %w", err)
	}
	hasHeaders := len(columns) > 5

	messages := make([]Message[T], 0)
	for rows.Next() {
		msg, err := scanMessage[T](rows.Scan, hasHeaders)
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
	query := fmt.Sprintf("SELECT * FROM %s.pop($1)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name)
	if err != nil {
		return nil, fmt.Errorf("pop 失败: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("pop 失败: %w", err)
	}
	hasHeaders := len(columns) > 5

	msg, err := scanMessage[T](rows.Scan, hasHeaders)
	if err != nil {
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

// ArchiveBatch 批量归档消息
func (q *Queue[T]) ArchiveBatch(ctx context.Context, messageIDs []int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT %s.archive($1, $2::bigint[])", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("archive_batch 失败: %w", err)
	}
	defer rows.Close()

	archived := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("archive_batch 失败: %w", err)
		}
		archived = append(archived, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("archive_batch 失败: %w", err)
	}
	return archived, nil
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

// DeleteBatch 批量删除消息
func (q *Queue[T]) DeleteBatch(ctx context.Context, messageIDs []int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT %s.delete($1, $2::bigint[])", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("delete_batch 失败: %w", err)
	}
	defer rows.Close()

	deleted := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("delete_batch 失败: %w", err)
		}
		deleted = append(deleted, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("delete_batch 失败: %w", err)
	}
	return deleted, nil
}

// SetVisibilityTimeout 设置消息可见性超时
func (q *Queue[T]) SetVisibilityTimeout(ctx context.Context, messageID int64, delay time.Duration) (*Message[T], error) {
	if delay < 0 {
		return nil, ErrInvalidDelay
	}
	query := fmt.Sprintf("SELECT * FROM %s.set_vt($1, $2, $3::int)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, q.name, messageID, int(delay.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("set_vt 失败: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNoRows
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("set_vt 失败: %w", err)
	}
	hasHeaders := len(columns) > 5

	msg, err := scanMessage[T](rows.Scan, hasHeaders)
	if err != nil {
		return nil, err
	}
	return &msg, nil
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

// SendRaw 直接发送 JSON 消息
func (q *Queue[T]) SendRaw(ctx context.Context, queue string, payload json.RawMessage) (int64, error) {
	return q.SendRawWithDelay(ctx, queue, payload, 0)
}

// SendRawWithDelay 直接发送 JSON 消息（延迟秒）
func (q *Queue[T]) SendRawWithDelay(ctx context.Context, queue string, payload json.RawMessage, delay time.Duration) (int64, error) {
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

// SendRawWithDelayTimestamp 直接发送 JSON 消息（指定时间）
func (q *Queue[T]) SendRawWithDelayTimestamp(ctx context.Context, queue string, payload json.RawMessage, delay time.Time) (int64, error) {
	if err := validateQueueName(queue); err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT %s.send($1, $2, $3::timestamptz)", q.config.Schema)
	var messageID int64
	if err := q.db.QueryRowContext(ctx, query, queue, payload, delay).Scan(&messageID); err != nil {
		return 0, fmt.Errorf("send_at 失败: %w", err)
	}
	return messageID, nil
}

// SendBatchRaw 批量发送 JSON 消息
func (q *Queue[T]) SendBatchRaw(ctx context.Context, queue string, payloads []json.RawMessage) ([]int64, error) {
	return q.SendBatchRawWithDelay(ctx, queue, payloads, 0)
}

// SendBatchRawWithDelay 批量发送 JSON 消息（延迟秒）
func (q *Queue[T]) SendBatchRawWithDelay(ctx context.Context, queue string, payloads []json.RawMessage, delay time.Duration) ([]int64, error) {
	if delay < 0 {
		return nil, ErrInvalidDelay
	}
	if err := validateQueueName(queue); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s.send_batch($1, $2::jsonb[], $3::int)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, queue, payloads, int(delay.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("send_batch 失败: %w", err)
	}
	defer rows.Close()

	messageIDs := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("send_batch 失败: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("send_batch 失败: %w", err)
	}
	return messageIDs, nil
}

// SendBatchRawWithDelayTimestamp 批量发送 JSON 消息（指定时间）
func (q *Queue[T]) SendBatchRawWithDelayTimestamp(ctx context.Context, queue string, payloads []json.RawMessage, delay time.Time) ([]int64, error) {
	if err := validateQueueName(queue); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s.send_batch($1, $2::jsonb[], $3::timestamptz)", q.config.Schema)
	rows, err := q.db.QueryContext(ctx, query, queue, payloads, delay)
	if err != nil {
		return nil, fmt.Errorf("send_batch_at 失败: %w", err)
	}
	defer rows.Close()

	messageIDs := make([]int64, 0)
	for rows.Next() {
		var messageID int64
		if err := rows.Scan(&messageID); err != nil {
			return nil, fmt.Errorf("send_batch_at 失败: %w", err)
		}
		messageIDs = append(messageIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("send_batch_at 失败: %w", err)
	}
	return messageIDs, nil
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

func encodePayloads[T any](payloads []T) ([]json.RawMessage, error) {
	if len(payloads) == 0 {
		return []json.RawMessage{}, nil
	}
	result := make([]json.RawMessage, 0, len(payloads))
	for _, payload := range payloads {
		data, err := encodePayload(payload)
		if err != nil {
			return nil, err
		}
		result = append(result, data)
	}
	return result, nil
}

func scanMessage[T any](scan func(dest ...any) error, hasHeaders bool) (Message[T], error) {
	var msg Message[T]
	if hasHeaders {
		if err := scan(&msg.MsgID, &msg.ReadCount, &msg.EnqueuedAt, &msg.VT, &msg.Raw, &msg.Headers); err != nil {
			return Message[T]{}, fmt.Errorf("解析消息失败: %w", err)
		}
	} else {
		if err := scan(&msg.MsgID, &msg.ReadCount, &msg.EnqueuedAt, &msg.VT, &msg.Raw); err != nil {
			return Message[T]{}, fmt.Errorf("解析消息失败: %w", err)
		}
	}
	if err := json.Unmarshal(msg.Raw, &msg.Body); err != nil {
		return Message[T]{}, fmt.Errorf("%w: %v", ErrDecodeMessage, err)
	}
	return msg, nil
}
