package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConfig 测试用的配置结构体
type TestConfig struct {
	App struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
		Port    int    `mapstructure:"port"`
		Debug   bool   `mapstructure:"debug"`
	} `mapstructure:"app"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`
}

func TestLoadConfig_DefaultPath(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  version: "1.0.0"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "testuser"
  password: "testpass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if cfg.App.Name != "Test App" {
		t.Errorf("期望 App.Name = 'Test App', 实际 = '%s'", cfg.App.Name)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("期望 App.Port = 8080, 实际 = %d", cfg.App.Port)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("期望 Database.Host = 'localhost', 实际 = '%s'", cfg.Database.Host)
	}
}

func TestLoadConfig_CustomPath(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "custom.yml")

	configContent := `
app:
  name: "Custom App"
  version: "2.0.0"
  port: 3000
  debug: true

database:
  host: "custom-host"
  port: 3306
  username: "custom-user"
  password: "custom-pass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	var cfg TestConfig
	err = LoadConfig(&cfg, configFile)
	if err != nil {
		t.Fatalf("加载自定义配置失败: %v", err)
	}

	// 验证配置值
	if cfg.App.Name != "Custom App" {
		t.Errorf("期望 App.Name = 'Custom App', 实际 = '%s'", cfg.App.Name)
	}
	if cfg.App.Port != 3000 {
		t.Errorf("期望 App.Port = 3000, 实际 = %d", cfg.App.Port)
	}
	if cfg.App.Debug != true {
		t.Errorf("期望 App.Debug = true, 实际 = %t", cfg.App.Debug)
	}
}

func TestLoadConfig_EnvironmentOverride(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Original App"
  version: "1.0.0"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "originaluser"
  password: "originalpass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 确保 APP_NAME 未设置，测试无前缀模式的环境变量覆盖
	os.Unsetenv("APP_NAME")

	// 设置环境变量（无前缀模式，不包括APP_NAME以避免前缀冲突）
	os.Setenv("APP_PORT", "9999")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("DATABASE_HOST", "env-host")
	os.Setenv("DATABASE_PORT", "3307")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_PORT")
		os.Unsetenv("APP_DEBUG")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
	}()

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量覆盖了配置文件中的值（除了app.name保持原值）
	if cfg.App.Name != "Original App" {
		t.Errorf("期望 App.Name = 'Original App' (未设置APP_NAME环境变量), 实际 = '%s'", cfg.App.Name)
	}
	if cfg.App.Port != 9999 {
		t.Errorf("期望 App.Port = 9999, 实际 = %d", cfg.App.Port)
	}
	if cfg.App.Debug != true {
		t.Errorf("期望 App.Debug = true, 实际 = %t", cfg.App.Debug)
	}
	if cfg.Database.Host != "env-host" {
		t.Errorf("期望 Database.Host = 'env-host', 实际 = '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("期望 Database.Port = 3307, 实际 = %d", cfg.Database.Port)
	}
}

func TestLoadConfig_AutoEnvPrefix(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Original App"
  version: "1.0.0"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "originaluser"
  password: "originalpass"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 设置 APP_NAME 环境变量来启用前缀模式
	os.Setenv("APP_NAME", "myapp")

	// 设置带前缀的环境变量
	os.Setenv("MYAPP_APP_NAME", "Prefix Override App")
	os.Setenv("MYAPP_APP_PORT", "7777")
	os.Setenv("MYAPP_DATABASE_HOST", "prefix-host")

	defer func() {
		// 清理环境变量
		os.Unsetenv("APP_NAME")
		os.Unsetenv("MYAPP_APP_NAME")
		os.Unsetenv("MYAPP_APP_PORT")
		os.Unsetenv("MYAPP_DATABASE_HOST")
	}()

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证带前缀的环境变量覆盖了配置文件中的值
	if cfg.App.Name != "Prefix Override App" {
		t.Errorf("期望 App.Name = 'Prefix Override App', 实际 = '%s'", cfg.App.Name)
	}
	if cfg.App.Port != 7777 {
		t.Errorf("期望 App.Port = 7777, 实际 = %d", cfg.App.Port)
	}
	if cfg.Database.Host != "prefix-host" {
		t.Errorf("期望 Database.Host = 'prefix-host', 实际 = '%s'", cfg.Database.Host)
	}
}

func TestLoadConfig_AppNamePriority(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 测试当配置文件和环境变量都有 app_name 时，环境变量优先级更高
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Config File App"
  port: 8080
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 设置 APP_NAME 环境变量
	os.Setenv("APP_NAME", "testapp")

	// 设置环境变量覆盖 app.name
	os.Setenv("TESTAPP_APP_NAME", "Env Priority App")

	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("TESTAPP_APP_NAME")
	}()

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量优先级更高
	if cfg.App.Name != "Env Priority App" {
		t.Errorf("期望 App.Name = 'Env Priority App' (环境变量优先), 实际 = '%s'", cfg.App.Name)
	}
}

func TestGetClient(t *testing.T) {
	// 重置全局状态
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  port: 8080
  debug: true

database:
  host: "localhost"
  port: 5432
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 获取客户端
	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient 失败: %v", err)
	}
	if client == nil {
		t.Fatal("GetClient 返回 nil")
	}

	// 测试基本配置获取
	appName := client.GetString("app.name")
	if appName != "Test App" {
		t.Errorf("期望 app.name = 'Test App', 实际 = '%s'", appName)
	}

	appPort := client.GetInt("app.port")
	if appPort != 8080 {
		t.Errorf("期望 app.port = 8080, 实际 = %d", appPort)
	}

	appDebug := client.GetBool("app.debug")
	if appDebug != true {
		t.Errorf("期望 app.debug = true, 实际 = %t", appDebug)
	}
}

func TestGetStringWithDefault(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	appName, err := GetStringWithDefault("app.name", "默认应用名")
	if err != nil {
		t.Fatalf("GetStringWithDefault 失败: %v", err)
	}
	if appName != "Test App" {
		t.Errorf("期望 app.name = 'Test App', 实际 = '%s'", appName)
	}

	// 测试不存在的配置项，应该返回默认值
	logLevel, err := GetStringWithDefault("logging.level", "info")
	if err != nil {
		t.Fatalf("GetStringWithDefault 失败: %v", err)
	}
	if logLevel != "info" {
		t.Errorf("期望 logging.level = 'info' (默认值), 实际 = '%s'", logLevel)
	}
}

func TestGetIntWithDefault(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  port: 8080
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	appPort, err := GetIntWithDefault("app.port", 3000)
	if err != nil {
		t.Fatalf("GetIntWithDefault 失败: %v", err)
	}
	if appPort != 8080 {
		t.Errorf("期望 app.port = 8080, 实际 = %d", appPort)
	}

	// 测试不存在的配置项，应该返回默认值
	dbPort, err := GetIntWithDefault("database.port", 5432)
	if err != nil {
		t.Fatalf("GetIntWithDefault 失败: %v", err)
	}
	if dbPort != 5432 {
		t.Errorf("期望 database.port = 5432 (默认值), 实际 = %d", dbPort)
	}
}

func TestGetBoolWithDefault(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  debug: true
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	appDebug, err := GetBoolWithDefault("app.debug", false)
	if err != nil {
		t.Fatalf("GetBoolWithDefault 失败: %v", err)
	}
	if appDebug != true {
		t.Errorf("期望 app.debug = true, 实际 = %t", appDebug)
	}

	// 测试不存在的配置项，应该返回默认值
	enableCache, err := GetBoolWithDefault("cache.enabled", false)
	if err != nil {
		t.Fatalf("GetBoolWithDefault 失败: %v", err)
	}
	if enableCache != false {
		t.Errorf("期望 cache.enabled = false (默认值), 实际 = %t", enableCache)
	}
}

func TestGetIntWithValidation(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  port: 8080
server:
  port: 99999  # 超出有效范围
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试有效范围内的配置
	port, valid, err := GetIntWithValidation("app.port", 3000, 1, 65535)
	if err != nil {
		t.Fatalf("GetIntWithValidation 失败: %v", err)
	}
	if !valid {
		t.Error("期望 app.port 在有效范围内")
	}
	if port != 8080 {
		t.Errorf("期望 app.port = 8080, 实际 = %d", port)
	}

	// 测试超出有效范围的配置
	serverPort, valid, err := GetIntWithValidation("server.port", 3000, 1, 65535)
	if err != nil {
		t.Fatalf("GetIntWithValidation 失败: %v", err)
	}
	if valid {
		t.Error("期望 server.port 超出有效范围")
	}
	if serverPort != 3000 {
		t.Errorf("期望 server.port = 3000 (默认值), 实际 = %d", serverPort)
	}

	// 测试不存在的配置项
	dbPort, valid, err := GetIntWithValidation("database.port", 5432, 1, 65535)
	if err != nil {
		t.Fatalf("GetIntWithValidation 失败: %v", err)
	}
	if !valid {
		t.Error("期望默认值在有效范围内")
	}
	if dbPort != 5432 {
		t.Errorf("期望 database.port = 5432 (默认值), 实际 = %d", dbPort)
	}
}

func TestGetStringSliceWithDefault(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
cors:
  allowed_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	origins, err := GetStringSliceWithDefault("cors.allowed_origins", []string{"*"})
	if err != nil {
		t.Fatalf("GetStringSliceWithDefault 失败: %v", err)
	}
	expected := []string{"http://localhost:3000", "http://localhost:8080"}
	if len(origins) != len(expected) {
		t.Errorf("期望 origins 长度 = %d, 实际 = %d", len(expected), len(origins))
	}
	for i, origin := range origins {
		if origin != expected[i] {
			t.Errorf("期望 origins[%d] = '%s', 实际 = '%s'", i, expected[i], origin)
		}
	}

	// 测试不存在的配置项，应该返回默认值
	methods, err := GetStringSliceWithDefault("cors.allowed_methods", []string{"GET", "POST"})
	if err != nil {
		t.Fatalf("GetStringSliceWithDefault 失败: %v", err)
	}
	expectedMethods := []string{"GET", "POST"}
	if len(methods) != len(expectedMethods) {
		t.Errorf("期望 methods 长度 = %d, 实际 = %d", len(expectedMethods), len(methods))
	}
}

func TestIsSet(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  port: 8080
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	exists, err := IsSet("app.name")
	if err != nil {
		t.Fatalf("IsSet 失败: %v", err)
	}
	if !exists {
		t.Error("期望 app.name 已设置")
	}

	exists, err = IsSet("app.port")
	if err != nil {
		t.Fatalf("IsSet 失败: %v", err)
	}
	if !exists {
		t.Error("期望 app.port 已设置")
	}

	// 测试不存在的配置项
	exists, err = IsSet("database.host")
	if err != nil {
		t.Fatalf("IsSet 失败: %v", err)
	}
	if exists {
		t.Error("期望 database.host 未设置")
	}
}

func TestAllKeys(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  port: 8080

database:
  host: "localhost"
  port: 5432
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	keys, err := AllKeys()
	if err != nil {
		t.Fatalf("AllKeys 失败: %v", err)
	}

	expectedKeys := []string{"app.name", "app.port", "database.host", "database.port"}

	if len(keys) < len(expectedKeys) {
		t.Errorf("期望至少 %d 个配置键, 实际 = %d", len(expectedKeys), len(keys))
	}

	// 检查期望的键是否都存在
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("期望配置键 '%s' 存在", expectedKey)
		}
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	var cfg TestConfig
	err := LoadConfig(&cfg, "nonexistent.yml")

	if err == nil {
		t.Error("期望返回错误，但没有返回错误")
	}

	if !contains(err.Error(), "配置文件未找到") {
		t.Errorf("期望错误消息包含 '配置文件未找到', 实际错误: %v", err)
	}
}

func TestLoadConfig_JSONFormat(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建临时 JSON 配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	configContent := `{
  "app": {
    "name": "JSON App",
    "version": "1.0.0",
    "port": 8080,
    "debug": false
  },
  "database": {
    "host": "json-host",
    "port": 5432,
    "username": "jsonuser",
    "password": "jsonpass"
  }
}`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时 JSON 配置文件失败: %v", err)
	}

	var cfg TestConfig
	err = LoadConfig(&cfg, configFile)
	if err != nil {
		t.Fatalf("加载 JSON 配置失败: %v", err)
	}

	// 验证配置值
	if cfg.App.Name != "JSON App" {
		t.Errorf("期望 App.Name = 'JSON App', 实际 = '%s'", cfg.App.Name)
	}
	if cfg.Database.Host != "json-host" {
		t.Errorf("期望 Database.Host = 'json-host', 实际 = '%s'", cfg.Database.Host)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// 重置全局状态确保测试隔离
	ResetGlobalState()

	// 创建无效的 YAML 文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid.yml")

	invalidContent := `
