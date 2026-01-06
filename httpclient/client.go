package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"io/ioutil"
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

// Option 选项函数
type Option func(*ClientOptions)

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

// WithExtraHeaders 追加请求头
func WithExtraHeaders(headers map[string]string) Option {
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

// Interceptor HTTP拦截器
type Interceptor func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error)

// Middleware HTTP中间件函数类型
type Middleware func(next http.RoundTripper) http.RoundTripper

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
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

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Execute(func() error) error
	State() string
}

var ErrCircuitOpen = errors.New("circuit breaker is open")
var ErrCircuitHalfOpenLimited = errors.New("circuit breaker half-open request limit reached")

type circuitState string

const (
	stateClosed   circuitState = "closed"
	stateOpen     circuitState = "open"
	stateHalfOpen circuitState = "half-open"
)

// statefulCircuitBreaker 具有基础状态机的熔断器实现
type statefulCircuitBreaker struct {
	config    CircuitBreakerConfig
	state     circuitState
	failures  uint32
	successes uint32
	openedAt  time.Time
	mu        sync.Mutex
}

// newCircuitBreaker 创建新的熔断器
func newCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 1
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &statefulCircuitBreaker{config: config, state: stateClosed}
}

// Execute 执行函数
func (cb *statefulCircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.currentStateLocked()

	if state == stateOpen {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	if state == stateHalfOpen && cb.successes >= cb.config.MaxRequests {
		cb.mu.Unlock()
		return ErrCircuitHalfOpenLimited
	}

	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.updateStateLocked(err == nil)

	return err
}

// State 获取熔断器状态
func (cb *statefulCircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return string(cb.currentStateLocked())
}

func (cb *statefulCircuitBreaker) currentStateLocked() circuitState {
	if cb.state == stateOpen && time.Since(cb.openedAt) >= cb.config.Timeout {
		cb.state = stateHalfOpen
		cb.failures = 0
		cb.successes = 0
	}

	return cb.state
}

func (cb *statefulCircuitBreaker) updateStateLocked(success bool) {
	switch cb.currentStateLocked() {
	case stateClosed:
		if success {
			cb.failures = 0
			return
		}

		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.tripLocked()
		}
	case stateHalfOpen:
		if success {
			cb.successes++
			if cb.successes >= cb.config.SuccessThreshold {
				cb.resetLocked()
			}
			return
		}

		cb.tripLocked()
	}
}

func (cb *statefulCircuitBreaker) tripLocked() {
	cb.state = stateOpen
	cb.openedAt = time.Now()
	cb.failures = 0
	cb.successes = 0
}

func (cb *statefulCircuitBreaker) resetLocked() {
	cb.state = stateClosed
	cb.failures = 0
	cb.successes = 0
}

// Client HTTP客户端
type Client struct {
	httpClient     *http.Client
	baseURL        string
	headers        map[string]string
	cookies        []*http.Cookie
	interceptors   []Interceptor
	middlewares    []Middleware
	retry          *RetryConfig
	circuitBreaker CircuitBreaker
	logger         Logger
	metrics        Metrics
	rateLimiter    RateLimiter
	mu             sync.RWMutex
	debugConfig    *DebugConfig
}

// Response HTTP响应
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Response   *http.Response
	Request    *http.Request
	Duration   time.Duration
}

// Request HTTP请求构建器
type Request struct {
	client  *Client
	method  string
	url     string
	headers map[string]string
	cookies []*http.Cookie
	body    io.Reader
	bodyErr error
	bodyRaw []byte
	bodySrc io.ReadSeeker
	timeout time.Duration
	ctx     context.Context
	retries int
}

// httpDebugInfo 调试信息结构体
type httpDebugInfo struct {
	// 请求信息
	RequestMethod  string
	RequestURL     string
	RequestHeaders string
	RequestBody    string

	// 响应信息
	ResponseStatus  string
	ResponseHeaders string
	ResponseBody    string

	// 错误信息
	Error string

	// 时间信息
	StartTime time.Time
	Duration  time.Duration
}

// NewClient 创建新的HTTP客户端，支持传入可选配置覆盖默认值
func NewClient(options ...Option) *Client {
	return newClientWithOptions(applyOptions(options...))
}

// NewClientWithOptions 根据选项创建HTTP客户端（兼容旧接口）
func NewClientWithOptions(opts ClientOptions) *Client {
	return newClientWithOptions(cloneOptions(opts))
}

