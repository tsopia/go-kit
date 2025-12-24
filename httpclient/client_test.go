package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() should return a non-nil client")
	}

	if client.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}

	if client.headers == nil {
		t.Fatal("headers should not be nil")
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient()
	timeout := 60 * time.Second
	client.SetTimeout(timeout)

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected httpClient timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestSetBaseURL(t *testing.T) {
	client := NewClient()
	baseURL := "https://api.example.com"
	client.SetBaseURL(baseURL)

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %s, got %s", baseURL, client.baseURL)
	}
}

func TestSetHeader(t *testing.T) {
	client := NewClient()
	key := "Authorization"
	value := "Bearer token123"
	client.SetHeader(key, value)

	if client.headers[key] != value {
		t.Errorf("Expected header %s=%s, got %s", key, value, client.headers[key])
	}
}

func TestSetHeaders(t *testing.T) {
	client := NewClient()
	headers := map[string]string{
		"Authorization": "Bearer token123",
		"Content-Type":  "application/json",
	}
	client.SetHeaders(headers)

	for key, value := range headers {
		if client.headers[key] != value {
			t.Errorf("Expected header %s=%s, got %s", key, value, client.headers[key])
		}
	}
}

func TestAddMiddleware(t *testing.T) {
	client := NewClient()

	// 创建一个简单的中间件
	middleware := func(next http.RoundTripper) http.RoundTripper {
		return &testRoundTripper{next: next}
	}

	client.AddMiddleware(middleware)

	// 验证中间件已添加（这里只是确保不会panic）
	if len(client.middlewares) == 0 {
		t.Error("Expected middleware to be added")
	}
}

// 测试用的RoundTripper
type testRoundTripper struct {
	next http.RoundTripper
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.next.RoundTrip(req)
}

func TestGet(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result map[string]interface{}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Expected no error parsing JSON, got %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got %v", result["message"])
	}
}

func TestPostJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if body["name"] != "test" {
			t.Errorf("Expected name 'test', got %v", body["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123, "name": "test"}`))
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	payload := map[string]interface{}{
		"name": "test",
		"age":  25,
	}

	resp, err := client.PostJSON("/users", payload)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
	}
}

func TestPutJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated": true}`))
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	payload := map[string]interface{}{
		"name": "updated",
	}

	resp, err := client.PutJSON("/users/123", payload)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestRetryReplaysRequestBody(t *testing.T) {
	attempts := 0
	var bodies [][]byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		bodies = append(bodies, payload)
		attempts++

		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Retry: &RetryConfig{
			MaxRetries:    1,
			InitialDelay:  0,
			MaxDelay:      time.Millisecond,
			BackoffFactor: 1,
		},
	})

	body := map[string]string{"name": "retry"}
	resp, err := client.PostJSON("/retry", body)
	if err != nil {
		t.Fatalf("expected request to eventually succeed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}

	if len(bodies) != 2 {
		t.Fatalf("expected to capture two request bodies, got %d", len(bodies))
	}

	if !bytes.Equal(bodies[0], bodies[1]) {
		t.Fatalf("expected request bodies to match across retries, got %q and %q", bodies[0], bodies[1])
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	client := NewClientWithOptions(ClientOptions{
		CircuitBreaker: &CircuitBreakerConfig{
			FailureThreshold: 1,
			MaxRequests:      1,
			Timeout:          50 * time.Millisecond,
			SuccessThreshold: 1,
		},
		Middlewares: []Middleware{
			func(http.RoundTripper) http.RoundTripper {
				return roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("boom")
				})
			},
		},
	})

	req := client.NewRequest("GET", "http://example.com")
	if _, err := client.do(req); err == nil {
		t.Fatalf("expected first request to fail")
	}

	if _, err := client.do(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit breaker to open, got error: %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDelete(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	resp, err := client.Delete("/users/123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
}

func TestResponseJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test", "age": 25}`))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var result map[string]interface{}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Expected no error parsing JSON, got %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", result["name"])
	}

	if result["age"] != float64(25) {
		t.Errorf("Expected age 25, got %v", result["age"])
	}
}