app:
  name: "Invalid YAML
  missing_quotes_and_colon
    port: 8080
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("创建无效配置文件失败: %v", err)
	}

	var cfg TestConfig
	err = LoadConfig(&cfg, configFile)

	if err == nil {
		t.Error("期望返回解析错误，但没有返回错误")
	}
}

// TestMustGetClient 测试MustGetClient函数
func TestMustGetClient(t *testing.T) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  port: 8080
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试正常情况
	client := MustGetClient()
	if client == nil {
		t.Fatal("MustGetClient 返回 nil")
	}

	value := client.GetString("app.name")
	if value != "Test App" {
		t.Errorf("期望 app.name = 'Test App', 实际 = '%s'", value)
	}
}

// TestMustGetStringWithDefault 测试MustGetStringWithDefault函数
func TestMustGetStringWithDefault(t *testing.T) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 测试存在的配置项
	appName := MustGetStringWithDefault("app.name", "默认值")
	if appName != "Test App" {
		t.Errorf("期望 app.name = 'Test App', 实际 = '%s'", appName)
	}

	// 测试不存在的配置项
	logLevel := MustGetStringWithDefault("logging.level", "info")
	if logLevel != "info" {
		t.Errorf("期望 logging.level = 'info', 实际 = '%s'", logLevel)
	}
}