func newClientWithOptions(opts ClientOptions) *Client {
	baseHTTPClient := opts.HTTPClient
	if baseHTTPClient == nil {
		baseHTTPClient = &http.Client{}
	}

	transport := baseHTTPClient.Transport
	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	// 应用连接池配置
	if opts.Pool != nil {
		if t, ok := transport.(*http.Transport); ok {
			if pool := opts.Pool; pool.MaxIdleConns != 0 {
				t.MaxIdleConns = pool.MaxIdleConns
			}
			if pool := opts.Pool; pool.MaxIdleConnsPerHost != 0 {
				t.MaxIdleConnsPerHost = pool.MaxIdleConnsPerHost
			}
			t.MaxConnsPerHost = opts.Pool.MaxConnsPerHost
			if pool := opts.Pool; pool.IdleConnTimeout != 0 {
				t.IdleConnTimeout = pool.IdleConnTimeout
			}
			t.DisableKeepAlives = opts.Pool.DisableKeepAlives
			t.DisableCompression = opts.Pool.DisableCompression
		}
	}

	// 应用TLS配置
	if opts.TLS != nil {
		if t, ok := transport.(*http.Transport); ok {
			t.TLSClientConfig = opts.TLS
		}
	}

	// 应用代理配置
	if opts.Proxy != nil {
		if t, ok := transport.(*http.Transport); ok {
			t.Proxy = opts.Proxy
		}
	}

	client := &Client{
		httpClient:   cloneHTTPClient(baseHTTPClient, opts.Timeout, transport),
		baseURL:      strings.TrimSuffix(opts.BaseURL, "/"),
		headers:      map[string]string{},
		cookies:      []*http.Cookie{},
		interceptors: []Interceptor{},
		middlewares:  []Middleware{},
		retry:        opts.Retry,
		logger:       opts.Logger,
		metrics:      opts.Metrics,
		rateLimiter:  opts.RateLimiter,
		debugConfig:  opts.Debug,
	}

	// 设置默认请求头
	client.headers["User-Agent"] = opts.UserAgent
	for key, value := range opts.Headers {
		client.headers[key] = value
	}

	// 设置Cookie
	if opts.Cookies != nil {
		client.cookies = append(client.cookies, opts.Cookies...)
	}

	// 添加拦截器
	if opts.Interceptors != nil {
		client.interceptors = append(client.interceptors, opts.Interceptors...)
	}

	// 添加中间件
	if len(opts.Middlewares) > 0 {
		client.middlewares = append(client.middlewares, opts.Middlewares...)
		client.rebuildTransport()
	}

	// 设置熔断器
	if opts.CircuitBreaker != nil {
		client.circuitBreaker = newCircuitBreaker(*opts.CircuitBreaker)
	}

	return client
}

func cloneHTTPClient(client *http.Client, timeout time.Duration, transport http.RoundTripper) *http.Client {
	if client == nil {
		return &http.Client{Timeout: timeout, Transport: transport}
	}

	clone := *client
	if timeout > 0 {
		clone.Timeout = timeout
	}
	if transport != nil {
		clone.Transport = transport
	}

	return &clone
}

// NewRequest 创建新的请求构建器
func (c *Client) NewRequest(method, url string) *Request {
	return &Request{
		client:  c,
		method:  method,
		url:     url,
		headers: make(map[string]string),
		ctx:     context.Background(),
	}
}

// Do 直接执行HTTP请求
func (c *Client) Do(ctx context.Context, method, url string, body io.Reader) (*Response, error) {
	return c.NewRequest(method, url).Context(ctx).Body(body).Do()
}

// HTTPClient 返回底层 http.Client
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// Transport 返回底层传输层
func (c *Client) Transport() http.RoundTripper {
	return c.httpClient.Transport
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient.Timeout = timeout
}

// SetBaseURL 设置基础URL
func (c *Client) SetBaseURL(baseURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimSuffix(baseURL, "/")
}

// SetHeader 设置请求头
func (c *Client) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[key] = value
}

// SetHeaders 批量设置请求头
func (c *Client) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, value := range headers {
		c.headers[key] = value
	}
}

// AddCookie 添加Cookie
func (c *Client) AddCookie(cookie *http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookies = append(c.cookies, cookie)
}

// AddInterceptor 添加拦截器
func (c *Client) AddInterceptor(interceptor Interceptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, interceptor)
}

// AddMiddleware 添加中间件
func (c *Client) AddMiddleware(middleware Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.middlewares = append(c.middlewares, middleware)

	// 重新构建传输层
	c.rebuildTransport()
}

