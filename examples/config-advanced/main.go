package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tsopia/go-kit/config"
)

// AppConfig 应用配置结构体
type AppConfig struct {
	App struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Port        int    `mapstructure:"port"`
		Debug       bool   `mapstructure:"debug"`
		Environment string `mapstructure:"environment"`
	} `mapstructure:"app"`

	Database struct {
		Host           string `mapstructure:"host"`
		Port           int    `mapstructure:"port"`
		MaxConnections int    `mapstructure:"max_connections"`
		SSL            bool   `mapstructure:"ssl"`
	} `mapstructure:"database"`

	Redis struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Password string `mapstructure:"password"`
	} `mapstructure:"redis"`

	Features struct {
		AdvancedMode bool     `mapstructure:"advanced_mode"`
		AllowedIPs   []string `mapstructure:"allowed_ips"`
	} `mapstructure:"features"`
}

func main() {
	fmt.Println("🚀 Go-Kit 配置系统优化 - 高级用法演示")
	fmt.Println(strings.Repeat("=", 50))

	// 创建配置文件用于演示
	createDemoConfig()
	defer os.Remove("config.yml")

	// 演示1: 基础配置加载 (推荐用于 80% 的场景)
	fmt.Println("\n📦 1. 基础配置加载 - 类型安全，编译时检查")
	demonstrateBasicConfig()

	// 演示2: 高级动态配置访问
	fmt.Println("\n🔧 2. 高级配置客户端 - 动态访问，完整功能")
	demonstrateAdvancedClient()

	// 演示3: 便利函数 - 快速开发
	fmt.Println("\n⚡ 3. 便利函数 - 默认值和验证")
	demonstrateConvenienceFunctions()

	// 演示4: 错误处理优化
	fmt.Println("\n🛡️ 4. 统一错误处理")
	demonstrateErrorHandling()

	// 演示5: Must函数 - 简化启动阶段
	fmt.Println("\n💪 5. Must函数 - 启动失败即终止")
	demonstrateMustFunctions()

	// 演示6: 配置诊断
	fmt.Println("\n🔍 6. 配置诊断和调试")
	demonstrateConfigDiagnostics()

	// 演示7: 清理资源
	fmt.Println("\n🧹 7. 资源清理")
	demonstrateCleanup()

	fmt.Println("\n✅ 所有演示完成!")
	fmt.Println("\n💡 使用建议:")
	fmt.Println("  - 基础应用: 使用 LoadConfig + 结构体")
	fmt.Println("  - 动态配置: 使用 GetClient")
	fmt.Println("  - 快速开发: 使用便利函数 GetXWithDefault")
	fmt.Println("  - 启动阶段: 使用 Must* 函数")
	fmt.Println("  - 应用退出: 调用 Cleanup()")
}

func demonstrateBasicConfig() {
	var cfg AppConfig
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("❌ 加载配置失败: %v", err)
		return
	}

	fmt.Printf("✅ 应用名称: %s\n", cfg.App.Name)
	fmt.Printf("✅ 运行端口: %d\n", cfg.App.Port)
	fmt.Printf("✅ 调试模式: %v\n", cfg.App.Debug)
	fmt.Printf("✅ 数据库连接数: %d\n", cfg.Database.MaxConnections)
}

func demonstrateAdvancedClient() {
	client, err := config.GetClient()
	if err != nil {
		log.Printf("❌ 获取配置客户端失败: %v", err)
		return
	}

	// 动态配置访问
	env := client.GetString("app.environment")
	fmt.Printf("✅ 运行环境: %s\n", env)

	// 配置存在性检查
	if client.IsSet("redis.password") {
		fmt.Println("✅ Redis 密码已配置")
	} else {
		fmt.Println("⚠️  Redis 密码未配置")
	}

	// 嵌套配置访问
	if client.IsSet("features") {
		fmt.Printf("✅ 功能配置存在，高级模式: %v\n", client.GetBool("features.advanced_mode"))
	}
}

