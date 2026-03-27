package httpclient

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	RetryableStatus []int         // 可重试的状态码
	RetryableErrors []error       // 可重试的错误类型
}

// DebugConfig Debug配置
type DebugConfig struct {
	Enabled            bool     // 是否启用Debug
	LogRequestHeaders  bool     // 是否记录请求头
	LogRequestBody     bool     // 是否记录请求体
	LogResponseHeaders bool     // 是否记录响应头
	LogResponseBody    bool     // 是否记录响应体
	MaxBodySize        int      // 最大记录的Body大小（字节），0表示不限制
	SensitiveHeaders   []string // 敏感请求头列表，将被脱敏
}

// DefaultDebugConfig 默认Debug配置
func DefaultDebugConfig() *DebugConfig {
	return &DebugConfig{
		Enabled:            true,
		LogRequestHeaders:  true,
		LogRequestBody:     true,
		LogResponseHeaders: true,
		LogResponseBody:    true,
		MaxBodySize:        1024 * 10, // 10KB
		SensitiveHeaders: []string{
			"Authorization",
			"Cookie",
			"Set-Cookie",
			"X-Api-Key",
			"X-Auth-Token",
			"Bearer",
		},
	}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	MaxRequests      uint32        // 半开状态最大请求数
	Interval         time.Duration // 统计时间窗口
	Timeout          time.Duration // 熔断超时时间
	FailureThreshold uint32        // 失败阈值
	SuccessThreshold uint32        // 成功阈值
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxIdleConns        int           // 最大空闲连接数
	MaxIdleConnsPerHost int           // 每个主机最大空闲连接数
	MaxConnsPerHost     int           // 每个主机最大连接数
	IdleConnTimeout     time.Duration // 空闲连接超时时间
	DisableKeepAlives   bool          // 禁用keep-alive
	DisableCompression  bool          // 禁用压缩
}

// ClientOptions HTTP客户端选项
type ClientOptions struct {
	Timeout        time.Duration                         // 超时时间
	BaseURL        string                                // 基础URL
	Headers        map[string]string                     // 默认请求头
	UserAgent      string                                // 用户代理
	Cookies        []*http.Cookie                        // 默认Cookie
	Retry          *RetryConfig                          // 重试配置
	CircuitBreaker *CircuitBreakerConfig                 // 熔断器配置
	Pool           *PoolConfig                           // 连接池配置
	TLS            *tls.Config                           // TLS配置
	Proxy          func(*http.Request) (*url.URL, error) // 代理函数
	Interceptors   []Interceptor                         // 拦截器
	Middlewares    []Middleware                          // 中间件
	Logger         Logger                                // 日志记录器
	Metrics        Metrics                               // 指标收集器
	RateLimiter    RateLimiter                           // 限流器
	Debug          *DebugConfig                          // Debug配置
	HTTPClient     *http.Client                          // 自定义HTTP Client
}

// Interceptor HTTP拦截器
type Interceptor func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)

// Middleware HTTP中间件函数类型
type Middleware func(next http.RoundTripper) http.RoundTripper

// Logger 日志接口
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...interface{})
	Info(ctx context.Context, msg string, fields ...interface{})
	Warn(ctx context.Context, msg string, fields ...interface{})
	Error(ctx context.Context, msg string, fields ...interface{})
}

// Metrics 指标接口
type Metrics interface {
	IncCounter(name string, labels map[string]string)
	AddHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
}

// Option 选项函数
type Option func(*ClientOptions)

// DefaultOptions 默认配置
func DefaultOptions() ClientOptions {
	debug := DefaultDebugConfig()
	debug.Enabled = false

	return ClientOptions{
		Timeout:   30 * time.Second,
		UserAgent: "go-kit-httpclient/1.0",
		Retry: &RetryConfig{
			MaxRetries:    2,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      2 * time.Second,
			BackoffFactor: 2.0,
			RetryableStatus: []int{
				408, 429, 500, 502, 503, 504,
			},
		},
		Pool: &PoolConfig{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     90 * time.Second,
		},
		CircuitBreaker: &CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 1,
			MaxRequests:      1,
			Timeout:          30 * time.Second,
		},
		Debug: debug,
	}
}

func applyOptions(options ...Option) ClientOptions {
	base := DefaultOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&base)
		}
	}
	return cloneOptions(base)
}