// SetDebug 设置Debug配置
func (c *Client) SetDebug(debug *DebugConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debugConfig = debug
}

// EnableDebug 启用Debug模式
func (c *Client) EnableDebug() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.debugConfig == nil {
		c.debugConfig = DefaultDebugConfig()
	} else {
		c.debugConfig.Enabled = true
	}
}

// DisableDebug 禁用Debug模式
func (c *Client) DisableDebug() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.debugConfig != nil {
		c.debugConfig.Enabled = false
	}
}

// rebuildTransport 重新构建传输层
func (c *Client) rebuildTransport() {
	transport := c.httpClient.Transport

	// 找到原始传输层
	for {
		if middleware, ok := transport.(*middlewareTransport); ok {
			transport = middleware.next
		} else {
			break
		}
	}

	// 重新应用中间件
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		transport = c.middlewares[i](transport)
	}

	c.httpClient.Transport = transport
}

// buildRequest 构建HTTP请求
func (c *Client) buildRequest(req *Request) (*http.Request, error) {
	// 构建完整URL
	fullURL := req.url
	if !strings.HasPrefix(req.url, "http") {
		fullURL = c.baseURL + "/" + strings.TrimPrefix(req.url, "/")
	}

	if req.bodyErr != nil {
		return nil, req.bodyErr
	}

	bodyReader, getBody, err := req.prepareBody()
	if err != nil {
		return nil, err
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(req.ctx, req.method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	if getBody != nil {
		httpReq.GetBody = getBody
	}

	// 设置默认请求头
	c.mu.RLock()
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}
	c.mu.RUnlock()

	// 设置请求特定的请求头
	for key, value := range req.headers {
		httpReq.Header.Set(key, value)
	}

	// 设置Cookie
	c.mu.RLock()
	for _, cookie := range c.cookies {
		httpReq.AddCookie(cookie)
	}
	c.mu.RUnlock()

	for _, cookie := range req.cookies {
		httpReq.AddCookie(cookie)
	}

	return httpReq, nil
}

func (r *Request) prepareBody() (io.Reader, func() (io.ReadCloser, error), error) {
	switch {
	case r.bodySrc != nil:
		if _, err := r.bodySrc.Seek(0, io.SeekStart); err != nil {
			return nil, nil, fmt.Errorf("重置请求体失败: %w", err)
		}

		return r.bodySrc, func() (io.ReadCloser, error) {
			if _, err := r.bodySrc.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
			return ioutil.NopCloser(r.bodySrc), nil
		}, nil
	case r.bodyRaw != nil:
		return bytes.NewReader(r.bodyRaw), func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(r.bodyRaw)), nil
		}, nil
	case r.body != nil:
		return r.body, nil, nil
	default:
		return nil, nil, nil
	}
}

// do 执行HTTP请求
func (c *Client) do(req *Request) (*Response, error) {
	start := time.Now()

	// 应用限流
	if c.rateLimiter != nil {
		if !c.rateLimiter.Allow() {
			if err := c.rateLimiter.Wait(req.ctx); err != nil {
				return nil, fmt.Errorf("限流等待失败: %w", err)
			}
		}
	}

	// 构建HTTP请求
	httpReq, err := c.buildRequest(req)
	if err != nil {
		return nil, err
	}

	// Debug: 初始化调试信息收集
	var debugInfo *httpDebugInfo
	if c.debugConfig != nil && c.debugConfig.Enabled {
		debugInfo = &httpDebugInfo{
			RequestMethod: req.method,
			RequestURL:    req.url,
			StartTime:     start,
		}

		// 收集请求信息
		c.collectRequestDebugInfo(debugInfo, httpReq, req)

		// 使用defer确保在函数返回时输出完整的调试信息
		defer func() {
			debugInfo.Duration = time.Since(debugInfo.StartTime)
			c.logCombinedDebugInfo(debugInfo)
		}()
	}

	// 记录请求指标
	if c.metrics != nil {
		c.metrics.IncCounter("http_requests_total", map[string]string{
			"method": req.method,
			"url":    req.url,
		})
	}

	// 执行请求
	var resp *http.Response
	if c.circuitBreaker != nil {
		err = c.circuitBreaker.Execute(func() error {
			resp, err = c.executeRequest(httpReq)
			return err
		})
	} else {
		resp, err = c.executeRequest(httpReq)
	}

	duration := time.Since(start)

	// 记录响应指标
	if c.metrics != nil {
		labels := map[string]string{
			"method": req.method,
			"url":    req.url,
		}
		if resp != nil {
			labels["status"] = fmt.Sprintf("%d", resp.StatusCode)
		}
		c.metrics.AddHistogram("http_request_duration_seconds", duration.Seconds(), labels)
	}

	if err != nil {
		// Debug: 记录错误信息到debugInfo
		if debugInfo != nil {
			debugInfo.Error = err.Error()
		}

		// 记录错误指标
		if c.metrics != nil {
			c.metrics.IncCounter("http_request_errors_total", map[string]string{
				"method": req.method,
				"url":    req.url,
				"error":  err.Error(),
			})
		}
		return nil, err
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}
	resp.Body.Close()

	response := &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       body,
		Response:   resp,
		Request:    httpReq,
		Duration:   duration,
	}

	// Debug: 收集响应信息到debugInfo
	if debugInfo != nil {
		c.collectResponseDebugInfo(debugInfo, response)
	}

	// 记录日志
	if c.logger != nil {
		c.logger.Info("HTTP请求完成",
			"method", req.method,
			"url", req.url,
			"status", resp.StatusCode,
			"duration", duration,
		)
	} else {
		// 没有logger时直接输出到终端
		fmt.Printf("[INFO] HTTP请求完成 - Method: %s, URL: %s, Status: %d, Duration: %v\n",
			req.method, req.url, resp.StatusCode, duration)
	}

	return response, nil
}

