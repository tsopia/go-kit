package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tsopia/go-kit/config"
)

// AppConfig 定义应用程序配置结构
type AppConfig struct {
	App struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Port        int    `mapstructure:"port"`
		Environment string `mapstructure:"environment"`
		Debug       bool   `mapstructure:"debug"`
	} `mapstructure:"app"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"database"`

	Redis struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
		Output string `mapstructure:"output"`
	} `mapstructure:"logging"`
}

func main() {
	fmt.Println("=== Go-Kit 配置加载演示 ===")
	fmt.Println()

	// 示例1：使用默认配置文件路径 (config.yml)
	fmt.Println("示例 1: 使用默认配置文件")
	demonstrateDefaultConfig()

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// 示例2：使用自定义配置文件路径
	fmt.Println("示例 2: 使用自定义配置文件路径")
	demonstrateCustomConfigPath()

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// 示例3：演示环境变量覆盖（无前缀模式）
	fmt.Println("示例 3: 环境变量覆盖配置文件（无前缀）")
	demonstrateEnvOverride()

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// 示例4：使用APP_NAME自动前缀
	fmt.Println("示例 4: 使用 APP_NAME 自动前缀")
	demonstrateAutoPrefix()

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// 示例5：APP_NAME 优先级演示
	fmt.Println("示例 5: APP_NAME 优先级演示")
	demonstrateAppNamePriority()
}

func demonstrateDefaultConfig() {
	var cfg AppConfig

	// 使用默认配置文件路径
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("加载默认配置失败: %v", err)
		return
	}

	printConfig("默认配置", &cfg)
}

func demonstrateCustomConfigPath() {
	var cfg AppConfig

	// 使用自定义配置文件路径
	err := config.LoadConfig(&cfg, "examples/config-demo/custom-config.yml")
	if err != nil {
		log.Printf("加载自定义配置失败: %v", err)
		return
	}

	printConfig("自定义路径配置", &cfg)
}

func demonstrateEnvOverride() {
	// 确保没有设置 APP_NAME，使用无前缀模式
	os.Unsetenv("APP_NAME")

	// 设置环境变量来覆盖配置文件中的值
	os.Setenv("APP_PORT", "9090")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("DATABASE_HOST", "env-db-host")
	os.Setenv("LOGGING_LEVEL", "debug")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_PORT")
		os.Unsetenv("APP_DEBUG")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("LOGGING_LEVEL")
	}()

	var cfg AppConfig
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return
	}

	fmt.Println("  💡 使用的环境变量（无前缀模式）:")
	fmt.Println("     APP_PORT=9090")
	fmt.Println("     APP_DEBUG=true")
	fmt.Println("     DATABASE_HOST=env-db-host")
	fmt.Println("     LOGGING_LEVEL=debug")
	fmt.Println()

	printConfig("环境变量覆盖后的配置", &cfg)
}

func demonstrateAutoPrefix() {
	// 设置 APP_NAME 环境变量启用自动前缀
	os.Setenv("APP_NAME", "myapp")

	// 设置带前缀的环境变量
	os.Setenv("MYAPP_APP_NAME", "带前缀的应用名称")
	os.Setenv("MYAPP_APP_PORT", "7777")
	os.Setenv("MYAPP_DATABASE_HOST", "prefix-db-host")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_NAME")
		os.Unsetenv("MYAPP_APP_NAME")
		os.Unsetenv("MYAPP_APP_PORT")
		os.Unsetenv("MYAPP_DATABASE_HOST")
	}()

	var cfg AppConfig
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return
	}

	fmt.Println("  💡 APP_NAME 自动前缀功能:")
	fmt.Println("     APP_NAME=myapp (设置前缀为 MYAPP_)")
	fmt.Println("     MYAPP_APP_NAME=带前缀的应用名称")
	fmt.Println("     MYAPP_APP_PORT=7777")
	fmt.Println("     MYAPP_DATABASE_HOST=prefix-db-host")
	fmt.Println()

	printConfig("自动前缀配置", &cfg)
}

func demonstrateAppNamePriority() {
	// 演示当配置文件和环境变量都有 app_name 时，环境变量优先级最高

	// 设置 APP_NAME 启用前缀
	os.Setenv("APP_NAME", "priority")

	// 设置环境变量覆盖 app.name
	os.Setenv("PRIORITY_APP_NAME", "环境变量优先的应用名")
	os.Setenv("PRIORITY_APP_PORT", "8888")

	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("PRIORITY_APP_NAME")
		os.Unsetenv("PRIORITY_APP_PORT")
	}()

	var cfg AppConfig
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return
	}

	fmt.Println("  💡 优先级演示:")
	fmt.Println("     配置文件中: app.name = 'Go-Kit 示例应用'")
	fmt.Println("     环境变量中: PRIORITY_APP_NAME = '环境变量优先的应用名'")
	fmt.Println("     结果: 环境变量优先级更高")
	fmt.Println()

	printConfig("优先级演示配置", &cfg)
}

func printConfig(title string, cfg *AppConfig) {
	fmt.Printf("📋 %s:\n", title)
	fmt.Printf("  应用信息:\n")
	fmt.Printf("    名称: %s\n", cfg.App.Name)
	fmt.Printf("    版本: %s\n", cfg.App.Version)
	fmt.Printf("    端口: %d\n", cfg.App.Port)
	fmt.Printf("    环境: %s\n", cfg.App.Environment)
	fmt.Printf("    调试模式: %t\n", cfg.App.Debug)

	fmt.Printf("  数据库配置:\n")
	fmt.Printf("    主机: %s\n", cfg.Database.Host)
	fmt.Printf("    端口: %d\n", cfg.Database.Port)
	fmt.Printf("    用户名: %s\n", cfg.Database.Username)
	fmt.Printf("    数据库名: %s\n", cfg.Database.DBName)

	fmt.Printf("  Redis配置:\n")
	fmt.Printf("    主机: %s\n", cfg.Redis.Host)
	fmt.Printf("    端口: %d\n", cfg.Redis.Port)
	fmt.Printf("    数据库: %d\n", cfg.Redis.DB)

	fmt.Printf("  日志配置:\n")
	fmt.Printf("    级别: %s\n", cfg.Logging.Level)
	fmt.Printf("    格式: %s\n", cfg.Logging.Format)
	fmt.Printf("    输出: %s\n", cfg.Logging.Output)
}
