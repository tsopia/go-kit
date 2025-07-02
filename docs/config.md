# 配置管理 (pkg/config)

基于Viper的现代化配置管理系统，支持多种配置格式和环境变量覆盖。

## 🚀 特性

- ✅ 支持YAML、JSON、TOML、HCL等多种格式
- ✅ 环境变量自动覆盖配置文件
- ✅ 支持APP_NAME前缀模式
- ✅ 线程安全的全局配置管理
- ✅ 配置验证和默认值支持
- ✅ 热重载支持（通过Viper）

## 📖 快速开始

### 基本使用

```go
package main

import (
    "log"
    "go-kit/pkg/config"
)

// 配置结构体
type AppConfig struct {
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
    // 加载配置到结构体
    var cfg AppConfig
    if err := config.LoadConfig(&cfg); err != nil {
        log.Fatal(err)
    }
    
    log.Printf("服务器配置: %s:%d", cfg.Server.Host, cfg.Server.Port)
    log.Printf("数据库配置: %s:%d", cfg.Database.Host, cfg.Database.Port)
}
```

### 配置文件示例

**config.yml**
```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  name: "myapp"

debug: false
```

## 🔧 API 参考

### 核心函数

#### LoadConfig
加载配置文件并解析到结构体

```go
// 使用默认配置文件路径 (config.yml)
err := config.LoadConfig(&cfg)

// 使用自定义配置文件路径
err := config.LoadConfig(&cfg, "custom/config.yml")
```

#### GetClient
获取配置客户端，提供完整的Viper功能

```go
client, err := config.GetClient()
if err != nil {
    log.Fatal(err)
}

// 获取配置值
host := client.GetString("server.host")
port := client.GetInt("server.port")
debug := client.GetBool("debug")
```

### 便利函数

#### 带默认值的获取函数

```go
// 字符串配置
host, err := config.GetStringWithDefault("server.host", "localhost")
port, err := config.GetIntWithDefault("server.port", 8080)
debug, err := config.GetBoolWithDefault("debug", false)

// 切片配置
origins, err := config.GetStringSliceWithDefault("cors.allowed_origins", []string{"*"})

// 时间间隔配置
timeout, err := config.GetDurationWithDefault("timeout", 30*time.Second)
```

#### 带验证的获取函数

```go
// 带范围验证的整数配置
port, valid, err := config.GetIntWithValidation("server.port", 8080, 1, 65535)
if !valid {
    log.Printf("端口配置超出范围，使用默认值: %d", port)
}

// 带范围验证的浮点数配置
ratio, valid, err := config.GetFloat64WithValidation("ratio", 0.5, 0.0, 1.0)
```

#### 配置检查函数

```go
// 检查配置项是否存在
exists, err := config.IsSet("server.host")

// 获取所有配置键
keys, err := config.AllKeys()
```

### 全局函数（Must版本）

```go
// 如果失败会panic的版本
client := config.MustGetClient()
host := config.MustGetStringWithDefault("server.host", "localhost")
port := config.MustGetIntWithDefault("server.port", 8080)
```

## 🌍 环境变量支持

### 基本环境变量

配置管理器自动支持环境变量，配置键会自动转换为环境变量名：

```bash
# 配置键: server.host → 环境变量: SERVER_HOST
export SERVER_HOST=localhost
export SERVER_PORT=8080
export DEBUG=true
export DATABASE_HOST=db.example.com
```

### APP_NAME前缀模式

设置`APP_NAME`环境变量启用前缀模式：

```bash
# 启用前缀模式
export APP_NAME=myapp

# 环境变量会自动添加前缀
export MYAPP_SERVER_HOST=localhost
export MYAPP_SERVER_PORT=8080
export MYAPP_DEBUG=true
export MYAPP_DATABASE_HOST=db.example.com
```

### 环境变量优先级

1. 带前缀的环境变量（如果设置了APP_NAME）
2. 无前缀的环境变量
3. 配置文件中的值