// executeRequest 执行HTTP请求（带重试）
func (c *Client) executeRequest(req *http.Request) (*http.Response, error) {
	if c.retry == nil {
		return c.executeWithInterceptors(req)
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		currentReq := req
		if attempt > 0 {
			currentReq = req.Clone(req.Context())
			switch {
			case req.GetBody != nil:
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("重建请求体失败: %w", err)
				}
				currentReq.Body = body
			case req.Body == nil:
				currentReq.Body = nil
			default:
				return nil, fmt.Errorf("请求体不可重放，无法执行重试")
			}
		}

		resp, err := c.executeWithInterceptors(currentReq)
		if err == nil && !c.shouldRetry(resp, err) {
			return resp, nil
		}

		lastErr = err
		if attempt < c.retry.MaxRetries {
			delay := c.calculateDelay(attempt)
			if c.logger != nil {
				c.logger.Warn("HTTP请求失败，准备重试",
					"attempt", attempt+1,
					"max_retries", c.retry.MaxRetries,
					"delay", delay,
					"error", err,
				)
			} else {
				// 没有logger时直接输出到终端
				fmt.Printf("[WARN] HTTP请求失败，准备重试 - Attempt: %d/%d, Delay: %v, Error: %v\n",
					attempt+1, c.retry.MaxRetries, delay, err)
			}
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %w", c.retry.MaxRetries, lastErr)
}

// executeWithInterceptors 使用拦截器执行请求
func (c *Client) executeWithInterceptors(req *http.Request) (*http.Response, error) {
	if len(c.interceptors) == 0 {
		return c.httpClient.Do(req)
	}

	var execute func(*http.Request) (*http.Response, error)
	execute = func(req *http.Request) (*http.Response, error) {
		return c.httpClient.Do(req)
	}

	// 从后往前应用拦截器
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		next := execute
		execute = func(req *http.Request) (*http.Response, error) {
			return interceptor(req, next)
		}
	}

	return execute(req)
}

// shouldRetry 判断是否应该重试
func (c *Client) shouldRetry(resp *http.Response, err error) bool {
	if c.retry == nil {
		return false
	}

	// 检查错误类型
	if err != nil {
		for _, retryableErr := range c.retry.RetryableErrors {
			if errors.Is(err, retryableErr) {
				return true
			}
		}
		// 默认网络错误可重试
		if isNetworkError(err) {
			return true
		}
	}

	// 检查状态码
	if resp != nil {
		if c.retry.RetryableStatus != nil {
			for _, status := range c.retry.RetryableStatus {
				if resp.StatusCode == status {
					return true
				}
			}
		} else if resp.StatusCode >= 500 {
			// 默认配置（未提供显式状态码列表）时，5xx 视为可重试
			return true
		}
	}

	return false
}

// calculateDelay 计算重试延迟
func (c *Client) calculateDelay(attempt int) time.Duration {
	if c.retry == nil {
		return time.Second
	}

	delay := c.retry.InitialDelay
	if c.retry.BackoffFactor > 1 {
		delay = time.Duration(float64(delay) * math.Pow(c.retry.BackoffFactor, float64(attempt)))
	}

	if delay > c.retry.MaxDelay {
		delay = c.retry.MaxDelay
	}

	return delay
}

