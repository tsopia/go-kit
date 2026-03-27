package httpclient

import (
	"bytes"
	"context"
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
	"time"
)

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

// ==================== Context-first Client 方法 ====================

// Get 发送GET请求
func (c *Client) Get(ctx context.Context, url string) (*Response, error) {
	return c.NewRequest(http.MethodGet, url).Context(ctx).Do()
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.NewRequest(http.MethodPost, url).Context(ctx).Body(body).Do()
}

// PostJSON 发送JSON POST请求
func (c *Client) PostJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return c.NewRequest(http.MethodPost, url).Context(ctx).JSON(data).Do()
}

// Put 发送PUT请求
func (c *Client) Put(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.NewRequest(http.MethodPut, url).Context(ctx).Body(body).Do()
}

// PutJSON 发送JSON PUT请求
func (c *Client) PutJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return c.NewRequest(http.MethodPut, url).Context(ctx).JSON(data).Do()
}

// Delete 发送DELETE请求
func (c *Client) Delete(ctx context.Context, url string) (*Response, error) {
	return c.NewRequest(http.MethodDelete, url).Context(ctx).Do()
}

// Patch 发送PATCH请求
func (c *Client) Patch(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.NewRequest(http.MethodPatch, url).Context(ctx).Body(body).Do()
}

// PatchJSON 发送JSON PATCH请求
func (c *Client) PatchJSON(ctx context.Context, url string, data interface{}) (*Response, error) {
	return c.NewRequest(http.MethodPatch, url).Context(ctx).JSON(data).Do()
}

// Do 直接执行HTTP请求
func (c *Client) Do(ctx context.Context, method, url string, body io.Reader) (*Response, error) {
	return c.NewRequest(method, url).Context(ctx).Body(body).Do()
}

// ==================== Client 配置方法 ====================

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

// ==================== 内部实现 ====================

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
			c.logCombinedDebugInfo(req.ctx, debugInfo)
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
		err = c.circuitBreaker.Execute(req.ctx, func() error {
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
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return nil, fmt.Errorf("读取响应体失败: %w; 关闭响应体失败: %v", readErr, closeErr)
		}
		return nil, fmt.Errorf("读取响应体失败: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("关闭响应体失败: %w", closeErr)
	}

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
		c.logger.Info(req.ctx, "HTTP请求完成",
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
		if !c.shouldRetry(resp, err) {
			return resp, err
		}
		if closeErr := closeRetryResponseBody(resp); closeErr != nil {
			return nil, closeErr
		}

		lastErr = err
		if attempt < c.retry.MaxRetries {
			delay := c.calculateDelay(attempt)
			if c.logger != nil {
				c.logger.Warn(req.Context(), "HTTP请求失败，准备重试",
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

			// 响应 context 取消，避免 sleep 阻塞
			select {
			case <-time.After(delay):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %w", c.retry.MaxRetries, lastErr)
}

func closeRetryResponseBody(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}

	_, readErr := io.Copy(io.Discard, resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		if closeErr != nil {
			return fmt.Errorf("丢弃重试响应体失败: %w; 关闭响应体失败: %v", readErr, closeErr)
		}
		return fmt.Errorf("丢弃重试响应体失败: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭重试响应体失败: %w", closeErr)
	}

	return nil
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

	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
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
		return netErr.Timeout()
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

// ==================== 预定义中间件 ====================

// middlewareTransport 中间件传输层
type middlewareTransport struct {
	next http.RoundTripper
}

func (mt *middlewareTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return mt.next.RoundTrip(req)
}

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
		if !rt.shouldRetry(resp, err) {
			return resp, err
		}
		lastErr = err
		if attempt < rt.config.MaxRetries {
			delay := rt.calculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
	}
	return nil, lastErr
}

func (rt *retryTransport) shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// 中间件中同样检查上下文相关错误不予重试
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		// 其他情况下可以认为默认错误值得一次重试尝试，或者你可以根据配置实现更细致的判断
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
		lt.logger.Error(req.Context(), "HTTP请求失败",
			"method", req.Method,
			"url", req.URL.String(),
			"duration", duration,
			"error", err,
		)
	} else {
		lt.logger.Info(req.Context(), "HTTP请求成功",
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

// ==================== Debug 信息收集 ====================

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
func (c *Client) logCombinedDebugInfo(ctx context.Context, debugInfo *httpDebugInfo) {

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
			c.logger.Error(ctx, combinedDebugInfo)
		} else {
			c.logger.Debug(ctx, combinedDebugInfo)
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
		if _, err := io.ReadFull(reader, content); err != nil {
			return nil, fmt.Errorf("读取字符串请求体失败: %w", err)
		}
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("重置字符串请求体失败: %w", err)
		}
		return content, nil
	}

	// 其他情况尝试读取
	if seeker, ok := body.(io.Seeker); ok {
		content, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("重置请求体失败: %w", err)
		}
		return content, nil
	}

	return nil, fmt.Errorf("无法安全读取body内容")
}