## 📁 配置文件查找

### 默认查找路径

如果不指定配置文件路径，会自动在以下位置查找：

1. 当前工作目录
2. `./configs/`
3. `./config/`

### 支持的文件格式

- `config.yml` / `config.yaml`
- `config.json`
- `config.toml`
- `config.hcl`

### 自定义配置文件

```go
// 使用绝对路径
err := config.LoadConfig(&cfg, "/etc/myapp/config.yml")

// 使用相对路径
err := config.LoadConfig(&cfg, "./configs/production.yml")
```

## 🔄 配置重载

### 手动重载

```go
// 清理当前配置
config.Cleanup()

// 重新加载配置
err := config.LoadConfig(&cfg, "new-config.yml")
```

### 热重载（通过Viper）

```go
client, err := config.GetClient()
if err != nil {
    log.Fatal(err)
}

// 监听配置文件变化
client.WatchConfig()
client.OnConfigChange(func(e fsnotify.Event) {
    log.Printf("配置文件发生变化: %s", e.Name)
    // 重新加载配置
    var newCfg AppConfig
    if err := client.Unmarshal(&newCfg); err != nil {
        log.Printf("重新加载配置失败: %v", err)
    }
})
```

## 🏗️ 最佳实践

### 1. 配置结构体设计

```go
type Config struct {
    // 使用mapstructure标签确保字段映射
    App struct {
        Name    string `mapstructure:"name"`
        Version string `mapstructure:"version"`
        Port    int    `mapstructure:"port"`
        Debug   bool   `mapstructure:"debug"`
    } `mapstructure:"app"`
    
    // 嵌套结构体组织相关配置
    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Username string `mapstructure:"username"`
        Password string `mapstructure:"password"`
        Name     string `mapstructure:"name"`
    } `mapstructure:"database"`
    
    // 使用有意义的字段名
    Logging struct {
        Level  string `mapstructure:"level"`
        Format string `mapstructure:"format"`
        Output string `mapstructure:"output"`
    } `mapstructure:"logging"`
}
```

### 2. 环境变量命名

```bash
# 无前缀模式（推荐用于简单应用）
export SERVER_HOST=localhost
export SERVER_PORT=8080
export DEBUG=true

# 前缀模式（推荐用于复杂应用）
export APP_NAME=myapp
export MYAPP_SERVER_HOST=localhost
export MYAPP_SERVER_PORT=8080
export MYAPP_DEBUG=true
```

### 3. 配置验证

```go
func validateConfig(cfg *AppConfig) error {
    if cfg.Server.Host == "" {
        return fmt.Errorf("服务器主机不能为空")
    }
    
    if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
        return fmt.Errorf("服务器端口必须在1-65535范围内")
    }
    
    if cfg.Database.Name == "" {
        return fmt.Errorf("数据库名称不能为空")
    }
    
    return nil
}

func main() {
    var cfg AppConfig
    if err := config.LoadConfig(&cfg); err != nil {
        log.Fatal(err)
    }
    
    if err := validateConfig(&cfg); err != nil {
        log.Fatal(err)
    }
}
```

### 4. 生产环境配置

```go
// 使用环境变量覆盖敏感信息
type ProductionConfig struct {
    Server struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"server"`
    
    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Username string `mapstructure:"username"`
        Password string `mapstructure:"password"` // 通过环境变量设置
        Name     string `mapstructure:"name"`
    } `mapstructure:"database"`
    
    // 敏感配置通过环境变量设置
    Secrets struct {
        APIKey    string `mapstructure:"api_key"`
        JWTSecret string `mapstructure:"jwt_secret"`
    } `mapstructure:"secrets"`
}
```

### 5. 配置分层

```go
// 基础配置
type BaseConfig struct {
    App struct {
        Name    string `mapstructure:"name"`
        Version string `mapstructure:"version"`
    } `mapstructure:"app"`
}

// 环境特定配置
type EnvironmentConfig struct {
    BaseConfig `mapstructure:",squash"`
    
    Server struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"server"`
    
    Database struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"database"`
}