// isNetworkError 判断是否为网络错误
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// 检查常见的网络错误类型
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// 检查URL错误
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isNetworkError(urlErr.Err)
	}

	// 检查其他网络相关错误
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "network is unreachable")
}

// Get 发送GET请求
func (c *Client) Get(url string) (*Response, error) {
	return c.NewRequest("GET", url).Do()
}

// Post 发送POST请求
func (c *Client) Post(url string, body io.Reader) (*Response, error) {
	return c.NewRequest("POST", url).Body(body).Do()
}

// PostJSON 发送JSON POST请求
func (c *Client) PostJSON(url string, data interface{}) (*Response, error) {
	return c.NewRequest("POST", url).JSON(data).Do()
}

// Put 发送PUT请求
func (c *Client) Put(url string, body io.Reader) (*Response, error) {
	return c.NewRequest("PUT", url).Body(body).Do()
}

// PutJSON 发送JSON PUT请求
func (c *Client) PutJSON(url string, data interface{}) (*Response, error) {
	return c.NewRequest("PUT", url).JSON(data).Do()
}

// Delete 发送DELETE请求
func (c *Client) Delete(url string) (*Response, error) {
	return c.NewRequest("DELETE", url).Do()
}

// Patch 发送PATCH请求
func (c *Client) Patch(url string, body io.Reader) (*Response, error) {
	return c.NewRequest("PATCH", url).Body(body).Do()
}

// PatchJSON 发送JSON PATCH请求
func (c *Client) PatchJSON(url string, data interface{}) (*Response, error) {
	return c.NewRequest("PATCH", url).JSON(data).Do()
}

// Request 请求构建器方法

// Header 设置请求头
func (r *Request) Header(key, value string) *Request {
	r.headers[key] = value
	return r
}

// Headers 批量设置请求头
func (r *Request) Headers(headers map[string]string) *Request {
	for key, value := range headers {
		r.headers[key] = value
	}
	return r
}

// Cookie 添加Cookie
func (r *Request) Cookie(cookie *http.Cookie) *Request {
	r.cookies = append(r.cookies, cookie)
	return r
}

// Body 设置请求体
func (r *Request) Body(body io.Reader) *Request {
	r.bodyErr = nil
	r.bodyRaw = nil
	r.bodySrc = nil

	if body == nil {
		r.body = nil
		return r
	}

	if seeker, ok := body.(io.ReadSeeker); ok {
		r.bodySrc = seeker
		r.body = seeker
		return r
	}

	data, err := ioutil.ReadAll(body)
	if err != nil {
		r.bodyErr = fmt.Errorf("读取请求体失败: %w", err)
		return r
	}

	r.bodyRaw = data
	r.body = bytes.NewReader(data)
	return r
}

// JSON 设置JSON请求体
func (r *Request) JSON(data interface{}) *Request {
	jsonData, err := json.Marshal(data)
	if err != nil {
		// 这里可以考虑返回错误，但为了链式调用的简洁性，暂时忽略
		return r
	}
	r.bodyErr = nil
	r.bodySrc = nil
	r.body = bytes.NewBuffer(jsonData)
	r.bodyRaw = jsonData
	r.headers["Content-Type"] = "application/json"
	return r
}

// Form 设置表单请求体
func (r *Request) Form(data url.Values) *Request {
	r.bodyErr = nil
	r.bodySrc = nil
	r.body = strings.NewReader(data.Encode())
	r.bodyRaw = []byte(data.Encode())
	r.headers["Content-Type"] = "application/x-www-form-urlencoded"
	return r
}

// Timeout 设置超时时间
func (r *Request) Timeout(timeout time.Duration) *Request {
	r.timeout = timeout
	return r
}

// Context 设置上下文
func (r *Request) Context(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// WithCtx 设置上下文 (Context方法的简洁版本)
func (r *Request) WithCtx(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Retries 设置重试次数
func (r *Request) Retries(retries int) *Request {
	r.retries = retries
	return r
}

// Do 执行请求
func (r *Request) Do() (*Response, error) {
	// 应用超时
	if r.timeout > 0 {
		ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
		defer cancel()
		r.ctx = ctx
	}

	return r.client.do(r)
}

// Response 响应方法

// JSON 解析响应为JSON
func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// String 获取响应字符串
func (r *Response) String() string {
	return string(r.Body)
}

// Bytes 获取响应字节
func (r *Response) Bytes() []byte {
	return r.Body
}

// IsSuccess 检查是否为成功响应 (仅2xx)
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsOK 检查是否为OK响应 (2xx + 3xx)
func (r *Response) IsOK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 400
}

// IsRedirect 检查是否为重定向响应
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode < 400
}

