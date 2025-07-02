# HTTP客户端 (pkg/httpclient)

功能强大的HTTP客户端，支持重试、熔断、调试、中间件等企业级特性。

## 🚀 特性

- ✅ 支持重试机制和指数退避
- ✅ 内置调试功能，详细记录请求/响应
- ✅ 支持中间件和拦截器
- ✅ 连接池管理和限流
- ✅ 熔断器支持
- ✅ 链式调用API
- ✅ 线程安全

## 📖 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "go-kit/pkg/httpclient"
)

func main() {
    // 创建默认客户端
    client := httpclient.NewClient()
    
    // 发送GET请求
    resp, err := client.Get("https://api.example.com/users")
    if err != nil {
        fmt.Printf("请求失败: %v\n", err)
        return
    }
    
    // 检查响应状态
    if resp.IsSuccess() {
        fmt.Printf("响应数据: %s\n", resp.String())
    } else {
        fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
    }
}
```

### 链式调用

```go
// 使用链式调用API
resp, err := client.NewRequest("POST", "https://api.example.com/users").
    Header("Content-Type", "application/json").
    JSON(map[string]interface{}{
        "name":  "张三",
        "email": "zhangsan@example.com",
    }).
    Timeout(10 * time.Second).
    Do()

if err != nil {
    log.Printf("请求失败: %v", err)
    return
}
```

## 🔧 API 参考

### 创建客户端

#### NewClient
创建默认配置的客户端

```go
client := httpclient.NewClient()
```

#### NewClientWithOptions
使用自定义选项创建客户端

```go
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Timeout: 30 * time.Second,
    BaseURL: "https://api.example.com",
    Headers: map[string]string{
        "User-Agent": "MyApp/1.0",
        "Authorization": "Bearer token",
    },
    Retry: &httpclient.RetryConfig{
        MaxRetries:      3,
        InitialDelay:    1 * time.Second,
        MaxDelay:        30 * time.Second,
        BackoffFactor:   2.0,
        RetryableStatus: []int{500, 502, 503, 504},
    },
    Debug: httpclient.DefaultDebugConfig(),
})
```

### 请求方法

#### 基本HTTP方法

```go
// GET请求
resp, err := client.Get("https://api.example.com/users")

// POST请求
resp, err := client.Post("https://api.example.com/users", strings.NewReader(`{"name":"张三"}`))

// POST JSON请求
resp, err := client.PostJSON("https://api.example.com/users", map[string]interface{}{
    "name":  "张三",
    "email": "zhangsan@example.com",
})

// PUT请求
resp, err := client.Put("https://api.example.com/users/1", strings.NewReader(`{"name":"李四"}`))

// DELETE请求
resp, err := client.Delete("https://api.example.com/users/1")

// PATCH请求
resp, err := client.Patch("https://api.example.com/users/1", strings.NewReader(`{"status":"active"}`))
```

#### 请求构建器

```go
// 创建请求构建器
req := client.NewRequest("POST", "https://api.example.com/users")

// 设置请求头
req.Header("Content-Type", "application/json")
req.Header("Authorization", "Bearer token")

// 设置请求体
req.JSON(map[string]interface{}{
    "name":  "张三",
    "email": "zhangsan@example.com",
})

// 设置超时
req.Timeout(10 * time.Second)

// 设置上下文
req.Context(context.Background())

// 执行请求
resp, err := req.Do()
```

### 响应处理

#### 响应方法

```go
// 检查响应状态
if resp.IsSuccess() {
    // 2xx状态码
}

if resp.IsOK() {
    // 2xx + 3xx状态码
}

if resp.IsError() {
    // 4xx + 5xx状态码
}

// 解析JSON响应
var user User
err = resp.JSON(&user)

// 获取响应字符串
body := resp.String()

// 获取响应字节
bytes := resp.Bytes()

// 获取错误信息
if resp.IsError() {
    errMsg := resp.Error()
    fmt.Printf("HTTP错误: %s\n", errMsg)
}
```

### 配置选项

#### RetryConfig - 重试配置

```go
retryConfig := &httpclient.RetryConfig{
    MaxRetries:      3,                    // 最大重试次数
    InitialDelay:    1 * time.Second,      // 初始延迟
    MaxDelay:        30 * time.Second,     // 最大延迟
    BackoffFactor:   2.0,                  // 退避因子
    RetryableStatus: []int{500, 502, 503}, // 可重试的状态码
    RetryableErrors: []error{              // 可重试的错误类型
        &url.Error{},
        &net.OpError{},
    },
}
```

#### DebugConfig - 调试配置

```go
debugConfig := &httpclient.DebugConfig{
    Enabled:            true,  // 启用调试
    LogRequestHeaders:  true,  // 记录请求头
    LogRequestBody:     true,  // 记录请求体
    LogResponseHeaders: true,  // 记录响应头
    LogResponseBody:    true,  // 记录响应体
    MaxBodySize:        10240, // 最大记录的Body大小（字节）
    SensitiveHeaders: []string{ // 敏感请求头列表
        "Authorization",
        "Cookie",
        "X-Api-Key",
    },
}
```

#### PoolConfig - 连接池配置

```go
poolConfig := &httpclient.PoolConfig{
    MaxIdleConns:        100,              // 最大空闲连接数
    MaxIdleConnsPerHost: 10,               // 每个主机最大空闲连接数
    MaxConnsPerHost:     100,              // 每个主机最大连接数
    IdleConnTimeout:     90 * time.Second, // 空闲连接超时时间
    DisableKeepAlives:   false,            // 禁用keep-alive
    DisableCompression:  false,            // 禁用压缩
}
```

## 🔧 高级功能

### 中间件系统

#### 内置中间件

```go
// 重试中间件
retryMiddleware := httpclient.RetryMiddleware(httpclient.RetryConfig{
    MaxRetries: 3,
    InitialDelay: 1 * time.Second,
})