// TestCleanup 测试Cleanup函数
func TestCleanup(t *testing.T) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 初始化配置
	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	// 验证配置已加载
	globalMutex.RLock()
	initialized := isInitialized
	globalMutex.RUnlock()

	if !initialized {
		t.Error("期望配置已初始化")
	}

	// 清理配置
	Cleanup()

	// 验证配置已清理
	globalMutex.RLock()
	initialized = isInitialized
	globalMutex.RUnlock()

	if initialized {
		t.Error("期望配置已清理")
	}
}

// TestGetClient_ErrorHandling 测试GetClient的错误处理
func TestGetClient_ErrorHandling(t *testing.T) {
	ResetGlobalState()

	// 测试文件不存在的情况
	_, err := GetClient("nonexistent.yml")
	if err == nil {
		t.Error("期望返回错误，但没有返回错误")
	}

	if !contains(err.Error(), "配置文件未找到") {
		t.Errorf("期望错误消息包含 '配置文件未找到', 实际错误: %v", err)
	}
}

// TestConcurrentAccess 测试简化的并发访问配置
func TestConcurrentAccess(t *testing.T) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Test App"
  port: 8080
  debug: true

database:
  host: "localhost"
  port: 5432
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 预先加载配置
	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	// 简化的并发测试参数
	const numGoroutines = 10
	const numOperations = 10

	// 使用WaitGroup等待所有goroutine完成
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// 启动多个goroutine并发访问配置
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 测试GetClient
				client, err := GetClient()
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: GetClient 失败: %w", id, err)
					return
				}

				// 测试配置读取
				appName := client.GetString("app.name")
				if appName != "Test App" {
					errors <- fmt.Errorf("goroutine %d: 期望 app.name = 'Test App', 实际 = '%s'", id, appName)
					return
				}

				// 测试便利函数
				port, err := GetIntWithDefault("app.port", 3000)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: GetIntWithDefault 失败: %w", id, err)
					return
				}
				if port != 8080 {
					errors <- fmt.Errorf("goroutine %d: 期望 port = 8080, 实际 = %d", id, port)
					return
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errors)

	// 检查是否有错误
	for err := range errors {
		t.Error(err)
	}
}