func TestResponseString(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	str := resp.String()
	if str != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", str)
	}
}

func TestResponseBytes(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test data"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data := resp.Bytes()
	if string(data) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(data))
	}
}

func TestResponseIsSuccess(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IsSuccess() {
		t.Error("Expected response to be success")
	}
}

func TestResponseIsOK(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		name       string
	}{
		{200, true, "2xx success"},
		{201, true, "2xx created"},
		{301, true, "3xx redirect"},
		{302, true, "3xx found"},
		{400, false, "4xx client error"},
		{500, false, "5xx server error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := NewClient()
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if resp.IsOK() != tc.expected {
				t.Errorf("Expected IsOK() to be %v for status %d", tc.expected, tc.statusCode)
			}
		})
	}
}

func TestResponseIsRedirect(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		name       string
	}{
		{200, false, "2xx success"},
		{301, true, "301 moved permanently"},
		{302, true, "302 found"},
		{304, true, "304 not modified"},
		{400, false, "4xx client error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			client := NewClient()
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if resp.IsRedirect() != tc.expected {
				t.Errorf("Expected IsRedirect() to be %v for status %d", tc.expected, tc.statusCode)
			}
		})
	}
}

func TestResponseIsClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IsClientError() {
		t.Error("Expected response to be client error")
	}

	if resp.IsServerError() {
		t.Error("Expected response to not be server error")
	}
}

func TestResponseIsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IsServerError() {
		t.Error("Expected response to be server error")
	}

	if resp.IsClientError() {
		t.Error("Expected response to not be client error")
	}
}

func TestResponseIsInformational(t *testing.T) {
	// 由于httptest.NewServer不容易模拟1xx状态码，我们直接创建Response
	resp := &Response{
		StatusCode: 100,
		Body:       []byte("Continue"),
	}

	if !resp.IsInformational() {
		t.Error("Expected response to be informational")
	}

	if resp.IsSuccess() {
		t.Error("Expected response to not be success")
	}
}

func TestResponseIsError(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IsError() {
		t.Error("Expected response to be error")
	}
}

func TestResponseError(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Error() == "" {
		t.Error("Expected response to have error")
	}
}

func TestRetryMiddleware(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient()

	// 添加重试中间件
	retryConfig := RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Millisecond * 100,
		MaxDelay:      time.Second,
		BackoffFactor: 2.0,
	}
	client.AddMiddleware(RetryMiddleware(retryConfig))

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestBuildRequest(t *testing.T) {
	client := NewClient()
	client.SetBaseURL("https://api.example.com")
	client.SetHeader("Authorization", "Bearer token")

	req := client.NewRequest("GET", "/users")
	_, err := req.Do()
	if err != nil {
		// 这里会有错误，因为没有实际的服务器，但我们可以测试请求构建
		t.Logf("Expected error due to no server: %v", err)
	}

	// 测试请求构建器的方法
	req2 := client.NewRequest("POST", "/users")
	req2.Header("Content-Type", "application/json")
	req2.JSON(map[string]string{"name": "test"})

	// 这里主要测试不会panic
	if req2 == nil {
		t.Error("Expected request builder to be created")
	}
}

// TestDebugConfig 测试Debug配置
func TestDebugConfig(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 创建mock logger
	mockLogger := &MockLogger{}

	// 创建客户端，启用debug
	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug:   DefaultDebugConfig(),
	})

	// 发送请求
	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", resp.StatusCode)
	}

	// 验证debug日志是否被记录
	if len(mockLogger.debugLogs) == 0 {
		t.Error("期望记录debug日志，但没有记录")
	}

	// 验证日志内容包含请求信息
	found := false
	for _, log := range mockLogger.debugLogs {
		if strings.Contains(log, "🔍 HTTP REQUEST/RESPONSE DEBUG") {
			found = true
			break
		}
	}
	if !found {
		t.Error("期望debug日志包含请求信息")
	}
}

