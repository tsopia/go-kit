# HTTPClient 包

基于 `net/http` 的 HTTP 客户端封装，提供重试、熔断、限流、调试等企业级能力。默认开箱即用，也可按需覆盖配置或注入自定义 `http.Client`/`Transport`。

## 定位

`httpclient` 是传输型 SDK：

- 包级便捷方法 `Get/Post/...` 零配置可用
- 显式实例 `NewClient` 支持精细控制
- 默认实例通过 `ConfigureDefault` / `ResetDefault` 管理生命周期

## 安装

```bash
go get github.com/tsopia/go-kit/httpclient
```

## 快速开始

### 全局方法（零配置）

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tsopia/go-kit/httpclient"
)

func main() {
	ctx := context.Background()

	resp, err := httpclient.Get(ctx, "https://api.example.com/ping")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.StatusCode, resp.String())
}
```

### 创建独立客户端

```go
cli := httpclient.NewClient(
	httpclient.WithBaseURL("https://api.example.com"),
	httpclient.WithTimeout(5*time.Second),
	httpclient.WithRetry(&httpclient.RetryConfig{MaxRetries: 2}),
)

resp, err := cli.PostJSON(ctx, "/users", map[string]string{"name": "Go"})
```

### 链式请求构建

```go
resp, err := cli.NewRequest(http.MethodPost, "/users").
	Context(ctx).
	Header("Content-Type", "application/json").
	JSON(map[string]any{"name": "kit"}).
	Timeout(3*time.Second).
	Do()
```

## 特性

- **重试**：指数退避，可配置重试状态码和错误类型
- **熔断器**：Closed → Open → Half-Open 状态机，防止级联故障
- **限流器**：通过 `RateLimiter` 接口注入
- **连接池**：可配置最大空闲连接数、每主机连接数
- **拦截器 / 中间件**：应用层 `Interceptor` + 传输层 `Middleware`
- **调试**：请求/响应头/体日志，敏感头脱敏，Body 大小限制
- **可观测**：通过 `Logger` / `Metrics` 接口注入（默认 no-op）
- **TLS / 代理**：完整支持

## 默认配置

| 配置项 | 默认值 |
|--------|--------|
| Timeout | 30s |
| User-Agent | go-kit-httpclient/1.0 |
| Retry MaxRetries | 2 |
| Retry InitialDelay | 100ms |
| Retry MaxDelay | 2s |
| Retry BackoffFactor | 2.0 |
| Retry RetryableStatus | 408, 429, 500, 502, 503, 504 |
| Pool MaxIdleConns | 100 |
| Pool MaxIdleConnsPerHost | 10 |
| Pool IdleConnTimeout | 90s |
| CircuitBreaker FailureThreshold | 5 |
| CircuitBreaker SuccessThreshold | 1 |
| CircuitBreaker Timeout | 30s |

## 核心 API

### 客户端管理

- `NewClient(options ...Option) *Client`
- `NewClientWithOptions(opts ClientOptions) *Client`
- `ConfigureDefault(opts ...Option)`
- `ResetDefault(opts ...Option) *Client`
- `GetDefaultClient() *Client`
- `SetDefaultClient(client *Client)`

### 全局 HTTP 方法

- `Get(ctx, url)` / `Post(ctx, url, body)` / `PostJSON(ctx, url, data)`
- `Put(ctx, url, body)` / `PutJSON(ctx, url, data)`
- `Delete(ctx, url)`
- `Patch(ctx, url, body)` / `PatchJSON(ctx, url, data)`
- `Do(ctx, method, url, body)`

### 配置选项

| 选项 | 说明 |
|------|------|
| `WithTimeout(d)` | 请求超时 |
| `WithBaseURL(url)` | 基础 URL |
| `WithHeaders(m)` | 默认请求头（覆盖） |
| `WithAdditionalHeaders(m)` | 追加请求头 |
| `WithRetry(*RetryConfig)` | 重试配置 |
| `WithCircuitBreaker(*CircuitBreakerConfig)` | 熔断器配置 |
| `WithPool(*PoolConfig)` | 连接池配置 |
| `WithTLS(*tls.Config)` | TLS 配置 |
| `WithProxy(fn)` | 代理函数 |
| `WithInterceptors(...)` | 应用层拦截器 |
| `WithMiddlewares(...)` | 传输层中间件 |
| `WithLogger(Logger)` | 日志接口 |
| `WithMetrics(Metrics)` | 指标接口 |
| `WithRateLimiter(RateLimiter)` | 限流器接口 |
| `WithDebug(*DebugConfig)` | 调试配置 |
| `WithHTTPClient(*http.Client)` | 注入自定义 Client |
| `WithTransport(http.RoundTripper)` | 注入自定义 Transport |

### 预定义中间件

- `RetryMiddleware(config RetryConfig) Middleware`
- `LoggingMiddleware(logger Logger) Middleware`
- `MetricsMiddleware(metrics Metrics) Middleware`

### Response 方法

- `JSON(v)` / `String()` / `Bytes()`
- `IsSuccess()` / `IsOK()` / `IsError()`
- `IsClientError()` / `IsServerError()`

## 注入自定义 http.Client

```go
transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
base := &http.Client{Transport: transport}
cli := httpclient.NewClient(httpclient.WithHTTPClient(base))

// 获取底层 Client 供框架使用
raw := cli.HTTPClient()
```

## 测试

```bash
go test ./httpclient -v
```
