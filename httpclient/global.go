package httpclient

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

var (
	defaultClient   atomic.Value
	defaultClientMu sync.Mutex
)

func getDefaultClient() *Client {
	// 快速路径：无锁读取
	if v := defaultClient.Load(); v != nil {
		if client, ok := v.(*Client); ok {
			return client
		}
	}
	
	// 慢速路径：加锁初始化
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	
	// 双重检查
	if v := defaultClient.Load(); v != nil {
		if client, ok := v.(*Client); ok {
			return client
		}
	}
	
	client := NewClient()
	defaultClient.Store(client)
	return client
}

// ConfigureDefault 配置默认客户端
// 必须在第一次使用全局函数之前调用
func ConfigureDefault(opts ...Option) {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient.Store(NewClient(opts...))
}

// ResetDefault 重置默认客户端（主要用于测试）
func ResetDefault(opts ...Option) *Client {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	client := NewClient(opts...)
	defaultClient.Store(client)
	return client
}

// GetDefaultClient 获取默认客户端实例
func GetDefaultClient() *Client {
	return getDefaultClient()
}

// SetDefaultClient 设置默认客户端
func SetDefaultClient(client *Client) {
	if client == nil {
		return
	}
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClient.Store(client)
}

// ==================== Context-first 全局函数 ====================

// Get 发送 GET 请求
func Get(ctx context.Context, url string) (*Response, error) {
	return getDefaultClient().Get(ctx, url)
}

// Post 发送 POST 请求
func Post(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Post(ctx, url, body)
}

// PostJSON 发送 JSON POST 请求
func PostJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return getDefaultClient().PostJSON(ctx, url, data)
}

// Put 发送 PUT 请求
func Put(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Put(ctx, url, body)
}

// PutJSON 发送 JSON PUT 请求
func PutJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return getDefaultClient().PutJSON(ctx, url, data)
}

// Delete 发送 DELETE 请求
func Delete(ctx context.Context, url string) (*Response, error) {
	return getDefaultClient().Delete(ctx, url)
}

// Patch 发送 PATCH 请求
func Patch(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Patch(ctx, url, body)
}

// PatchJSON 发送 JSON PATCH 请求
func PatchJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return getDefaultClient().PatchJSON(ctx, url, data)
}

// Do 直接执行 HTTP 请求
func Do(ctx context.Context, method, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Do(ctx, method, url, body)
}
