package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tsopia/go-kit/constants"

	"github.com/gin-gonic/gin"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil) // 使用默认配置
	if server == nil {
		t.Fatal("NewServer() should return a non-nil server")
	}

	if server.engine == nil {
		t.Fatal("engine should not be nil")
	}

	if server.config == nil {
		t.Fatal("config should not be nil")
	}
}

func TestNewServerWithConfig(t *testing.T) {
	config := &Config{
		Host:        "127.0.0.1",
		Port:        9000,
		ReadTimeout: 5 * time.Second,
	}

	server := NewServer(config)
	if server == nil {
		t.Fatal("NewServer() should return a non-nil server")
	}

	if server.config.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", server.config.Host)
	}

	if server.config.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", server.config.Port)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", config.Host)
	}

	if config.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Port)
	}

	if config.ReadTimeout != 10*time.Second {
		t.Errorf("Expected default ReadTimeout 10s, got %v", config.ReadTimeout)
	}
}

func TestEngine(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	if engine == nil {
		t.Fatal("Engine() should return a non-nil engine")
	}
}

func TestGET(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "success" {
		t.Errorf("Expected message 'success', got '%v'", response["message"])
	}
}

func TestPOST(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.POST("/test", func(c *gin.Context) {
		var requestBody map[string]interface{}
		c.ShouldBindJSON(&requestBody)

		c.JSON(http.StatusCreated, gin.H{
			"message": "created",
			"data":    requestBody,
		})
	})

	// 创建测试请求
	requestData := map[string]interface{}{
		"name": "test",
	}
	jsonData, _ := json.Marshal(requestData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "created" {
		t.Errorf("Expected message 'created', got '%v'", response["message"])
	}
}

func TestPUT(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "updated" {
		t.Errorf("Expected message 'updated', got '%v'", response["message"])
	}
}

func TestDELETE(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.DELETE("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestGroup(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	api := engine.Group("/api")
	api.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{"user1", "user2"}})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	users, ok := response["users"].([]interface{})
	if !ok {
		t.Fatal("Expected users array in response")
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestStatic(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.Static("/static", "./static")
	// 这里主要测试方法不会panic
}

func TestNoRoute(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] != "not found" {
		t.Errorf("Expected error 'not found', got '%v'", response["error"])
	}
}

func TestUse(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加全局中间件
	engine.Use(func(c *gin.Context) {
		c.Header("X-Test", "test-value")
		c.Next()
	})

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("X-Test") != "test-value" {
		t.Errorf("Expected X-Test header 'test-value', got '%s'", w.Header().Get("X-Test"))
	}
}

func TestShutdown(t *testing.T) {
	server := NewServer(nil)

	// 测试关闭方法
	ctx := context.Background()
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error on shutdown, got %v", err)
	}
}

func TestMiddlewareExecution(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	executionOrder := []string{}

	// 添加多个中间件
	engine.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "middleware1")
		c.Next()
		executionOrder = append(executionOrder, "middleware1-after")
	})

	engine.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "middleware2")
		c.Next()
		executionOrder = append(executionOrder, "middleware2-after")
	})

	engine.GET("/test", func(c *gin.Context) {
		executionOrder = append(executionOrder, "handler")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	expectedOrder := []string{
		"middleware1",
		"middleware2",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d execution steps, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, step := range expectedOrder {
		if i < len(executionOrder) && executionOrder[i] != step {
			t.Errorf("Expected step %d to be '%s', got '%s'", i, step, executionOrder[i])
		}
	}
}

func TestQueryParameters(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		page := c.DefaultQuery("page", "1")

		c.JSON(http.StatusOK, gin.H{
			"query": query,
			"page":  page,
		})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/search?q=test&page=2", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["query"] != "test" {
		t.Errorf("Expected query 'test', got '%v'", response["query"])
	}

	if response["page"] != "2" {
		t.Errorf("Expected page '2', got '%v'", response["page"])
	}
}

func TestPathParameters(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	engine.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/123", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["id"] != "123" {
		t.Errorf("Expected id '123', got '%v'", response["id"])
	}
}