func loadEnvironmentConfig(env string) (*EnvironmentConfig, error) {
    var cfg EnvironmentConfig
    
    // 加载基础配置
    if err := config.LoadConfig(&cfg.BaseConfig, "config/base.yml"); err != nil {
        return nil, err
    }
    
    // 加载环境特定配置
    envFile := fmt.Sprintf("config/%s.yml", env)
    if err := config.LoadConfig(&cfg, envFile); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}
```

## 🧪 测试

### 单元测试

```go
func TestLoadConfig(t *testing.T) {
    // 创建临时配置文件
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "config.yml")
    
    configContent := `
app:
  name: "Test App"
  port: 8080
database:
  host: "localhost"
  port: 3306
`
    
    err := os.WriteFile(configFile, []byte(configContent), 0644)
    if err != nil {
        t.Fatalf("创建临时配置文件失败: %v", err)
    }
    
    // 切换到临时目录
    oldDir, _ := os.Getwd()
    defer os.Chdir(oldDir)
    os.Chdir(tempDir)
    
    var cfg AppConfig
    err = config.LoadConfig(&cfg)
    if err != nil {
        t.Fatalf("加载配置失败: %v", err)
    }
    
    // 验证配置值
    if cfg.App.Name != "Test App" {
        t.Errorf("期望 App.Name = 'Test App', 实际 = '%s'", cfg.App.Name)
    }
}
```

### 环境变量测试

```go
func TestEnvironmentOverride(t *testing.T) {
    // 设置环境变量
    os.Setenv("SERVER_HOST", "env-host")
    os.Setenv("SERVER_PORT", "9999")
    defer func() {
        os.Unsetenv("SERVER_HOST")
        os.Unsetenv("SERVER_PORT")
    }()
    
    var cfg AppConfig
    err := config.LoadConfig(&cfg)
    if err != nil {
        t.Fatalf("加载配置失败: %v", err)
    }
    
    // 验证环境变量覆盖了配置文件
    if cfg.Server.Host != "env-host" {
        t.Errorf("期望 Server.Host = 'env-host', 实际 = '%s'", cfg.Server.Host)
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 配置文件未找到

```bash
# 错误信息
配置文件未找到: config.yml。请确保配置文件存在于正确的路径

# 解决方案
# 1. 确保配置文件存在
ls -la config.yml

# 2. 使用绝对路径
config.LoadConfig(&cfg, "/absolute/path/to/config.yml")

# 3. 检查工作目录
pwd
```

#### 2. 环境变量未生效

```bash
# 检查环境变量是否正确设置
echo $SERVER_HOST
echo $APP_NAME

# 检查环境变量格式
# 正确: SERVER_HOST=localhost
# 错误: server.host=localhost
```

#### 3. 配置字段映射失败

```go
// 确保使用mapstructure标签
type Config struct {
    Server struct {
        Host string `mapstructure:"host"` // 必须添加这个标签
        Port int    `mapstructure:"port"`
    } `mapstructure:"server"`
}
```

### 调试技巧

```go
// 1. 检查所有配置键
keys, err := config.AllKeys()
if err != nil {
    log.Printf("获取配置键失败: %v", err)
} else {
    log.Printf("所有配置键: %v", keys)
}

// 2. 检查配置项是否存在
exists, err := config.IsSet("server.host")
if err != nil {
    log.Printf("检查配置失败: %v", err)
} else {
    log.Printf("server.host 存在: %t", exists)
}

// 3. 获取配置客户端进行调试
client, err := config.GetClient()
if err != nil {
    log.Printf("获取配置客户端失败: %v", err)
} else {
    // 使用Viper的所有调试功能
    log.Printf("所有设置: %v", client.AllSettings())
}
```

## 📚 相关链接

- [Viper官方文档](https://github.com/spf13/viper)
- [示例项目](./examples/basic-config/)
- [返回首页](../README.md) 