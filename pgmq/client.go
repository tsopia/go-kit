package pgmq

import (
	"context"
	"sync"
	"time"
)

var (
	clientMu sync.RWMutex
	_client  *Client
)

// Client SDK 风格客户端
type Client struct {
	db   DB
	opts []Option
}

// NewClient 创建 Client
func NewClient(db DB, opts ...Option) (*Client, error) {
	if db == nil {
		return nil, ErrMissingDB
	}
	return &Client{db: db, opts: opts}, nil
}

// Configure 初始化/替换默认 Client
func Configure(db DB, opts ...Option) (*Client, error) {
	client, err := NewClient(db, opts...)
	if err != nil {
		return nil, err
	}
	clientMu.Lock()
	_client = client
	clientMu.Unlock()
	return client, nil
}

// GetClient 获取默认 Client
func GetClient() *Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return _client
}

func resolveClient(overrides ...*Client) (*Client, error) {
	if len(overrides) > 0 && overrides[0] != nil {
		return overrides[0], nil
	}
	clientMu.RLock()
	defer clientMu.RUnlock()
	if _client == nil {
		return nil, ErrMissingClient
	}
	return _client, nil
}

// NewQueueWithClient 使用 Client 创建队列实例
func NewQueueWithClient[T any](ctx context.Context, name string, c ...*Client) (*Queue[T], error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	merged := append([]Option{}, client.opts...)
	return NewQueue(ctx, client.db, name, merged...)
}

// SendMsg 发送消息
func SendMsg[T any](ctx context.Context, queue string, payload T, c ...*Client) (int64, error) {
	client, err := resolveClient(c...)
	if err != nil {
		return 0, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return 0, err
	}
	return q.Send(ctx, payload)
}

// SendMsgWithDelay 发送消息（延迟秒）
func SendMsgWithDelay[T any](ctx context.Context, queue string, payload T, delay time.Duration, c ...*Client) (int64, error) {
	client, err := resolveClient(c...)
	if err != nil {
		return 0, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return 0, err
	}
	return q.SendWithDelay(ctx, payload, delay)
}

// SendBatchMsg 批量发送消息
func SendBatchMsg[T any](ctx context.Context, queue string, payloads []T, c ...*Client) ([]int64, error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return nil, err
	}
	return q.SendBatch(ctx, payloads)
}

// SendBatchMsgWithDelay 批量发送消息（延迟秒）
func SendBatchMsgWithDelay[T any](ctx context.Context, queue string, payloads []T, delay time.Duration, c ...*Client) ([]int64, error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return nil, err
	}
	return q.SendBatchWithDelay(ctx, payloads, delay)
}

// ReadMsg 读取消息
func ReadMsg[T any](ctx context.Context, queue string, opts ReadOptions, c ...*Client) ([]Message[T], error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return nil, err
	}
	return q.Read(ctx, opts)
}

// PopMsg 读取并删除消息
func PopMsg[T any](ctx context.Context, queue string, c ...*Client) (*Message[T], error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return nil, err
	}
	return q.Pop(ctx)
}

// ArchiveMsg 归档消息
func ArchiveMsg(ctx context.Context, queue string, messageID int64, c ...*Client) error {
	client, err := resolveClient(c...)
	if err != nil {
		return err
	}
	q, err := NewQueueWithClient[any](ctx, queue, client)
	if err != nil {
		return err
	}
	return q.Archive(ctx, messageID)
}

// DeleteMsg 删除消息
func DeleteMsg(ctx context.Context, queue string, messageID int64, c ...*Client) error {
	client, err := resolveClient(c...)
	if err != nil {
		return err
	}
	q, err := NewQueueWithClient[any](ctx, queue, client)
	if err != nil {
		return err
	}
	return q.Delete(ctx, messageID)
}

// SetVisibilityTimeoutMsg 设置消息可见性超时
func SetVisibilityTimeoutMsg[T any](ctx context.Context, queue string, messageID int64, delay time.Duration, c ...*Client) (*Message[T], error) {
	client, err := resolveClient(c...)
	if err != nil {
		return nil, err
	}
	q, err := NewQueueWithClient[T](ctx, queue, client)
	if err != nil {
		return nil, err
	}
	return q.SetVisibilityTimeout(ctx, messageID, delay)
}
