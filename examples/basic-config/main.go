package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tsopia/go-kit/pkg/config"
)

// Config 应用配置结构（使用 mapstructure 标签）
type Config struct {
	Server struct {
		Host string `yaml:"host" mapstructure:"host"`
		Port int    `yaml:"port" mapstructure:"port"`
	} `yaml:"server" mapstructure:"server"`
	Database struct {
		Host     string `yaml:"host" mapstructure:"host"`
		Port     int    `yaml:"port" mapstructure:"port"`
		Username string `yaml:"username" mapstructure:"username"`
		Password string `yaml:"password" mapstructure:"password"`
		Name     string `yaml:"name" mapstructure:"name"`
	} `yaml:"database" mapstructure:"database"`
	Debug bool `yaml:"debug" mapstructure:"debug"`
}

func main() {
	fmt.Println("=== Go-Kit 配置系统演示 ===\n")
	fmt.Println("🚀 基于 Viper 的现代配置加载方案\n")

	// 示例1: 使用默认配置文件路径
	fmt.Println("📋 示例1: 使用默认配置文件路径")
	demonstrateDefaultConfig()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// 示例2: 环境变量覆盖演示（无前缀模式）
	fmt.Println("📋 示例2: 环境变量覆盖演示（无前缀）")
	demonstrateEnvOverride()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// 示例3: 使用APP_NAME自动前缀
	fmt.Println("📋 示例3: 使用 APP_NAME 自动前缀")
	demonstrateAutoPrefix()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// 示例4: 自定义配置文件路径
	fmt.Println("📋 示例4: 自定义配置文件路径")
	demonstrateCustomPath()

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// 示例5: 使用建议和最佳实践
	fmt.Println("📋 示例5: 使用建议和最佳实践")
	demonstrateBestPractices()
}

func demonstrateDefaultConfig() {
	var cfg Config
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("  ❌ 加载默认配置失败: %v", err)
		return
	}

	fmt.Printf("  ✅ 服务器配置: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  ✅ 数据库配置: %s:%d (%s)\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	fmt.Printf("  ✅ 数据库用户: %s\n", cfg.Database.Username)
	fmt.Printf("  ✅ 调试模式: %v\n", cfg.Debug)
}

func demonstrateEnvOverride() {
	// 确保没有设置APP_NAME，使用无前缀模式
	os.Unsetenv("APP_NAME")

	// 设置一些环境变量来覆盖配置文件中的值
	fmt.Println("  💡 设置环境变量来覆盖配置文件中的值...")

	os.Setenv("SERVER_HOST", "env-override-host")
	os.Setenv("SERVER_PORT", "9999")
	os.Setenv("DEBUG", "true")
	os.Setenv("DATABASE_NAME", "env_override_db")

	defer func() {
		// 清理环境变量
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("DEBUG")
		os.Unsetenv("DATABASE_NAME")
	}()

	var cfg Config
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("  ❌ 加载配置失败: %v", err)
		return
	}

	fmt.Printf("  ✅ 环境变量覆盖后 - 服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  ✅ 环境变量覆盖后 - 数据库名: %s\n", cfg.Database.Name)
	fmt.Printf("  ✅ 环境变量覆盖后 - 调试模式: %v\n", cfg.Debug)

	fmt.Println("  📝 使用的环境变量:")
	fmt.Println("     SERVER_HOST=env-override-host")
	fmt.Println("     SERVER_PORT=9999")
	fmt.Println("     DEBUG=true")
	fmt.Println("     DATABASE_NAME=env_override_db")
}

func demonstrateAutoPrefix() {
	// 设置APP_NAME环境变量启用自动前缀模式
	fmt.Println("  💡 使用 APP_NAME 环境变量启用自动前缀...")

	os.Setenv("APP_NAME", "myapp")
	os.Setenv("MYAPP_SERVER_HOST", "prefix-host")
	os.Setenv("MYAPP_SERVER_PORT", "7777")
	os.Setenv("MYAPP_DATABASE_HOST", "prefix-db-host")
	os.Setenv("MYAPP_DEBUG", "true")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_NAME")
		os.Unsetenv("MYAPP_SERVER_HOST")
		os.Unsetenv("MYAPP_SERVER_PORT")
		os.Unsetenv("MYAPP_DATABASE_HOST")
		os.Unsetenv("MYAPP_DEBUG")
	}()

	var cfg Config
	err := config.LoadConfig(&cfg)
	if err != nil {
		log.Printf("  ❌ 加载配置失败: %v", err)
		return
	}

	fmt.Printf("  ✅ 自动前缀模式 - 服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  ✅ 自动前缀模式 - 数据库主机: %s\n", cfg.Database.Host)
	fmt.Printf("  ✅ 自动前缀模式 - 调试模式: %v\n", cfg.Debug)

	fmt.Println("  📝 使用的环境变量:")
	fmt.Println("     APP_NAME=myapp (启用MYAPP_前缀)")
	fmt.Println("     MYAPP_SERVER_HOST=prefix-host")
	fmt.Println("     MYAPP_SERVER_PORT=7777")
	fmt.Println("     MYAPP_DATABASE_HOST=prefix-db-host")
	fmt.Println("     MYAPP_DEBUG=true")
}

