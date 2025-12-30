# HTTP客户端 (httpclient)

功能强大的 HTTP 客户端，支持重试、熔断、调试、中间件等企业级特性，默认开箱即用，亦可按需覆盖配置或注入自定义 `http.Client`/`Transport`。

## 🚀 特性

- ✅ 直接导入包即可使用全局 `Get/Post/...` 方法（默认超时 30s）
- ✅ 函数式 `With*` 选项，后传入的配置覆盖默认值
- ✅ 支持重试、熔断、连接池、TLS/代理、中间件/拦截器
- ✅ Debug/日志/指标/限流均可通过接口注入（默认 no-op）
- ✅ 链式请求构建器，涵盖常见 HTTP 方法
- ✅ 线程安全的默认客户端生命周期：`GetDefaultClient/ResetDefault/SetDefaultClient`

## 🏁 快速开始

### 1）默认即用（全局方法）
```go
import "github.com/tsopia/go-kit/httpclient"

resp, err := httpclient.Get("https://api.example.com/ping")
if err != nil {
    // 处理错误
}
fmt.Println(resp.StatusCode, resp.String())
```

### 2）创建独立客户端并覆盖配置
```go
import (
    "time"

    "github.com/tsopia/go-kit/httpclient"
)

cli := httpclient.NewClient(
    httpclient.WithBaseURL("https://api.example.com"),
    httpclient.WithTimeout(5*time.Second),
    httpclient.WithExtraHeaders(map[string]string{"X-Trace": "demo"}),
    httpclient.WithRetry(&httpclient.RetryConfig{MaxRetries: 2}),
)

resp, err := cli.PostJSON("/users", map[string]string{"name": "Go"})
```

### 3）重置全局默认客户端
```go
// 替换包级默认客户端（线程安全）
httpclient.ResetDefault(
    httpclient.WithBaseURL("https://api.example.com"),
    httpclient.WithTimeout(2*time.Second),
)

resp, _ := httpclient.Get("/health")
```

### 4）注入自定义 http.Client / Transport
```go
transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
base := &http.Client{Transport: transport}
cli := httpclient.NewClient(httpclient.WithHTTPClient(base))
// 或者仅注入 Transport
cli = httpclient.NewClient(httpclient.WithTransport(transport))

// 需要裸 http.Client 的框架可以使用
raw := cli.HTTPClient()
_ = raw.Transport // 复用同一传输层
```

### 5）链式请求构建与上下文
```go
resp, err := cli.NewRequest(http.MethodPost, "/users").
    Header("Content-Type", "application/json").
    JSON(map[string]any{"name": "kit"}).
    Timeout(3*time.Second).
    Context(ctx).
    Do()
```

## ⚙️ 默认配置（可通过 With* 覆盖）
- 超时：`30s`
- User-Agent：`go-kit-httpclient/1.0`
- 重试：`MaxRetries=2`，`InitialDelay=100ms`，`MaxDelay=2s`，`BackoffFactor=2.0`，重试状态码 `408/429/500/502/503/504`
- 连接池：`MaxIdleConns=100`，`MaxIdleConnsPerHost=10`，`IdleConnTimeout=90s`
- 熔断：`FailureThreshold=5`，`SuccessThreshold=1`，`MaxRequests=1`，`Timeout=30s`
- Debug：默认关闭（开启后自动脱敏常见敏感头，Body 记录上限 10KB）
- Logger/Metrics/RateLimiter：默认 no-op

## 🔧 配置选项速查
常用函数式选项（后传入的覆盖前者）：
- `WithTimeout(duration)`
- `WithBaseURL(url)` / `WithHeaders(map)` / `WithExtraHeaders(map)` / `WithCookies([]*http.Cookie)` / `WithUserAgent(string)`
- `WithRetry(*RetryConfig)` / `WithCircuitBreaker(*CircuitBreakerConfig)`
- `WithPool(*PoolConfig)` / `WithTLS(*tls.Config)` / `WithProxy(func(*http.Request) (*url.URL, error))`
- `WithInterceptors(...Interceptor)` / `WithMiddlewares(...Middleware)`
- `WithLogger(Logger)` / `WithMetrics(Metrics)` / `WithRateLimiter(RateLimiter)`
- `WithDebug(*DebugConfig)` / `WithHTTPClient(*http.Client)` / `WithTransport(http.RoundTripper)`

## 📚 更多示例
- 上下文版全局方法：`GetContext/PostContext/...`
- 释放连接：`client.HTTPClient().CloseIdleConnections()`
- 重置默认客户端：`httpclient.ResetDefault(opts...)`；获取当前默认：`httpclient.GetDefaultClient()`

> 旧接口 `NewClientWithOptions` 依旧可用，但推荐使用 `NewClient(With...)` 的函数式写法以获得明确的覆盖顺序。