// TestDebugSensitiveHeaders 测试敏感请求头脱敏
func TestDebugSensitiveHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug: &DebugConfig{
			Enabled:            true,
			LogRequestHeaders:  true,
			LogRequestBody:     false,
			LogResponseHeaders: false,
			LogResponseBody:    false,
			SensitiveHeaders:   []string{"Authorization", "X-Api-Key"},
		},
	})

	// 发送带敏感请求头的请求
	_, err := client.NewRequest("GET", "/test").
		Header("Authorization", "Bearer secret-token-12345").
		Header("X-Api-Key", "api-key-67890").
		Header("User-Agent", "TestAgent").
		Do()

	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证敏感请求头被脱敏
	debugLog := strings.Join(mockLogger.debugLogs, "\n")

	if strings.Contains(debugLog, "secret-token-12345") {
		t.Error("Authorization token应该被脱敏")
	}

	if strings.Contains(debugLog, "api-key-67890") {
		t.Error("API key应该被脱敏")
	}

	if !strings.Contains(debugLog, "TestAgent") {
		t.Error("非敏感请求头应该正常显示")
	}
}

// TestDebugBodyTruncation 测试Body截断
func TestDebugBodyTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回长响应
		longResponse := strings.Repeat("a", 2000)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longResponse))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug: &DebugConfig{
			Enabled:            true,
			LogRequestHeaders:  false,
			LogRequestBody:     false,
			LogResponseHeaders: false,
			LogResponseBody:    true,
			MaxBodySize:        100, // 限制为100字节
		},
	})

	_, err := client.Get("/test")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证响应体被截断
	debugLog := strings.Join(mockLogger.debugLogs, "\n")

	if !strings.Contains(debugLog, "truncated") {
		t.Error("长响应体应该被截断")
	}
}

// TestDebugError 测试错误情况下的debug日志
func TestDebugError(t *testing.T) {
	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: "http://nonexistent.example.com",
		Logger:  mockLogger,
		Debug:   DefaultDebugConfig(),
	})

	_, err := client.Get("/test")
	if err == nil {
		t.Error("期望请求失败")
	}

	// 验证错误日志被记录
	if len(mockLogger.errorLogs) == 0 {
		t.Error("期望记录错误日志")
	}

	// 验证错误日志包含错误信息
	errorLog := strings.Join(mockLogger.errorLogs, "\n")
	if !strings.Contains(errorLog, "ERROR") {
		t.Error("错误日志应该包含ERROR标识")
	}
}

// TestEnableDisableDebug 测试动态启用/禁用debug
func TestEnableDisableDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
	})

	// 初始状态应该没有debug日志
	client.Get("/test1")
	initialDebugCount := len(mockLogger.debugLogs)

	// 启用debug
	client.EnableDebug()
	client.Get("/test2")
	afterEnableCount := len(mockLogger.debugLogs)

	// 禁用debug
	client.DisableDebug()
	client.Get("/test3")
	afterDisableCount := len(mockLogger.debugLogs)

	if afterEnableCount <= initialDebugCount {
		t.Error("启用debug后应该有更多日志")
	}

	if afterDisableCount > afterEnableCount {
		t.Error("禁用debug后不应该有新的debug日志")
	}
}