func demonstrateCustomPath() {
	// 尝试使用自定义配置文件路径
	customPath := "examples/basic-config/config.yaml"

	var cfg Config
	err := config.LoadConfig(&cfg, customPath)
	if err != nil {
		log.Printf("  ❌ 加载自定义路径配置失败: %v", err)
		fmt.Println("  💡 提示: 确保配置文件存在于指定路径")
		return
	}

	fmt.Printf("  ✅ 自定义路径配置 - 服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  ✅ 自定义路径配置 - 数据库: %s\n", cfg.Database.Name)
	fmt.Printf("  ✅ 使用的配置文件: %s\n", customPath)
}

func demonstrateBestPractices() {
	fmt.Println("  💡 使用建议和最佳实践:")
	fmt.Println()

	fmt.Println("  1️⃣ 配置结构体设计:")
	fmt.Println("     • 使用 `mapstructure` 标签进行字段映射")
	fmt.Println("     • 嵌套结构体组织相关配置")
	fmt.Println("     • 使用有意义的字段名")
	fmt.Println()

	fmt.Println("  2️⃣ 环境变量命名:")
	fmt.Println("     • 无前缀模式: server.host → SERVER_HOST")
	fmt.Println("     • 前缀模式: APP_NAME=myapp → server.host → MYAPP_SERVER_HOST")
	fmt.Println("     • 点号(.)自动转换为下划线(_)并转大写")
	fmt.Println()

	fmt.Println("  3️⃣ APP_NAME 环境变量的特殊作用:")
	fmt.Println("     • 设置 APP_NAME 自动启用环境变量前缀模式")
	fmt.Println("     • APP_NAME=myapp → 前缀变为 MYAPP_")
	fmt.Println("     • 环境变量优先级: MYAPP_APP_NAME > 配置文件中的 app.name")
	fmt.Println()

	fmt.Println("  4️⃣ 配置文件格式:")
	fmt.Println("     • 支持 YAML (.yml, .yaml)")
	fmt.Println("     • 支持 JSON (.json)")
	fmt.Println("     • 支持 TOML (.toml)")
	fmt.Println("     • 支持 HCL (.hcl)")
	fmt.Println()

	fmt.Println("  5️⃣ 错误处理:")
	fmt.Println("     • 总是检查 LoadConfig 的返回错误")
	fmt.Println("     • 提供有意义的错误信息")
	fmt.Println("     • 考虑配置验证逻辑")
	fmt.Println()

	fmt.Println("  6️⃣ 生产环境建议:")
	fmt.Println("     • 使用环境变量覆盖敏感信息（密码、密钥等）")
	fmt.Println("     • 设置合理的默认值")
	fmt.Println("     • 使用配置验证确保必需字段存在")
	fmt.Println("     • 考虑使用 APP_NAME 前缀避免环境变量冲突")

	// 展示一个完整的生产级示例
	fmt.Println("\n  🏆 生产级配置加载示例:")
	var cfg Config
	err := config.LoadConfig(&cfg)
	if err != nil {
		fmt.Printf("     ❌ 配置加载失败: %v\n", err)
		return
	}

	// 简单的配置验证
	if cfg.Server.Host == "" {
		fmt.Println("     ⚠️  警告: 服务器主机未设置")
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		fmt.Println("     ⚠️  警告: 服务器端口设置无效")
	}
	if cfg.Database.Name == "" {
		fmt.Println("     ⚠️  警告: 数据库名称未设置")
	}

	fmt.Printf("     ✅ 配置验证通过 - 服务器将在 %s:%d 启动\n", cfg.Server.Host, cfg.Server.Port)
}
