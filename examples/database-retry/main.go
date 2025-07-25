package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tsopia/go-kit/database"
)

func main() {
	fmt.Println("=== Database 连接重试机制演示 ===")

	// 1. 默认重试配置
	fmt.Println("\n1. 使用默认重试配置")
	config1 := &database.Config{
		Driver:   "sqlite",
		Database: ":memory:",
	}
	config1.SetDefaults()

	fmt.Printf("重试配置: 最大次数=%d, 初始延迟=%v, 最大延迟=%v, 退避因子=%.1f, 抖动=%v\n",
		config1.RetryMaxAttempts,
		config1.RetryInitialDelay,
		config1.RetryMaxDelay,
		config1.RetryBackoffFactor,
		config1.RetryJitterEnabled)

	db1, err := database.New(config1)
	if err != nil {
		log.Fatalf("创建数据库连接失败: %v", err)
	}
	defer db1.Close()

	fmt.Println("✅ 数据库连接成功")

	// 2. 自定义重试配置
	fmt.Println("\n2. 使用自定义重试配置")
	config2 := &database.Config{
		Driver:   "sqlite",
		Database: ":memory:",

		// 自定义重试策略
		RetryEnabled:       true,
		RetryMaxAttempts:   5,
		RetryInitialDelay:  500 * time.Millisecond,
		RetryMaxDelay:      10 * time.Second,
		RetryBackoffFactor: 1.5,
		RetryJitterEnabled: true,
	}

	fmt.Printf("重试配置: 最大次数=%d, 初始延迟=%v, 最大延迟=%v, 退避因子=%.1f, 抖动=%v\n",
		config2.RetryMaxAttempts,
		config2.RetryInitialDelay,
		config2.RetryMaxDelay,
		config2.RetryBackoffFactor,
		config2.RetryJitterEnabled)

	db2, err := database.New(config2)
	if err != nil {
		log.Fatalf("创建数据库连接失败: %v", err)
	}
	defer db2.Close()

	fmt.Println("✅ 数据库连接成功")

	// 3. 禁用重试
	fmt.Println("\n3. 禁用重试机制")
	config3 := &database.Config{
		Driver:   "sqlite",
		Database: ":memory:",

		// 禁用重试
		RetryEnabled:     false,
		RetryMaxAttempts: 1,
	}

	fmt.Printf("重试配置: 启用=%v, 最大次数=%d\n",
		config3.RetryEnabled,
		config3.RetryMaxAttempts)

	db3, err := database.New(config3)
	if err != nil {
		log.Fatalf("创建数据库连接失败: %v", err)
	}
	defer db3.Close()

	fmt.Println("✅ 数据库连接成功")

	// 4. 测试重试延迟计算
	fmt.Println("\n4. 重试延迟计算演示")
	testConfig := &database.Config{
		RetryInitialDelay:  time.Second,
		RetryMaxDelay:      30 * time.Second,
		RetryBackoffFactor: 2.0,
		RetryJitterEnabled: false, // 禁用抖动以显示精确值
	}

	fmt.Println("重试次数 -> 延迟时间:")
	for i := 0; i < 6; i++ {
		delay := calculateRetryDelay(testConfig, i)
		fmt.Printf("  第%d次重试: %v\n", i+1, delay)
	}

	fmt.Println("\n=== 演示完成 ===")
}

// 导出计算重试延迟的函数用于演示
func calculateRetryDelay(config *database.Config, attempt int) time.Duration {
	// 这里使用与database包中相同的逻辑
	delay := float64(config.RetryInitialDelay) * pow(config.RetryBackoffFactor, float64(attempt))

	// 限制最大延迟
	if config.RetryMaxDelay > 0 && time.Duration(delay) > config.RetryMaxDelay {
		delay = float64(config.RetryMaxDelay)
	}

	return time.Duration(delay)
}

// 简单的幂运算函数
func pow(base float64, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