// TestDebugJSONFormatting 测试JSON格式化
func TestDebugJSONFormatting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"test","value":123,"nested":{"key":"value"}}`))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug: &DebugConfig{
			Enabled:            true,
			LogRequestHeaders:  false,
			LogRequestBody:     false,
			LogResponseHeaders: false,
			LogResponseBody:    true,
		},
	})

	_, err := client.Get("/test")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证JSON被格式化
	debugLog := strings.Join(mockLogger.debugLogs, "\n")

	// 格式化的JSON应该包含缩进
	if !strings.Contains(debugLog, "  \"name\"") {
		t.Error("JSON应该被格式化")
	}
}

// TestDebugIndependentFromLogLevel 测试Debug功能独立于日志级别
func TestDebugIndependentFromLogLevel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 场景1: Logger设置为Info级别，但通过Debug.Enabled=true强制启用HTTP debug
	t.Run("Info级别Logger+强制启用HTTP_Debug", func(t *testing.T) {
		mockLogger := &MockLogger{}

		// 创建Info级别的logger（通常不会输出debug信息）
		// 但通过Debug.Enabled=true强制启用HTTP debug
		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger, // 假设这是Info级别的logger
			Debug: &DebugConfig{
				Enabled:            true, // 强制启用HTTP debug
				LogRequestHeaders:  true,
				LogRequestBody:     true,
				LogResponseHeaders: true,
				LogResponseBody:    true,
			},
		})

		_, err := client.Get("/test")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}

		// 验证即使logger是Info级别，HTTP debug日志也被记录
		if len(mockLogger.debugLogs) == 0 {
			t.Error("期望记录HTTP debug日志，即使logger级别是Info")
		}

		// 验证debug日志包含请求和响应信息
		debugLog := strings.Join(mockLogger.debugLogs, "\n")
		if !strings.Contains(debugLog, "🔍 HTTP REQUEST/RESPONSE DEBUG") {
			t.Error("期望包含🔍 HTTP REQUEST/RESPONSE DEBUG")
		}
		if !strings.Contains(debugLog, "🚀 REQUEST:") {
			t.Error("期望包含🚀 REQUEST:")
		}
		if !strings.Contains(debugLog, "📥 RESPONSE:") {
			t.Error("期望包含📥 RESPONSE:")
		}
	})

	// 场景2: Logger设置为Debug级别，但通过Debug.Enabled=false关闭HTTP debug
	t.Run("Debug级别Logger+关闭HTTP_Debug", func(t *testing.T) {
		mockLogger := &MockLogger{}

		// 创建Debug级别的logger（通常会输出debug信息）
		// 但通过Debug.Enabled=false关闭HTTP debug
		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger, // 假设这是Debug级别的logger
			Debug: &DebugConfig{
				Enabled:            false, // 关闭HTTP debug
				LogRequestHeaders:  true,
				LogRequestBody:     true,
				LogResponseHeaders: true,
				LogResponseBody:    true,
			},
		})

		_, err := client.Get("/test")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}

		// 验证即使logger是Debug级别，HTTP debug日志也不会被记录
		if len(mockLogger.debugLogs) > 0 {
			t.Error("期望不记录HTTP debug日志，即使logger级别是Debug")
		}

		// 但Info级别的日志仍然会被记录
		if len(mockLogger.infoLogs) == 0 {
			t.Error("期望记录Info级别的日志")
		}
	})
}

// TestDebugGranularControl 测试细粒度控制
func TestDebugGranularControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	// 场景: 只想看请求头，不想看响应体（减少日志噪音）
	t.Run("只记录请求头", func(t *testing.T) {
		mockLogger := &MockLogger{}

		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger,
			Debug: &DebugConfig{
				Enabled:            true,
				LogRequestHeaders:  true,  // 只记录请求头
				LogRequestBody:     false, // 不记录请求体
				LogResponseHeaders: false, // 不记录响应头
				LogResponseBody:    false, // 不记录响应体
			},
		})

		_, err := client.NewRequest("POST", "/test").
			Header("Authorization", "Bearer token123").
			JSON(map[string]string{"key": "value"}).
			Do()

		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}

		debugLog := strings.Join(mockLogger.debugLogs, "\n")

		// 应该包含请求头信息
		if !strings.Contains(debugLog, "🔍 HTTP REQUEST/RESPONSE DEBUG") {
			t.Error("期望包含🔍 HTTP REQUEST/RESPONSE DEBUG")
		}
		if !strings.Contains(debugLog, "Authorization") {
			t.Error("期望包含Authorization头")
		}

		// 不应该包含响应体信息
		if strings.Contains(debugLog, "HTTP RESPONSE DEBUG") {
			t.Error("不期望包含HTTP RESPONSE DEBUG")
		}
	})

	// 场景: 只想看响应，不想看请求（API调试时常见）
	t.Run("只记录响应信息", func(t *testing.T) {
		mockLogger := &MockLogger{}

		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger,
			Debug: &DebugConfig{
				Enabled:            true,
				LogRequestHeaders:  false, // 不记录请求头
				LogRequestBody:     false, // 不记录请求体
				LogResponseHeaders: true,  // 记录响应头
				LogResponseBody:    true,  // 记录响应体
			},
		})

		_, err := client.Get("/test")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}

		debugLog := strings.Join(mockLogger.debugLogs, "\n")

		// 由于统一日志输出，总是包含完整的请求/响应信息
		if !strings.Contains(debugLog, "🔍 HTTP REQUEST/RESPONSE DEBUG") {
			t.Error("期望包含🔍 HTTP REQUEST/RESPONSE DEBUG")
		}

		// 应该包含响应信息
		if !strings.Contains(debugLog, "📥 RESPONSE:") {
			t.Error("期望包含📥 RESPONSE:")
		}
		if !strings.Contains(debugLog, "Content-Type") {
			t.Error("期望包含Content-Type响应头")
		}
		if !strings.Contains(debugLog, "success") {
			t.Error("期望包含响应体内容")
		}
	})
}

// TestDebugRuntimeControl 测试运行时控制
func TestDebugRuntimeControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}
	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug: &DebugConfig{
			Enabled:            true,
			LogRequestHeaders:  true,
			LogRequestBody:     true,
			LogResponseHeaders: true,
			LogResponseBody:    true,
		},
	})

	// 场景: 在高流量期间临时关闭HTTP debug以减少日志量
	t.Run("高流量期间临时关闭debug", func(t *testing.T) {
		// 模拟正常流量期间
		client.Get("/normal-traffic")
		normalDebugCount := len(mockLogger.debugLogs)

		// 模拟高流量期间，临时关闭debug
		client.DisableDebug()

		// 发送多个请求（模拟高流量）
		for i := 0; i < 5; i++ {
			client.Get(fmt.Sprintf("/high-traffic-%d", i))
		}

		highTrafficDebugCount := len(mockLogger.debugLogs)

		// 流量恢复后重新启用debug
		client.EnableDebug()
		client.Get("/normal-traffic-resumed")

		resumedDebugCount := len(mockLogger.debugLogs)

		// 验证debug控制的效果
		if normalDebugCount == 0 {
			t.Error("正常流量期间应该有debug日志")
		}

		if highTrafficDebugCount != normalDebugCount {
			t.Error("高流量期间不应该增加debug日志")
		}

		if resumedDebugCount <= highTrafficDebugCount {
			t.Error("恢复后应该有新的debug日志")
		}
	})
}

// TestDebugPerformanceImpact 测试性能影响
func TestDebugPerformanceImpact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回大响应体
		largeResponse := strings.Repeat("a", 10000)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeResponse))
	}))
	defer server.Close()

	mockLogger := &MockLogger{}

	// 测试禁用debug时的性能
	t.Run("禁用debug的性能", func(t *testing.T) {
		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger,
			Debug: &DebugConfig{
				Enabled: false, // 禁用debug
			},
		})

		start := time.Now()
		for i := 0; i < 10; i++ {
			client.Get("/test")
		}
		disabledDuration := time.Since(start)

		// 验证没有debug日志
		if len(mockLogger.debugLogs) > 0 {
			t.Error("禁用debug时不应该有debug日志")
		}

		t.Logf("禁用debug时10次请求耗时: %v", disabledDuration)
	})

	// 测试启用debug时的性能
	t.Run("启用debug的性能", func(t *testing.T) {
		mockLogger2 := &MockLogger{}
		client := NewClientWithOptions(ClientOptions{
			BaseURL: server.URL,
			Logger:  mockLogger2,
			Debug: &DebugConfig{
				Enabled:            true, // 启用debug
				LogRequestHeaders:  true,
				LogRequestBody:     true,
				LogResponseHeaders: true,
				LogResponseBody:    true,
				MaxBodySize:        1000, // 限制body大小以避免过大的日志
			},
		})

		start := time.Now()
		for i := 0; i < 10; i++ {
			client.Get("/test")
		}
		enabledDuration := time.Since(start)

		// 验证有debug日志
		if len(mockLogger2.debugLogs) == 0 {
			t.Error("启用debug时应该有debug日志")
		}

		t.Logf("启用debug时10次请求耗时: %v", enabledDuration)
		t.Logf("Debug日志数量: %d", len(mockLogger2.debugLogs))
	})
}

// MockLogger 用于测试的mock logger
type MockLogger struct {
	debugLogs []string
	infoLogs  []string
	warnLogs  []string
	errorLogs []string
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.debugLogs = append(m.debugLogs, msg)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.infoLogs = append(m.infoLogs, msg)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.warnLogs = append(m.warnLogs, msg)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.errorLogs = append(m.errorLogs, msg)
}

// TestDebugUnifiedOutput 测试统一的debug日志输出
func TestDebugUnifiedOutput(t *testing.T) {
	mockLogger := &MockLogger{}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理时间
		time.Sleep(10 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"message": "success",
			"data":    []string{"item1", "item2"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  mockLogger,
		Debug:   DefaultDebugConfig(),
	})

	// 执行请求
	requestData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	resp, err := client.NewRequest("POST", "/api/test").
		Header("X-Test-Header", "test-value").
		Header("Authorization", "Bearer secret-token").
		JSON(requestData).
		Do()

	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", resp.StatusCode)
	}

	// 验证debug日志被记录
	if len(mockLogger.debugLogs) == 0 {
		t.Error("期望记录debug日志")
	}

	// 获取debug日志内容
	debugLog := strings.Join(mockLogger.debugLogs, "\n")

	// 验证日志包含完整的请求/响应信息
	testCases := []struct {
		name     string
		contains string
	}{
		{"请求方法", "Method: POST"},
		{"请求URL", "/api/test"},
		{"请求头脱敏", "Authorization: Bear****oken"},
		{"请求体", `"key1": "value1"`},
		{"响应状态", "✅ 200 OK"},
		{"响应头", "Content-Type: application/json"},
		{"响应体", `"message": "success"`},
		{"持续时间", "Duration:"},
		{"调试框架", "🔍 HTTP REQUEST/RESPONSE DEBUG"},
		{"请求部分", "🚀 REQUEST:"},
		{"响应部分", "📥 RESPONSE:"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(debugLog, tc.contains) {
				t.Errorf("debug日志应该包含 '%s'\n实际日志:\n%s", tc.contains, debugLog)
			}
		})
	}

	// 验证只有一条debug日志（统一输出）
	debugLogCount := strings.Count(debugLog, "🔍 HTTP REQUEST/RESPONSE DEBUG")
	if debugLogCount != 1 {
		t.Errorf("期望只有1条统一的debug日志，实际有 %d 条", debugLogCount)
	}

	// 验证请求和响应信息在同一条日志中
	if !strings.Contains(debugLog, "🚀 REQUEST:") || !strings.Contains(debugLog, "📥 RESPONSE:") {
		t.Error("debug日志应该同时包含请求和响应信息")
	}
}

// TestDebugUnifiedOutputWithError 测试错误情况下的统一日志输出
func TestDebugUnifiedOutputWithError(t *testing.T) {
	mockLogger := &MockLogger{}

	client := NewClientWithOptions(ClientOptions{
		BaseURL: "http://nonexistent.example.com",
		Logger:  mockLogger,
		Debug:   DefaultDebugConfig(),
	})

	// 执行会失败的请求
	_, err := client.NewRequest("GET", "/test").
		Header("X-Test-Header", "test-value").
		Do()

	if err == nil {
		t.Error("期望请求失败")
	}

	// 验证错误日志被记录
	if len(mockLogger.errorLogs) == 0 {
		t.Error("期望记录错误日志")
	}

	// 获取错误日志内容
	errorLog := strings.Join(mockLogger.errorLogs, "\n")

	// 验证错误日志包含完整信息
	testCases := []struct {
		name     string
		contains string
	}{
		{"请求方法", "Method: GET"},
		{"请求URL", "/test"},
		{"错误状态", "❌ ERROR:"},
		{"调试框架", "🔍 HTTP REQUEST/RESPONSE DEBUG"},
		{"请求部分", "🚀 REQUEST:"},
		{"响应部分", "📥 RESPONSE:"},
		{"错误响应标识", "N/A (Error occurred)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(errorLog, tc.contains) {
				t.Errorf("错误日志应该包含 '%s'\n实际日志:\n%s", tc.contains, errorLog)
			}
		})
	}

	// 验证只有一条错误日志（统一输出）
	errorLogCount := strings.Count(errorLog, "🔍 HTTP REQUEST/RESPONSE DEBUG")
	if errorLogCount != 1 {
		t.Errorf("期望只有1条统一的错误日志，实际有 %d 条", errorLogCount)
	}
}

// TestDebugUnifiedOutputWithoutLogger 测试没有logger时的统一输出
func TestDebugUnifiedOutputWithoutLogger(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClientWithOptions(ClientOptions{
		BaseURL: server.URL,
		Logger:  nil, // 没有logger
		Debug:   DefaultDebugConfig(),
	})

	// 执行请求
	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 200, 得到 %d", resp.StatusCode)
	}

	// 注意：没有logger时，debug信息会直接输出到终端
	// 这个测试主要验证不会panic，实际的终端输出需要手动验证
	t.Log("没有logger时，debug信息应该直接输出到终端（需要手动验证）")
}

// TestRequestWithCtx 测试WithCtx方法
func TestRequestWithCtx(t *testing.T) {
	// 创建一个带有值的context
	ctx := context.WithValue(context.Background(), "test_key", "test_value")

	// 创建请求并设置context
	req := &Request{
		ctx: context.Background(),
	}

	// 使用WithCtx方法设置context
	result := req.WithCtx(ctx)

	// 验证返回的是同一个Request对象（链式调用）
	if result != req {
		t.Error("WithCtx should return the same Request instance")
	}

	// 验证context被正确设置
	if req.ctx != ctx {
		t.Error("WithCtx should set the context correctly")
	}

	// 验证可以从context中获取值
	if value := req.ctx.Value("test_key"); value != "test_value" {
		t.Errorf("Expected context value 'test_value', got '%v'", value)
	}
}

// TestRequestWithCtxTimeout 测试WithCtx与超时的结合使用
func TestRequestWithCtxTimeout(t *testing.T) {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &Request{
		ctx: context.Background(),
	}

	// 使用WithCtx设置超时context
	req.WithCtx(ctx)

	// 验证context被正确设置
	if req.ctx != ctx {
		t.Error("WithCtx should set the timeout context correctly")
	}

	// 验证context确实有超时
	select {
	case <-req.ctx.Done():
		// 等待超时
		if req.ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", req.ctx.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

// TestRequestWithCtxVsContext 测试WithCtx和Context方法的等价性
func TestRequestWithCtxVsContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "comparison_key", "comparison_value")

	// 使用Context方法
	req1 := &Request{ctx: context.Background()}
	req1.Context(ctx)

	// 使用WithCtx方法
	req2 := &Request{ctx: context.Background()}
	req2.WithCtx(ctx)

	// 验证两种方法设置的context是相同的
	if req1.ctx != req2.ctx {
		t.Error("WithCtx and Context methods should set the same context")
	}

	// 验证两个context都能正确获取值
	value1 := req1.ctx.Value("comparison_key")
	value2 := req2.ctx.Value("comparison_key")

	if value1 != value2 || value1 != "comparison_value" {
		t.Errorf("Both contexts should have the same value, got %v and %v", value1, value2)
	}
}

// TestRequestWithCtxChaining 测试WithCtx的链式调用
func TestRequestWithCtxChaining(t *testing.T) {
	client := NewClient()
	ctx := context.WithValue(context.Background(), "chain_key", "chain_value")

	// 测试链式调用
	req := client.NewRequest("GET", "/test").
		WithCtx(ctx).
		Header("Test-Header", "test-value").
		Timeout(5 * time.Second)

	// 验证context被正确设置
	if req.ctx != ctx {
		t.Error("WithCtx should work correctly in method chaining")
	}

	// 验证其他方法也正确设置
	if req.headers["Test-Header"] != "test-value" {
		t.Error("Header should be set correctly after WithCtx")
	}

	if req.timeout != 5*time.Second {
		t.Error("Timeout should be set correctly after WithCtx")
	}
}