func demonstrateConvenienceFunctions() {
	// 带默认值的配置获取
	logLevel, err := config.GetStringWithDefault("logging.level", "info")
	if err != nil {
		log.Printf("❌ 获取日志级别失败: %v", err)
		return
	}
	fmt.Printf("✅ 日志级别: %s\n", logLevel)

	// 端口验证
	port, valid, err := config.GetIntWithValidation("app.port", 8080, 1, 65535)
	if err != nil {
		log.Printf("❌ 端口验证失败: %v", err)
		return
	}
	if valid {
		fmt.Printf("✅ 端口配置有效: %d\n", port)
	} else {
		fmt.Printf("⚠️  端口配置无效，使用默认值: %d\n", port)
	}

	// 布尔配置
	enableMetrics, err := config.GetBoolWithDefault("metrics.enabled", false)
	if err != nil {
		log.Printf("❌ 获取指标配置失败: %v", err)
		return
	}
	fmt.Printf("✅ 指标收集: %v\n", enableMetrics)

	// 字符串数组配置
	allowedIPs, err := config.GetStringSliceWithDefault("features.allowed_ips", []string{"127.0.0.1"})
	if err != nil {
		log.Printf("❌ 获取IP白名单失败: %v", err)
		return
	}
	fmt.Printf("✅ 允许的IP: %v\n", allowedIPs)
}

func demonstrateErrorHandling() {
	// 演示统一的错误处理
	_, err := config.GetStringWithDefault("invalid.nested.key", "default")
	if err != nil {
		fmt.Printf("❌ 预期错误（演示用）: %v\n", err)
	} else {
		fmt.Println("✅ 错误处理正常")
	}

	// 演示配置检查
	exists, err := config.IsSet("nonexistent.key")
	if err != nil {
		fmt.Printf("❌ 检查配置存在性失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 不存在的配置键检查结果: %v\n", exists)
}

func demonstrateMustFunctions() {
	// Must函数用于启动阶段，失败即终止
	fmt.Println("✅ 使用Must函数获取关键配置:")

	// 这些函数在失败时会panic，适合应用启动阶段
	appName := config.MustGetStringWithDefault("app.name", "MyApp")
	fmt.Printf("  - 应用名称: %s\n", appName)

	dbPort := config.MustGetIntWithDefault("database.port", 5432)
	fmt.Printf("  - 数据库端口: %d\n", dbPort)

	debugMode := config.MustGetBoolWithDefault("app.debug", false)
	fmt.Printf("  - 调试模式: %v\n", debugMode)

	fmt.Println("💡 Must函数适用于应用启动阶段，配置缺失时立即终止程序")
}

func demonstrateConfigDiagnostics() {
	// 获取所有配置键
	keys, err := config.AllKeys()
	if err != nil {
		log.Printf("❌ 获取配置键失败: %v", err)
		return
	}

	fmt.Printf("✅ 当前配置项数量: %d\n", len(keys))
	fmt.Println("📋 配置键列表:")
	for i, key := range keys {
		if i < 5 { // 只显示前5个，避免输出太长
			exists, _ := config.IsSet(key)
			fmt.Printf("  - %s (存在: %v)\n", key, exists)
		}
	}
	if len(keys) > 5 {
		fmt.Printf("  ... 以及其他 %d 个配置项\n", len(keys)-5)
	}
}

func demonstrateCleanup() {
	fmt.Println("🧹 清理配置资源...")
	config.Cleanup()
	fmt.Println("✅ 配置资源已清理")

	// 验证清理后状态
	_, err := config.GetClient()
	if err != nil {
		fmt.Printf("✅ 清理验证成功 - 需要重新初始化: %v\n", err)
	}
}

func createDemoConfig() {
	configContent := `
app:
  name: "Go-Kit Advanced Demo"
  version: "2.0.0"
  port: 8080
  debug: true
  environment: "development"

database:
  host: "localhost"
  port: 5432
  max_connections: 20
  ssl: false

redis:
  host: "localhost"
  port: 6379
  password: ""

features:
  advanced_mode: true
  allowed_ips:
    - "127.0.0.1"
    - "192.168.1.0/24"

metrics:
  enabled: true
  port: 9090

logging:
  level: "debug"
  format: "json"
`

	err := os.WriteFile("config.yml", []byte(configContent), 0644)
	if err != nil {
		log.Fatalf("创建演示配置文件失败: %v", err)
	}
}