// 新增：测试 TraceID 中间件
func TestTraceIDMiddleware(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加 TraceID 中间件
	engine.Use(TraceIDMiddleware())

	engine.GET("/test", func(c *gin.Context) {
		traceID := GetTraceID(c)
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	// 创建测试请求（不带 Trace ID）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// 检查响应头中是否有 Trace ID
	traceIDHeader := w.Header().Get(constants.TraceIDHeader)
	if traceIDHeader == "" {
		t.Error("Expected X-Trace-ID header to be set")
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["trace_id"] == "" {
		t.Error("Expected trace_id in response")
	}

	// 验证响应头和响应体中的 trace_id 是否一致
	if response["trace_id"] != traceIDHeader {
		t.Errorf("Expected trace_id in response (%s) to match header (%s)", response["trace_id"], traceIDHeader)
	}
}

// 新增：测试自定义 Trace ID
func TestTraceIDMiddlewareWithCustomID(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加 TraceID 中间件
	engine.Use(TraceIDMiddleware())

	engine.GET("/test", func(c *gin.Context) {
		traceID := GetTraceID(c)
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	// 创建测试请求（带自定义 Trace ID）
	customTraceID := "custom-trace-12345"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set(constants.TraceIDHeader, customTraceID)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// 检查响应头中的 Trace ID
	traceIDHeader := w.Header().Get(constants.TraceIDHeader)
	if traceIDHeader != customTraceID {
		t.Errorf("Expected X-Trace-ID header to be '%s', got '%s'", customTraceID, traceIDHeader)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["trace_id"] != customTraceID {
		t.Errorf("Expected trace_id to be '%s', got '%s'", customTraceID, response["trace_id"])
	}
}

// 新增：测试 RequestID 中间件
func TestRequestIDMiddleware(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加 RequestID 中间件
	engine.Use(RequestIDMiddleware())

	engine.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.JSON(http.StatusOK, gin.H{"request_id": requestID})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// 检查响应头中是否有 Request ID
	requestIDHeader := w.Header().Get(constants.RequestIDHeader)
	if requestIDHeader == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["request_id"] == "" {
		t.Error("Expected request_id in response")
	}

	// 验证响应头和响应体中的 request_id 是否一致
	if response["request_id"] != requestIDHeader {
		t.Errorf("Expected request_id in response (%s) to match header (%s)", response["request_id"], requestIDHeader)
	}
}

// 新增：测试 CORS 中间件
func TestCORSMiddleware(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加 CORS 中间件
	engine.Use(CORSMiddleware())

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// 检查 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header to be '*'")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}
}

// 新增：测试 ContextFromGin
func TestContextFromGin(t *testing.T) {
	server := NewServer(nil)
	engine := server.Engine()

	// 添加 TraceID 和 RequestID 中间件
	engine.Use(TraceIDMiddleware())
	engine.Use(RequestIDMiddleware())

	engine.GET("/test", func(c *gin.Context) {
		ctx := ContextFromGin(c)

		// 从 context 中提取 trace_id 和 request_id
		traceID := constants.TraceIDFromContext(ctx)
		requestID := constants.RequestIDFromContext(ctx)

		// 也从 gin context 中获取进行对比
		ginTraceID := GetTraceID(c)
		ginRequestID := GetRequestID(c)

		c.JSON(http.StatusOK, gin.H{
			"ctx_trace_id":   traceID,
			"ctx_request_id": requestID,
			"gin_trace_id":   ginTraceID,
			"gin_request_id": ginRequestID,
		})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// 验证从 context 和 gin 中获取的 ID 是否一致
	if response["ctx_trace_id"] != response["gin_trace_id"] {
		t.Errorf("Context trace_id (%s) should match gin trace_id (%s)",
			response["ctx_trace_id"], response["gin_trace_id"])
	}

	if response["ctx_request_id"] != response["gin_request_id"] {
		t.Errorf("Context request_id (%s) should match gin request_id (%s)",
			response["ctx_request_id"], response["gin_request_id"])
	}
}