// IsClientError 检查是否为客户端错误
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError 检查是否为服务器错误
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}

// IsError 检查是否为错误响应 (4xx + 5xx)
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// IsInformational 检查是否为信息性响应
func (r *Response) IsInformational() bool {
	return r.StatusCode >= 100 && r.StatusCode < 200
}

// Error 获取错误信息
func (r *Response) Error() string {
	if r.IsError() {
		return fmt.Sprintf("HTTP %d: %s", r.StatusCode, r.String())
	}
	return ""
}

// middlewareTransport 中间件传输层
type middlewareTransport struct {
	next http.RoundTripper
}

func (mt *middlewareTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return mt.next.RoundTrip(req)
}

// 预定义的中间件

// RetryMiddleware 重试中间件
func RetryMiddleware(config RetryConfig) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return &retryTransport{
			next:   next,
			config: config,
		}
	}
}

type retryTransport struct {
	next   http.RoundTripper
	config RetryConfig
}

func (rt *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= rt.config.MaxRetries; attempt++ {
		resp, err := rt.next.RoundTrip(req)
		if err == nil && !rt.shouldRetry(resp, err) {
			return resp, nil
		}
		lastErr = err
		if attempt < rt.config.MaxRetries {
			delay := rt.calculateDelay(attempt)
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}

func (rt *retryTransport) shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp != nil {
		for _, status := range rt.config.RetryableStatus {
			if resp.StatusCode == status {
				return true
			}
		}
		return resp.StatusCode >= 500
	}
	return false
}

func (rt *retryTransport) calculateDelay(attempt int) time.Duration {
	delay := rt.config.InitialDelay
	if rt.config.BackoffFactor > 1 {
		delay = time.Duration(float64(delay) * math.Pow(rt.config.BackoffFactor, float64(attempt)))
	}
	if delay > rt.config.MaxDelay {
		delay = rt.config.MaxDelay
	}
	return delay
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger Logger) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return &loggingTransport{
			next:   next,
			logger: logger,
		}
	}
}

type loggingTransport struct {
	next   http.RoundTripper
	logger Logger
}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := lt.next.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		lt.logger.Error("HTTP请求失败",
			"method", req.Method,
			"url", req.URL.String(),
			"duration", duration,
			"error", err,
		)
	} else {
		lt.logger.Info("HTTP请求成功",
			"method", req.Method,
			"url", req.URL.String(),
			"status", resp.StatusCode,
			"duration", duration,
		)
	}

	return resp, err
}

// MetricsMiddleware 指标中间件
func MetricsMiddleware(metrics Metrics) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return &metricsTransport{
			next:    next,
			metrics: metrics,
		}
	}
}

type metricsTransport struct {
	next    http.RoundTripper
	metrics Metrics
}

func (mt *metricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := mt.next.RoundTrip(req)
	duration := time.Since(start)

	labels := map[string]string{
		"method": req.Method,
		"host":   req.URL.Host,
	}

	if resp != nil {
		labels["status"] = fmt.Sprintf("%d", resp.StatusCode)
	}

	mt.metrics.IncCounter("http_requests_total", labels)
	mt.metrics.AddHistogram("http_request_duration_seconds", duration.Seconds(), labels)

	if err != nil {
		labels["error"] = err.Error()
		mt.metrics.IncCounter("http_request_errors_total", labels)
	}

	return resp, err
}

// 全局客户端实例
var (
	defaultClientValue atomic.Value // *Client
	defaultClientOnce  sync.Once
	defaultClientMu    sync.Mutex
)

func getDefaultClient() *Client {
	if client, ok := defaultClientValue.Load().(*Client); ok {
		return client
	}

	defaultClientOnce.Do(func() {
		defaultClientValue.Store(NewClient())
	})

	if client, ok := defaultClientValue.Load().(*Client); ok {
		return client
	}

	return NewClient()
}

func ResetDefault(opts ...Option) *Client {
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()

	client := NewClient(opts...)
	defaultClientValue.Store(client)
	return client
}

func SetDefaultClient(client *Client) {
	if client == nil {
		return
	}
	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	defaultClientValue.Store(client)
}

func GetDefaultClient() *Client {
	return getDefaultClient()
}

// 全局函数
func Get(url string) (*Response, error) {
	return getDefaultClient().Get(url)
}