// 日志中间件
loggingMiddleware := httpclient.LoggingMiddleware(logger)

// 指标中间件
metricsMiddleware := httpclient.MetricsMiddleware(metrics)

// 创建带中间件的客户端
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Middlewares: []httpclient.Middleware{
        retryMiddleware,
        loggingMiddleware,
        metricsMiddleware,
    },
})
```

#### 自定义中间件

```go
// 自定义认证中间件
func AuthMiddleware(token string) httpclient.Middleware {
    return func(next http.RoundTripper) http.RoundTripper {
        return &authTransport{
            next:  next,
            token: token,
        }
    }
}

type authTransport struct {
    next  http.RoundTripper
    token string
}

func (a *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+a.token)
    return a.next.RoundTrip(req)
}

// 使用自定义中间件
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Middlewares: []httpclient.Middleware{
        AuthMiddleware("your-token"),
    },
})
```

### 拦截器系统

```go
// 自定义拦截器
func LoggingInterceptor(logger Logger) httpclient.Interceptor {
    return func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
        start := time.Now()
        
        logger.Info("开始HTTP请求",
            "method", req.Method,
            "url", req.URL.String(),
        )
        
        resp, err := next(req)
        
        duration := time.Since(start)
        logger.Info("HTTP请求完成",
            "method", req.Method,
            "url", req.URL.String(),
            "status", resp.StatusCode,
            "duration", duration,
        )
        
        return resp, err
    }
}

// 使用拦截器
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Interceptors: []httpclient.Interceptor{
        LoggingInterceptor(logger),
    },
})
```

### 调试功能

#### 启用调试

```go
// 创建带调试功能的客户端
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Debug: &httpclient.DebugConfig{
        Enabled:         true,
        LogRequestBody:  true,
        LogResponseBody: true,
        MaxBodySize:     1024,
    },
})