// BenchmarkGetClient 基准测试GetClient性能
func BenchmarkGetClient(b *testing.B) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Benchmark App"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
  max_connections: 100
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 预先加载配置
	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		b.Fatalf("LoadConfig 失败: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client, err := GetClient()
			if err != nil {
				b.Fatalf("GetClient 失败: %v", err)
			}
			_ = client.GetString("app.name")
		}
	})
}

// BenchmarkGetStringWithDefault 基准测试GetStringWithDefault性能
func BenchmarkGetStringWithDefault(b *testing.B) {
	ResetGlobalState()

	// 创建临时配置文件
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Benchmark App"
  environment: "test"
  debug: false

database:
  host: "localhost"
  name: "testdb"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// 预先加载配置
	var cfg TestConfig
	err = LoadConfig(&cfg)
	if err != nil {
		b.Fatalf("LoadConfig 失败: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := GetStringWithDefault("app.name", "default")
			if err != nil {
				b.Fatalf("GetStringWithDefault 失败: %v", err)
			}
		}
	})
}

// BenchmarkLoadConfig 基准测试LoadConfig性能
func BenchmarkLoadConfig(b *testing.B) {
	// 创建临时配置文件
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	configContent := `
app:
  name: "Benchmark App"
  version: "1.0.0"
  port: 8080
  debug: false

database:
  host: "localhost"
  port: 5432
  max_connections: 100
  username: "testuser"
  password: "testpass"

logging:
  level: "info"
  format: "json"
  output: "/var/log/app.log"

cache:
  enabled: true
  ttl: 3600
  size: 1000
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("创建临时配置文件失败: %v", err)
	}

	// 切换到临时目录
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResetGlobalState() // 每次重置以测试冷启动性能

		var cfg TestConfig
		err := LoadConfig(&cfg)
		if err != nil {
			b.Fatalf("LoadConfig 失败: %v", err)
		}
	}
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