func Post(url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Post(url, body)
}

func PostJSON(url string, data interface{}) (*Response, error) {
	return getDefaultClient().PostJSON(url, data)
}

func Put(url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Put(url, body)
}

func PutJSON(url string, data interface{}) (*Response, error) {
	return getDefaultClient().PutJSON(url, data)
}

func Delete(url string) (*Response, error) {
	return getDefaultClient().Delete(url)
}

func Patch(url string, body io.Reader) (*Response, error) {
	return getDefaultClient().Patch(url, body)
}

func PatchJSON(url string, data interface{}) (*Response, error) {
	return getDefaultClient().PatchJSON(url, data)
}

func GetContext(ctx context.Context, url string) (*Response, error) {
	return getDefaultClient().NewRequest(http.MethodGet, url).Context(ctx).Do()
}

func PostContext(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().NewRequest(http.MethodPost, url).Context(ctx).Body(body).Do()
}

func PutContext(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().NewRequest(http.MethodPut, url).Context(ctx).Body(body).Do()
}

func DeleteContext(ctx context.Context, url string) (*Response, error) {
	return getDefaultClient().NewRequest(http.MethodDelete, url).Context(ctx).Do()
}

func PatchContext(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return getDefaultClient().NewRequest(http.MethodPatch, url).Context(ctx).Body(body).Do()
}

func SetTimeout(timeout time.Duration) {
	getDefaultClient().SetTimeout(timeout)
}

func SetBaseURL(baseURL string) {
	getDefaultClient().SetBaseURL(baseURL)
}

func SetHeader(key, value string) {
	getDefaultClient().SetHeader(key, value)
}

func SetHeaders(headers map[string]string) {
	getDefaultClient().SetHeaders(headers)
}

func DefaultClient() *Client {
	return getDefaultClient()
}

// collectRequestDebugInfo 收集请求调试信息
func (c *Client) collectRequestDebugInfo(debugInfo *httpDebugInfo, httpReq *http.Request, req *Request) {
	// 收集请求头信息
	if c.debugConfig.LogRequestHeaders {
		debugInfo.RequestHeaders = c.formatHeaders(httpReq.Header, true)
	}

	// 收集请求体信息
	if c.debugConfig.LogRequestBody && req.body != nil {
		if bodyBytes, err := c.readBodySafely(req.body); err == nil {
			debugInfo.RequestBody = c.formatBody(bodyBytes)
		}
	}
}

// collectResponseDebugInfo 收集响应调试信息
func (c *Client) collectResponseDebugInfo(debugInfo *httpDebugInfo, response *Response) {
	// 收集响应状态信息
	debugInfo.ResponseStatus = fmt.Sprintf("✅ %s", response.Status)

	// 收集响应头信息
	if c.debugConfig.LogResponseHeaders {
		debugInfo.ResponseHeaders = c.formatHeaders(response.Headers, false)
	}

	// 收集响应体信息
	if c.debugConfig.LogResponseBody {
		debugInfo.ResponseBody = c.formatBody(response.Body)
	}
}

// logCombinedDebugInfo 输出合并的调试信息
func (c *Client) logCombinedDebugInfo(debugInfo *httpDebugInfo) {

	// 检查是否有任何信息需要记录
	if !c.debugConfig.LogRequestHeaders && !c.debugConfig.LogRequestBody &&
		!c.debugConfig.LogResponseHeaders && !c.debugConfig.LogResponseBody {
		return
	}

	var statusInfo string
	var responseHeaders string
	var responseBody string

	if debugInfo.Error != "" {
		statusInfo = fmt.Sprintf("❌ ERROR: %v", debugInfo.Error)
		responseHeaders = "N/A (Error occurred)"
		responseBody = "N/A (Error occurred)"
	} else {
		statusInfo = debugInfo.ResponseStatus
		responseHeaders = debugInfo.ResponseHeaders
		responseBody = debugInfo.ResponseBody
	}

	// 构建完整的调试信息
	combinedDebugInfo := fmt.Sprintf(`
┌─────────────────────────────────────────────────────────────────────────────────
│ 🔍 HTTP REQUEST/RESPONSE DEBUG [%s %s]
├─────────────────────────────────────────────────────────────────────────────────
│ 🚀 REQUEST:
│ Method: %s
│ URL: %s
│ Headers: %s
│ Body: %s
├─────────────────────────────────────────────────────────────────────────────────
│ 📥 RESPONSE:
│ Status: %s
│ Duration: %v
│ Headers: %s
│ Body: %s
└─────────────────────────────────────────────────────────────────────────────────`,
		debugInfo.RequestMethod,
		debugInfo.RequestURL,
		debugInfo.RequestMethod,
		debugInfo.RequestURL,
		debugInfo.RequestHeaders,
		debugInfo.RequestBody,
		statusInfo,
		debugInfo.Duration,
		responseHeaders,
		responseBody,
	)

	// 根据是否有logger决定输出方式
	if c.logger != nil {
		if debugInfo.Error != "" {
			c.logger.Error(combinedDebugInfo)
		} else {
			c.logger.Debug(combinedDebugInfo)
		}
	} else {
		// 没有logger时直接输出到终端
		if debugInfo.Error != "" {
			fmt.Printf("[ERROR] %s\n", combinedDebugInfo)
		} else {
			fmt.Printf("[DEBUG] %s\n", combinedDebugInfo)
		}
	}
}

