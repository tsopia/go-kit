package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	httpkit "github.com/tsopia/go-kit/pkg/httpclient"
)

// SimpleLogger 实现一个简单的日志记录器
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("[DEBUG] %s\n", msg)
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("[INFO] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	fmt.Printf("[WARN] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("[ERROR] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func main() {
	fmt.Println("=== HTTP Debug功能演示 ===")

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理时间
		time.Sleep(50 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "req-12345")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"status":  "success",
			"message": "请求处理成功",
			"data": map[string]interface{}{
				"user_id": 123,
				"name":    "张三",
				"email":   "zhangsan@example.com",
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建HTTP客户端，启用debug
	client := httpkit.NewClientWithOptions(httpkit.ClientOptions{
		BaseURL: server.URL,
		Logger:  &SimpleLogger{},
		Debug:   httpkit.DefaultDebugConfig(),
		Headers: map[string]string{
			"User-Agent": "Go-Kit-Demo/1.0",
		},
	})

	fmt.Println("\n1. 测试成功的POST请求（带JSON数据）")
	fmt.Println("====================================================")

	requestData := map[string]interface{}{
		"username": "testuser",
		"password": "secret123",
		"email":    "test@example.com",
	}

	resp, err := client.NewRequest("POST", "/api/login").
		Header("Authorization", "Bearer secret-token-12345").
		Header("X-Client-Version", "1.0.0").
		JSON(requestData).
		Do()

	if err != nil {
		log.Printf("请求失败: %v", err)
	} else {
		fmt.Printf("请求成功，状态码: %d\n", resp.StatusCode)
		var result map[string]interface{}
		if err := resp.JSON(&result); err == nil {
			fmt.Printf("响应数据: %+v\n", result)
		}
	}

	fmt.Println("\n2. 测试失败的请求（演示错误情况）")
	fmt.Println("====================================================")

	// 创建一个会失败的客户端
	failClient := httpkit.NewClientWithOptions(httpkit.ClientOptions{
		BaseURL: "http://nonexistent.example.com",
		Logger:  &SimpleLogger{},
		Debug:   httpkit.DefaultDebugConfig(),
	})

	_, err = failClient.NewRequest("GET", "/api/data").
		Header("X-Test-Header", "test-value").
		Do()

	if err != nil {
		fmt.Printf("预期的错误: %v\n", err)
	}

	fmt.Println("\n3. 测试没有logger的情况（直接输出到终端）")
	fmt.Println("====================================================")

	// 创建没有logger的客户端
	noLoggerClient := httpkit.NewClientWithOptions(httpkit.ClientOptions{
		BaseURL: server.URL,
		Logger:  nil, // 没有logger
		Debug:   httpkit.DefaultDebugConfig(),
	})

	resp, err = noLoggerClient.Get("/api/simple")
	if err != nil {
		log.Printf("请求失败: %v", err)
	} else {
		fmt.Printf("请求成功，状态码: %d\n", resp.StatusCode)
	}

	fmt.Println("\n=== 演示完成 ===")
}