// 或者动态启用调试
client.EnableDebug()
```

#### 调试输出示例

```
┌─────────────────────────────────────────────────────────────────────────────────
│ 🔍 HTTP REQUEST/RESPONSE DEBUG [GET https://api.example.com/users]
├─────────────────────────────────────────────────────────────────────────────────
│ 🚀 REQUEST:
│ Method: GET
│ URL: https://api.example.com/users
│ Headers: 
│         Content-Type: application/json
│         User-Agent: MyApp/1.0
│ Body: Empty
├─────────────────────────────────────────────────────────────────────────────────
│ 📥 RESPONSE:
│ Status: ✅ 200 OK
│ Duration: 245ms
│ Headers: 
│         Content-Type: application/json
│         Content-Length: 1234
│ Body: 
│         {
│           "users": [
│             {"id": 1, "name": "张三"},
│             {"id": 2, "name": "李四"}
│           ]
│         }
└─────────────────────────────────────────────────────────────────────────────────
```

## 🏗️ 最佳实践

### 1. 客户端配置

```go
// 生产环境配置
func createProductionClient() *httpclient.Client {
    return httpclient.NewClientWithOptions(httpclient.ClientOptions{
        Timeout: 30 * time.Second,
        BaseURL: "https://api.example.com",
        Headers: map[string]string{
            "User-Agent": "MyApp/1.0",
            "Accept":     "application/json",
        },
        Retry: &httpclient.RetryConfig{
            MaxRetries:      3,
            InitialDelay:    1 * time.Second,
            MaxDelay:        30 * time.Second,
            BackoffFactor:   2.0,
            RetryableStatus: []int{500, 502, 503, 504},
        },
        Pool: &httpclient.PoolConfig{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
        },
    })
}
```

### 2. 错误处理

```go
resp, err := client.Get("https://api.example.com/users")
if err != nil {
    // 检查网络错误
    if isNetworkError(err) {
        log.Printf("网络错误: %v", err)
        return
    }
    
    // 检查超时错误
    if isTimeoutError(err) {
        log.Printf("请求超时: %v", err)
        return
    }
    
    log.Printf("请求失败: %v", err)
    return
}

// 检查HTTP错误
if resp.IsError() {
    log.Printf("HTTP错误: %d - %s", resp.StatusCode, resp.String())
    return
}
```

### 3. 请求构建

```go
// 使用链式调用构建复杂请求
resp, err := client.NewRequest("POST", "/api/v1/users").
    Header("Content-Type", "application/json").
    Header("Authorization", "Bearer "+token).
    JSON(map[string]interface{}{
        "name":     user.Name,
        "email":    user.Email,
        "password": user.Password,
    }).
    Timeout(10 * time.Second).
    Context(ctx).
    Do()

if err != nil {
    return err
}

// 解析响应
var result CreateUserResponse
if err := resp.JSON(&result); err != nil {
    return err
}
```

### 4. 批量请求

```go
func fetchUsers(client *httpclient.Client, userIDs []int) ([]User, error) {
    var users []User
    var wg sync.WaitGroup
    var mu sync.Mutex
    errChan := make(chan error, len(userIDs))
    
    for _, id := range userIDs {
        wg.Add(1)
        go func(userID int) {
            defer wg.Done()
            
            resp, err := client.Get(fmt.Sprintf("/api/users/%d", userID))
            if err != nil {
                errChan <- err
                return
            }
            
            var user User
            if err := resp.JSON(&user); err != nil {
                errChan <- err
                return
            }
            
            mu.Lock()
            users = append(users, user)
            mu.Unlock()
        }(id)
    }
    
    wg.Wait()
    close(errChan)
    
    // 检查错误
    for err := range errChan {
        if err != nil {
            return nil, err
        }
    }
    
    return users, nil
}
```

### 5. 监控和指标

```go
// 创建指标收集器
type MetricsCollector struct {
    requestCounter   *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    errorCounter     *prometheus.CounterVec
}

func (m *MetricsCollector) IncCounter(name string, labels map[string]string) {
    m.requestCounter.With(labels).Inc()
}

func (m *MetricsCollector) AddHistogram(name string, value float64, labels map[string]string) {
    m.requestDuration.With(labels).Observe(value)
}

// 使用指标中间件
metrics := &MetricsCollector{...}
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Middlewares: []httpclient.Middleware{
        httpclient.MetricsMiddleware(metrics),
    },
})
```

## 🧪 测试

### 单元测试

```go
func TestHTTPClient(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message":"success"}`))
    }))
    defer server.Close()
    
    // 创建客户端
    client := httpclient.NewClient()
    
    // 发送请求
    resp, err := client.Get(server.URL)
    if err != nil {
        t.Fatalf("请求失败: %v", err)
    }
    
    // 验证响应
    if !resp.IsSuccess() {
        t.Errorf("期望成功响应，实际状态码: %d", resp.StatusCode)
    }
    
    var result map[string]interface{}
    if err := resp.JSON(&result); err != nil {
        t.Fatalf("解析JSON失败: %v", err)
    }
    
    if result["message"] != "success" {
        t.Errorf("期望message=success，实际=%v", result["message"])
    }
}
```

### 集成测试

```go
func TestHTTPClientWithRetry(t *testing.T) {
    attemptCount := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attemptCount++
        if attemptCount < 3 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    }))
    defer server.Close()
    
    client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
        Retry: &httpclient.RetryConfig{
            MaxRetries:      3,
            InitialDelay:    10 * time.Millisecond,
            MaxDelay:        100 * time.Millisecond,
            RetryableStatus: []int{500},
        },
    })
    
    resp, err := client.Get(server.URL)
    if err != nil {
        t.Fatalf("请求失败: %v", err)
    }
    
    if !resp.IsSuccess() {
        t.Errorf("期望成功响应，实际状态码: %d", resp.StatusCode)
    }
    
    if attemptCount != 3 {
        t.Errorf("期望重试3次，实际重试次数: %d", attemptCount)
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 连接超时

```go
// 增加超时时间
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Timeout: 60 * time.Second,
})

// 或者为特定请求设置超时
resp, err := client.NewRequest("GET", "https://slow-api.com").
    Timeout(30 * time.Second).
    Do()
```

#### 2. 重试不生效

```go
// 确保配置了可重试的状态码
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Retry: &httpclient.RetryConfig{
        MaxRetries:      3,
        RetryableStatus: []int{500, 502, 503, 504}, // 明确指定可重试状态码
    },
})
```

#### 3. 调试信息不显示

```go
// 确保启用了调试功能
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Debug: &httpclient.DebugConfig{
        Enabled:         true,
        LogRequestBody:  true,
        LogResponseBody: true,
    },
})

// 或者动态启用
client.EnableDebug()
```

### 性能优化

```go
// 1. 使用连接池
client := httpclient.NewClientWithOptions(httpclient.ClientOptions{
    Pool: &httpclient.PoolConfig{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
})

// 2. 复用客户端
// 不要为每个请求创建新的客户端
var globalClient *httpclient.Client

func init() {
    globalClient = httpclient.NewClient()
}

// 3. 使用上下文控制超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.NewRequest("GET", "https://api.example.com").
    Context(ctx).
    Do()
```

## 📚 相关链接

- [示例项目](./examples/httpclient-ctx/)
- [返回首页](../README.md) 