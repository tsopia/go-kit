package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client gRPC客户端
type Client struct {
	conn   *grpc.ClientConn
	target string
	opts   []grpc.DialOption
}

// CallResult gRPC调用结果
type CallResult struct {
	Response interface{}
	Error    error
}

// NewClient 创建新的gRPC客户端
func NewClient(target string) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(30 * time.Second),
	}

	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	return &Client{
		conn:   conn,
		target: target,
		opts:   opts,
	}, nil
}

// NewClientWithOptions 使用自定义选项创建gRPC客户端
func NewClientWithOptions(target string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	return &Client{
		conn:   conn,
		target: target,
		opts:   opts,
	}, nil
}

// Close 关闭gRPC连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetConnection 获取gRPC连接
func (c *Client) GetConnection() *grpc.ClientConn {
	return c.conn
}

// GetTarget 获取目标地址
func (c *Client) GetTarget() string {
	return c.target
}

// Call 调用gRPC方法
func (c *Client) Call(method string, request interface{}, response interface{}, opts ...grpc.CallOption) error {
	ctx := context.Background()
	return c.CallWithContext(ctx, method, request, response, opts...)
}

// CallWithContext 使用上下文调用gRPC方法
func (c *Client) CallWithContext(ctx context.Context, method string, request interface{}, response interface{}, opts ...grpc.CallOption) error {
	return c.conn.Invoke(ctx, method, request, response, opts...)
}

// CallWithMetadata 使用元数据调用gRPC方法
func (c *Client) CallWithMetadata(ctx context.Context, method string, request interface{}, response interface{}, md map[string]string, opts ...grpc.CallOption) error {
	grpcMD := metadata.New(md)
	ctx = metadata.NewOutgoingContext(ctx, grpcMD)
	return c.CallWithContext(ctx, method, request, response, opts...)
}

// CallWithTimeout 使用超时调用gRPC方法
func (c *Client) CallWithTimeout(method string, request interface{}, response interface{}, timeout time.Duration, opts ...grpc.CallOption) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.CallWithContext(ctx, method, request, response, opts...)
}

// StreamCall 流式调用gRPC方法
func (c *Client) StreamCall(method string, request interface{}, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx := context.Background()
	return c.StreamCallWithContext(ctx, method, request, opts...)
}

// StreamCallWithContext 使用上下文进行流式调用
func (c *Client) StreamCallWithContext(ctx context.Context, method string, request interface{}, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return c.conn.NewStream(ctx, nil, method, opts...)
}

// Ping 测试连接
func (c *Client) Ping() error {
	// 这里可以添加具体的ping逻辑
	// 例如调用一个简单的健康检查方法
	return nil
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.GetState().String() == "READY"
}

// GetState 获取连接状态
func (c *Client) GetState() string {
	if c.conn != nil {
		return c.conn.GetState().String()
	}
	return "DISCONNECTED"
}

// Reconnect 重新连接
func (c *Client) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := grpc.Dial(c.target, c.opts...)
	if err != nil {
		return fmt.Errorf("failed to reconnect to gRPC server: %v", err)
	}

	c.conn = conn
	return nil
}

// WithInsecure 设置不安全连接
func (c *Client) WithInsecure() *Client {
	c.opts = append(c.opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return c
}

// WithBlock 设置阻塞连接
func (c *Client) WithBlock() *Client {
	c.opts = append(c.opts, grpc.WithBlock())
	return c
}

// WithTimeout 设置连接超时
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.opts = append(c.opts, grpc.WithTimeout(timeout))
	return c
}

// WithUserAgent 设置用户代理
func (c *Client) WithUserAgent(userAgent string) *Client {
	c.opts = append(c.opts, grpc.WithUserAgent(userAgent))
	return c
}

// WithDefaultCallOptions 设置默认调用选项
func (c *Client) WithDefaultCallOptions(opts ...grpc.CallOption) *Client {
	c.opts = append(c.opts, grpc.WithDefaultCallOptions(opts...))
	return c
}

// Pool gRPC连接池
type Pool struct {
	clients map[string]*Client
	maxSize int
}

// NewPool 创建新的gRPC连接池
func NewPool(maxSize int) *Pool {
	return &Pool{
		clients: make(map[string]*Client),
		maxSize: maxSize,
	}
}

// Get 从连接池获取客户端
func (p *Pool) Get(target string) (*Client, error) {
	if client, exists := p.clients[target]; exists {
		if client.IsConnected() {
			return client, nil
		}
		// 连接已断开，重新连接
		if err := client.Reconnect(); err != nil {
			delete(p.clients, target)
		} else {
			return client, nil
		}
	}

	// 创建新连接
	if len(p.clients) >= p.maxSize {
		return nil, fmt.Errorf("connection pool is full")
	}

	client, err := NewClient(target)
	if err != nil {
		return nil, err
	}

	p.clients[target] = client
	return client, nil
}

// Close 关闭连接池
func (p *Pool) Close() error {
	var lastErr error
	for target, client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
		delete(p.clients, target)
	}
	return lastErr
}

// Size 获取连接池大小
func (p *Pool) Size() int {
	return len(p.clients)
}

// Remove 移除连接
func (p *Pool) Remove(target string) error {
	if client, exists := p.clients[target]; exists {
		if err := client.Close(); err != nil {
			return err
		}
		delete(p.clients, target)
	}
	return nil
}
