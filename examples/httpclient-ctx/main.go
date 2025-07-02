package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-kit/pkg/httpclient"
)

func main() {
	// 创建HTTP客户端
	client := httpclient.NewClient()

	// 演示1: 使用WithCtx方法设置带超时的context
	fmt.Println("=== 演示1: 使用WithCtx设置超时context ===")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.NewRequest("GET", "https://httpbin.org/delay/1").
		WithCtx(ctx).
		Do()

	if err != nil {
		log.Printf("请求失败: %v", err)
	} else {
		fmt.Printf("请求成功，状态码: %d, 耗时: %v\n", resp.StatusCode, resp.Duration)
	}

	// 演示2: 使用WithCtx方法设置带取消的context
	fmt.Println("\n=== 演示2: 使用WithCtx设置可取消的context ===")
	ctx2, cancel2 := context.WithCancel(context.Background())

	// 2秒后取消请求
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("取消请求...")
		cancel2()
	}()

	resp2, err2 := client.NewRequest("GET", "https://httpbin.org/delay/5").
		WithCtx(ctx2).
		Do()

	if err2 != nil {
		fmt.Printf("请求被取消: %v\n", err2)
	} else {
		fmt.Printf("请求成功，状态码: %d\n", resp2.StatusCode)
	}

	// 演示3: 使用WithCtx传递trace信息
	fmt.Println("\n=== 演示3: 使用WithCtx传递trace信息 ===")
	ctx3 := context.WithValue(context.Background(), "trace_id", "abc123")
	ctx3 = context.WithValue(ctx3, "user_id", "user456")

	resp3, err3 := client.NewRequest("GET", "https://httpbin.org/get").
		WithCtx(ctx3).
		Header("X-Trace-ID", "abc123").
		Header("X-User-ID", "user456").
		Do()

	if err3 != nil {
		log.Printf("请求失败: %v", err3)
	} else {
		fmt.Printf("请求成功，状态码: %d\n", resp3.StatusCode)
		// 可以通过context获取trace信息
		if traceID := ctx3.Value("trace_id"); traceID != nil {
			fmt.Printf("Trace ID: %v\n", traceID)
		}
		if userID := ctx3.Value("user_id"); userID != nil {
			fmt.Printf("User ID: %v\n", userID)
		}
	}

	// 演示4: 对比Context方法和WithCtx方法
	fmt.Println("\n=== 演示4: Context方法 vs WithCtx方法 ===")
	ctx4, cancel4 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel4()

	// 使用Context方法
	req1 := client.NewRequest("GET", "https://httpbin.org/get").Context(ctx4)
	fmt.Println("使用Context方法创建请求")

	// 使用WithCtx方法（更简洁）
	req2 := client.NewRequest("GET", "https://httpbin.org/get").WithCtx(ctx4)
	fmt.Println("使用WithCtx方法创建请求")

	// 两种方法功能完全一样
	resp4a, _ := req1.Do()
	resp4b, _ := req2.Do()

	if resp4a != nil && resp4b != nil {
		fmt.Printf("两种方法都成功，状态码: %d, %d\n", resp4a.StatusCode, resp4b.StatusCode)
	}

	fmt.Println("\n✅ WithCtx 方法演示完成！")
}