func cloneOptions(opts ClientOptions) ClientOptions {
	copyHeaders := func(in map[string]string) map[string]string {
		if in == nil {
			return nil
		}
		out := make(map[string]string, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out
	}

	copyCookies := func(in []*http.Cookie) []*http.Cookie {
		if in == nil {
			return nil
		}
		out := make([]*http.Cookie, len(in))
		for i, c := range in {
			if c == nil {
				continue
			}
			clone := *c
			out[i] = &clone
		}
		return out
	}

	copyInterceptors := func(in []Interceptor) []Interceptor {
		if in == nil {
			return nil
		}
		out := make([]Interceptor, len(in))
		copy(out, in)
		return out
	}

	copyMiddlewares := func(in []Middleware) []Middleware {
		if in == nil {
			return nil
		}
		out := make([]Middleware, len(in))
		copy(out, in)
		return out
	}

	copyDebug := func(in *DebugConfig) *DebugConfig {
		if in == nil {
			return nil
		}
		out := *in
		if in.SensitiveHeaders != nil {
			out.SensitiveHeaders = append([]string{}, in.SensitiveHeaders...)
		}
		return &out
	}

	return ClientOptions{
		Timeout:        opts.Timeout,
		BaseURL:        opts.BaseURL,
		Headers:        copyHeaders(opts.Headers),
		UserAgent:      opts.UserAgent,
		Cookies:        copyCookies(opts.Cookies),
		Retry:          opts.Retry,
		CircuitBreaker: opts.CircuitBreaker,
		Pool:           opts.Pool,
		TLS:            opts.TLS,
		Proxy:          opts.Proxy,
		Interceptors:   copyInterceptors(opts.Interceptors),
		Middlewares:    copyMiddlewares(opts.Middlewares),
		Logger:         opts.Logger,
		Metrics:        opts.Metrics,
		RateLimiter:    opts.RateLimiter,
		Debug:          copyDebug(opts.Debug),
		HTTPClient:     opts.HTTPClient,
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(opts *ClientOptions) {
		opts.Timeout = timeout
	}
}

// WithBaseURL 设置基础URL
func WithBaseURL(baseURL string) Option {
	return func(opts *ClientOptions) {
		opts.BaseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithHeaders 覆盖默认请求头
func WithHeaders(headers map[string]string) Option {
	return func(opts *ClientOptions) {
		opts.Headers = headers
	}
}

// WithAdditionalHeaders 追加请求头
func WithAdditionalHeaders(headers map[string]string) Option {
	return func(opts *ClientOptions) {
		if opts.Headers == nil {
			opts.Headers = map[string]string{}
		}
		for k, v := range headers {
			opts.Headers[k] = v
		}
	}
}

// WithCookies 设置默认Cookie
func WithCookies(cookies []*http.Cookie) Option {
	return func(opts *ClientOptions) {
		opts.Cookies = cookies
	}
}

// WithUserAgent 设置User-Agent
func WithUserAgent(ua string) Option {
	return func(opts *ClientOptions) {
		opts.UserAgent = ua
	}
}

// WithRetry 设置重试配置
func WithRetry(cfg *RetryConfig) Option {
	return func(opts *ClientOptions) {
		opts.Retry = cfg
	}
}

// WithCircuitBreaker 设置熔断配置
func WithCircuitBreaker(cfg *CircuitBreakerConfig) Option {
	return func(opts *ClientOptions) {
		opts.CircuitBreaker = cfg
	}
}

// WithPool 设置连接池配置
func WithPool(cfg *PoolConfig) Option {
	return func(opts *ClientOptions) {
		opts.Pool = cfg
	}
}

// WithTLS 设置TLS配置
func WithTLS(cfg *tls.Config) Option {
	return func(opts *ClientOptions) {
		opts.TLS = cfg
	}
}

// WithProxy 设置代理
func WithProxy(proxy func(*http.Request) (*url.URL, error)) Option {
	return func(opts *ClientOptions) {
		opts.Proxy = proxy
	}
}

// WithInterceptors 设置拦截器
func WithInterceptors(interceptors ...Interceptor) Option {
	return func(opts *ClientOptions) {
		opts.Interceptors = append(opts.Interceptors, interceptors...)
	}
}

// WithMiddlewares 设置中间件
func WithMiddlewares(middlewares ...Middleware) Option {
	return func(opts *ClientOptions) {
		opts.Middlewares = append(opts.Middlewares, middlewares...)
	}
}

// WithLogger 设置日志器
func WithLogger(logger Logger) Option {
	return func(opts *ClientOptions) {
		opts.Logger = logger
	}
}

// WithMetrics 设置指标收集
func WithMetrics(metrics Metrics) Option {
	return func(opts *ClientOptions) {
		opts.Metrics = metrics
	}
}

// WithRateLimiter 设置限流器
func WithRateLimiter(rateLimiter RateLimiter) Option {
	return func(opts *ClientOptions) {
		opts.RateLimiter = rateLimiter
	}
}

// WithDebug 设置调试配置
func WithDebug(debug *DebugConfig) Option {
	return func(opts *ClientOptions) {
		opts.Debug = debug
	}
}

// WithHTTPClient 注入自定义 http.Client
func WithHTTPClient(client *http.Client) Option {
	return func(opts *ClientOptions) {
		opts.HTTPClient = client
	}
}

// WithTransport 注入自定义 RoundTripper
func WithTransport(transport http.RoundTripper) Option {
	return func(opts *ClientOptions) {
		if opts.HTTPClient == nil {
			opts.HTTPClient = &http.Client{}
		}
		opts.HTTPClient.Transport = transport
	}
}
