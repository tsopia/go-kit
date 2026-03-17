package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tsopia/go-kit/cfg"
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
	defer func() {
		if err := os.Remove("config.yml"); err != nil && !os.IsNotExist(err) {
			log.Printf("删除演示配置文件失败: %v", err)
		}
	}()

	// 演示1: 基础配置加载 (推荐用于 80% 的场景)
	fmt.Println("\n📦 1. 基础配置加载 - 类型安全，编译时检查")
	demonstrateBasicConfig()

	// 演示2: 动态配置访问
	fmt.Println("\n🔧 2. 动态配置访问 - 使用 Get 方法")
	demonstrateDynamicAccess()

	// 演示3: 便利函数 - 快速开发
	fmt.Println("\n⚡ 3. 便利函数 - 默认值支持")
	demonstrateConvenienceFunctions()

	// 演示4: 错误处理优化
	fmt.Println("\n🛡️  4. 统一错误处理")
	demonstrateErrorHandling()

	fmt.Println("\n✅ 所有演示完成!")
	fmt.Println("\n💡 使用建议:")
	fmt.Println("  - 基础应用: 使用 cfg.New + 结构体")
	fmt.Println("  - 动态配置: 使用 GetString/GetInt 等方法")
	fmt.Println("  - 可选配置: 使用变长参数设置默认值")
}

func demonstrateBasicConfig() {
	provider, err := cfg.New()
	if err != nil {
		log.Printf("❌ 创建 Provider 失败: %v", err)
		return
	}

	var c AppConfig
	if err := provider.Unmarshal(&c); err != nil {
		log.Printf("❌ 解析配置失败: %v", err)
		return
	}

	fmt.Printf("✅ 应用名称: %s\n", c.App.Name)
	fmt.Printf("✅ 运行端口: %d\n", c.App.Port)
	fmt.Printf("✅ 调试模式: %v\n", c.App.Debug)
	fmt.Printf("✅ 数据库连接数: %d\n", c.Database.MaxConnections)
}

func demonstrateDynamicAccess() {
	provider, err := cfg.New()
	if err != nil {
		log.Printf("❌ 创建 Provider 失败: %v", err)
		return
	}

	// 动态配置访问
	env, _ := provider.GetString("app.environment")
	fmt.Printf("✅ 运行环境: %s\n", env)

	// 检查配置是否存在
	exists := provider.Exists("redis.password")
	if exists {
		fmt.Println("✅ Redis 密码已配置")
	} else {
		fmt.Println("⚠️  Redis 密码未配置")
	}

	// 嵌套配置访问
	if provider.Exists("features") {
		advancedMode, _ := provider.GetBool("features.advanced_mode")
		fmt.Printf("✅ 功能配置存在，高级模式: %v\n", advancedMode)
	}
}

func demonstrateConvenienceFunctions() {
	provider, err := cfg.New()
	if err != nil {
		log.Printf("❌ 创建 Provider 失败: %v", err)
		return
	}

	// 带默认值的配置获取
	logLevel, _ := provider.GetString("logging.level", "info")
	fmt.Printf("✅ 日志级别: %s\n", logLevel)

	// 整数配置（带默认值）
	port, _ := provider.GetInt("app.port", 8080)
	fmt.Printf("✅ 应用端口: %d\n", port)

	// 布尔配置（带默认值）
	enableMetrics, _ := provider.GetBool("metrics.enabled", false)
	fmt.Printf("✅ 指标收集: %v\n", enableMetrics)
}

func demonstrateErrorHandling() {
	provider, err := cfg.New()
	if err != nil {
		log.Printf("❌ 创建 Provider 失败: %v", err)
		return
	}

	// 获取不存在的配置键（带默认值）
	value, _ := provider.GetString("invalid.nested.key", "default")
	fmt.Printf("✅ 使用默认值: %s\n", value)

	// 检查配置是否存在
	exists := provider.Exists("nonexistent.key")
	fmt.Printf("✅ 不存在的配置键检查结果: %v\n", exists)
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