// formatHeaders 格式化请求头
func (c *Client) formatHeaders(headers http.Header, isRequest bool) string {
	if len(headers) == 0 {
		return "None"
	}

	var formatted []string
	for key, values := range headers {
		value := strings.Join(values, ", ")

		// 脱敏处理
		if c.isSensitiveHeader(key) {
			value = c.maskSensitiveValue(value)
		}

		formatted = append(formatted, fmt.Sprintf("%s: %s", key, value))
	}

	if len(formatted) > 5 {
		return fmt.Sprintf("\n│         %s\n│         ... (%d more headers)",
			strings.Join(formatted[:5], "\n│         "), len(formatted)-5)
	}

	return fmt.Sprintf("\n│         %s", strings.Join(formatted, "\n│         "))
}

// formatBody 格式化请求/响应体
func (c *Client) formatBody(body []byte) string {
	if len(body) == 0 {
		return "Empty"
	}

	// 限制body大小
	if c.debugConfig.MaxBodySize > 0 && len(body) > c.debugConfig.MaxBodySize {
		truncated := body[:c.debugConfig.MaxBodySize]
		return fmt.Sprintf("%s\n│         ... (truncated %d bytes)",
			c.formatBodyContent(truncated), len(body)-c.debugConfig.MaxBodySize)
	}

	return c.formatBodyContent(body)
}

// formatBodyContent 格式化body内容
func (c *Client) formatBodyContent(body []byte) string {
	content := string(body)

	// 检查是否是JSON
	if c.isJSON(content) {
		if formatted, err := c.formatJSON(content); err == nil {
			lines := strings.Split(formatted, "\n")
			if len(lines) > 10 {
				return fmt.Sprintf("\n│         %s\n│         ... (%d more lines)",
					strings.Join(lines[:10], "\n│         "), len(lines)-10)
			}
			return fmt.Sprintf("\n│         %s", strings.Join(lines, "\n│         "))
		}
	}

	// 普通文本处理
	lines := strings.Split(content, "\n")
	if len(lines) > 5 {
		return fmt.Sprintf("\n│         %s\n│         ... (%d more lines)",
			strings.Join(lines[:5], "\n│         "), len(lines)-5)
	}

	return fmt.Sprintf("\n│         %s", strings.Join(lines, "\n│         "))
}

// isSensitiveHeader 检查是否为敏感请求头
func (c *Client) isSensitiveHeader(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, sensitive := range c.debugConfig.SensitiveHeaders {
		if strings.ToLower(sensitive) == lowerKey {
			return true
		}
	}
	return false
}

// maskSensitiveValue 脱敏处理敏感值
func (c *Client) maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

// isJSON 检查内容是否为JSON
func (c *Client) isJSON(content string) bool {
	content = strings.TrimSpace(content)
	return (strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}")) ||
		(strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]"))
}

// formatJSON 格式化JSON内容
func (c *Client) formatJSON(content string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// readBodySafely 安全读取body内容
func (c *Client) readBodySafely(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	// 如果是字节缓冲区，直接读取
	if buf, ok := body.(*bytes.Buffer); ok {
		return buf.Bytes(), nil
	}

	// 如果是字符串读取器，直接读取
	if reader, ok := body.(*strings.Reader); ok {
		content := make([]byte, reader.Len())
		reader.Read(content)
		reader.Seek(0, 0) // 重置位置
		return content, nil
	}

	// 其他情况尝试读取
	if seeker, ok := body.(io.Seeker); ok {
		content, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		seeker.Seek(0, 0) // 重置位置
		return content, nil
	}

	return nil, fmt.Errorf("无法安全读取body内容")
}
