# cfg - Go 配置管理包

基于 [spf13/viper](https://github.com/spf13/viper) 的简化封装，提供统一的配置管理接口，支持配置文件、环境变量、.env 文件等多种配置源。

## 设计理念

- **零配置**：大多数情况下无需配置，自动发现配置文件
- **简洁 API**：只有 `New()` 和 `NewWithPrefix()` 两个构造函数
- **显式错误处理**：所有 Get 方法返回 `(value, error)`，业务代码自行决定如何处理错误
- **可选默认值**：通过变长参数支持默认值，不修改内部状态

## 快速开始

### 1. 创建 Provider（推荐）

```go
// 自动查找配置文件（从当前目录向上查找3层，查找 config.yml/yaml/json/toml/env）
provider, err := cfg.New()
if err != nil {
    log.Fatal(err)
}

// 使用显式路径
provider, err := cfg.New("./config/app.yaml")

// 带环境变量前缀（环境变量如 MYAPP_DB_HOST 会覆盖配置文件的 db.host）
provider, err := cfg.NewWithPrefix("MYAPP")
provider, err := cfg.NewWithPrefix("MYAPP", "./config/app.yaml")
```

### 2. 使用全局实例（便捷方式）

```go
// 初始化（只需一次，通常在 main 函数中）
if err := cfg.Init(); err != nil {
    log.Fatal(err)
}
// 或指定路径
if err := cfg.Init("./config.yaml"); err != nil {
    log.Fatal(err)
}
// 或带前缀
if err := cfg.InitWithPrefix("MYAPP"); err != nil {
    log.Fatal(err)
}

// 全局读取
name, err := cfg.GetString("app.name")
if err != nil {
    // 处理错误：键不存在且未提供默认值
}

port, _ := cfg.GetInt("server.port", 8080)  // 提供默认值，错误可忽略
debug, _ := cfg.GetBool("app.debug", false)
```

### 3. 读取配置值

```go
// 基本类型 - 返回 (value, error)
str, err := provider.GetString("db.host")
if err != nil {
    log.Fatal(err)
}

// 使用默认值（键不存在时返回默认值，不报错）
port, _ := provider.GetInt("db.port", 5432)
debug, _ := provider.GetBool("app.debug", false)
timeout, _ := provider.GetDuration("server.timeout", 30*time.Second)

// 检查是否存在
if provider.Exists("db.password") {
    // ...
}

// 获取原始值（需要类型断言）
rawValue := provider.Get("db.port")
if port, ok := rawValue.(int); ok {
    fmt.Println(port)
}

// 获取子配置（支持嵌套）
dbConfig := provider.Sub("db")
host, _ := dbConfig.GetString("host")  // 等同于 provider.GetString("db.host")
```

### 4. 反序列化到结构体

```go
// 定义配置结构体
type Config struct {
    App struct {
        Name    string `mapstructure:"name"`
        Version string `mapstructure:"version"`
        Debug   bool   `mapstructure:"debug"`
    } `mapstructure:"app"`
    DB struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        Username string `mapstructure:"username"`
        Password string `mapstructure:"password"`
        Pool     struct {
            MaxOpen int `mapstructure:"max_open"`
            MaxIdle int `mapstructure:"max_idle"`
        } `mapstructure:"pool"`
    } `mapstructure:"db"`
    Server struct {
        Port    int           `mapstructure:"port"`
        Timeout time.Duration `mapstructure:"timeout"`
    } `mapstructure:"server"`
}

// 反序列化整个配置
var cfg Config
if err := provider.Unmarshal(&cfg); err != nil {
    log.Fatal(err)
}

// 使用配置
fmt.Println(cfg.App.Name)
fmt.Println(cfg.DB.Host)
fmt.Println(cfg.DB.Pool.MaxOpen)

// 也可以只反序列化子配置
dbProvider := provider.Sub("db")
var dbConfig DB  // 只包含 db 相关字段的结构体
if err := dbProvider.Unmarshal(&dbConfig); err != nil {
    log.Fatal(err)
}
```

### 5. Mapstructure Tag 详解

`mapstructure` tag 控制配置键与结构体字段的映射关系：

#### 基本映射
```go
type Config struct {
    // 配置键名与字段名不同时使用
    HostName string `mapstructure:"host"`      // 对应配置中的 host
    DBPort   int    `mapstructure:"db_port"`   // 对应配置中的 db_port
}
```

#### 忽略字段
```go
type Config struct {
    Name     string `mapstructure:"name"`
    Password string `mapstructure:"-"`  // 忽略此字段，不参与反序列化
}
```

#### 嵌入结构体（扁平化）
```go
// 配置文件: host: localhost, port: 5432
type CommonConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}

type DBConfig struct {
    CommonConfig `mapstructure:",squash"`  // 扁平化嵌入，配置中直接写 host, port
    Database     string `mapstructure:"database"`
}
// 结果: DBConfig.Host = "localhost", DBConfig.Port = 5432
```

#### 捕获剩余字段
```go
type Config struct {
    Name    string                 `mapstructure:"name"`
    Extra   map[string]interface{} `mapstructure:",remain"`  // 捕获未映射的字段
}
// 配置文件: name: myapp, debug: true, version: 1.0
// 结果: Extra = {"debug": true, "version": "1.0"}
```

#### 嵌套切片
```go
type Config struct {
    Servers []struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"servers"`
}
// 配置文件:
// servers:
//   - host: srv1
//     port: 8080
//   - host: srv2
//     port: 8081
```

#### 弱类型解析
```go
type Config struct {
    Port int `mapstructure:"port"`  // 配置中是字符串 "5432" 也能解析为 int
}
// 注意：viper 默认启用弱类型解析，字符串 "5432" 会自动转为 5432
```

## 支持的配置文件格式

### YAML (推荐)
```yaml
app:
  name: myapp
  version: 1.0.0
  debug: true

db:
  host: localhost
  port: 5432
  pool:
    max_open: 10
    max_idle: 5

server:
  port: 8080
  timeout: 30s
```

### JSON
```json
{
  "app": {
    "name": "myapp",
    "debug": true
  },
  "db": {
    "host": "localhost",
    "port": 5432
  }
}
```

### TOML
```toml
[app]
name = "myapp"
debug = true

[db]
host = "localhost"
port = 5432

[db.pool]
max_open = 10
```

### .env
```bash
APP_NAME=myapp
APP_DEBUG=true
DB_HOST=localhost
DB_PORT=5432
DB_POOL_MAX_OPEN=10
```

## 配置文件查找顺序

1. 如果调用 `New("./path/to/config.yaml")`，直接使用该路径
2. 否则，从当前目录开始，向上查找最多3层父目录
3. 查找名为 `config` 的文件，支持以下扩展名（按优先级）：
   - `.yml`
   - `.yaml`
   - `.json`
   - `.toml`
   - `.env`

## 环境变量映射

配置文件中的键通过以下规则映射到环境变量：

- `app.name` → `MYAPP_APP_NAME`
- `db.host` → `MYAPP_DB_HOST`
- `db.pool.max_open` → `MYAPP_DB_POOL_MAX_OPEN`

```bash
# 设置环境变量会覆盖配置文件中的值
export MYAPP_DB_HOST=production.db.com
export MYAPP_DB_PORT=3306
```

## 配置优先级（由高到低）

1. **环境变量**（如 `MYAPP_DB_HOST`）
2. **配置文件**（config.yml / config.json / .env 等）
3. **默认值**（代码中设置的默认值）

## 时间格式说明

### GetDuration

支持 Go 的 duration 格式：

```go
// 支持的格式
"1h30m"      // 1小时30分钟
"90m"        // 90分钟
"2s"         // 2秒
"500ms"      // 500毫秒
"1h30m10s"   // 1小时30分10秒

// 使用示例
timeout, err := provider.GetDuration("server.timeout")
// 或使用默认值
timeout, _ := provider.GetDuration("server.timeout", 30*time.Second)
```

### GetSizeInBytes

解析人类可读的大小格式：

```go
// 支持的格式
"1GB"      // 1 GB
"100MB"    // 100 MB
"512KB"    // 512 KB
"1024B"    // 1024 字节

// 使用示例
maxSize, err := provider.GetSizeInBytes("upload.max_size")
// 或使用默认值
maxSize, _ := provider.GetSizeInBytes("upload.max_size", 100*1024*1024)  // 默认 100MB
```

## 特殊 Provider

### Map Provider（测试用）

创建一个独立的、不受环境变量影响的 Provider：

```go
data := map[string]any{
    "app": map[string]any{
        "name": "test-app",
    },
    "db": map[string]any{
        "host": "localhost",
        "port": 5432,
    },
}
provider := cfg.NewFromMap(data)
```

### Empty Provider

返回零值的空 Provider，用于安全链式调用：

```go
empty := cfg.EmptyProvider()
// 所有 Get 方法返回 ErrNotFound（或默认值如果提供）
// Exists 返回 false，Sub 返回自身

host, err := empty.GetString("host")  // 返回 ("", ErrNotFound)
port, _ := empty.GetInt("port", 3306) // 返回 (3306, nil)
```

## 链式调用安全

所有 Provider 都支持安全的链式调用：

```go
// 即使键不存在也不会 panic
value, _ := provider.Sub("nonexistent").Sub("nested").GetInt("key")  // 返回 (0, ErrNotFound)

// 链式调用中使用默认值
timeout, _ := provider.Sub("server").GetDuration("timeout", 30*time.Second)
```

## 完整示例

```go
package main

import (
    "errors"
    "log"
    "github.com/your-org/go-kit/cfg"
)

func main() {
    // 方式1：使用全局实例（推荐）
    if err := cfg.InitWithPrefix("MYAPP"); err != nil {
        log.Fatal(err)
    }

    // 关键配置：严格处理错误
    dbHost, err := cfg.GetString("db.host")
    if err != nil {
        log.Fatal("db.host is required")
    }

    // 可选配置：使用默认值
    dbPort, _ := cfg.GetInt("db.port", 5432)

    // 方式2：创建独立 Provider
    provider, err := cfg.New("./config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 读取子配置
    db := provider.Sub("database")
    maxConns, _ := db.GetInt("pool.max_open", 10)

    // 检查特定错误类型
    timeout, err := cfg.GetDuration("server.timeout")
    if errors.Is(err, cfg.ErrNotFound) {
        log.Println("using default timeout")
        timeout = 30 * time.Second
    }
}
```

## API 参考

### 构造函数

```go
// 自动查找配置文件
cfg.New()
cfg.New("./path/to/config.yaml")

// 带环境变量前缀
cfg.NewWithPrefix("MYAPP")
cfg.NewWithPrefix("MYAPP", "./path/to/config.yaml")

// 从 map 创建（测试用）
cfg.NewFromMap(map[string]any{...})

// 空 Provider
cfg.EmptyProvider()
```

### 全局函数

```go
// 初始化
cfg.Init()
cfg.Init("./config.yaml")
cfg.InitWithPrefix("MYAPP")
cfg.InitWithPrefix("MYAPP", "./config.yaml")

// 获取全局 Provider
cfg.Default()
cfg.IsInitialized()

// 读取配置 - 都返回 (value, error)
cfg.GetString(key string, defaultValue ...string) (string, error)
cfg.GetInt(key string, defaultValue ...int) (int, error)
cfg.GetBool(key string, defaultValue ...bool) (bool, error)
cfg.GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error)

// 更多数值类型
cfg.GetInt32(key string, defaultValue ...int32) (int32, error)
cfg.GetInt64(key string, defaultValue ...int64) (int64, error)
cfg.GetUint(key string, defaultValue ...uint) (uint, error)
cfg.GetUint8(key string, defaultValue ...uint8) (uint8, error)
cfg.GetUint16(key string, defaultValue ...uint16) (uint16, error)
cfg.GetUint32(key string, defaultValue ...uint32) (uint32, error)
cfg.GetUint64(key string, defaultValue ...uint64) (uint64, error)
cfg.GetFloat64(key string, defaultValue ...float64) (float64, error)

// 时间和大小
cfg.GetTime(key string, defaultValue ...time.Time) (time.Time, error)
cfg.GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error)  // 解析 "1GB", "100MB"

// 切片类型 - 只返回 error，不支持默认值
cfg.GetStringSlice(key string) ([]string, error)
cfg.GetIntSlice(key string) ([]int, error)

// Map 类型
cfg.GetStringMap(key string) (map[string]any, error)

// 其他
cfg.Get(key string) any
cfg.Exists(key string) bool
cfg.Unmarshal(dst any) error
cfg.Sub(key string) Provider
```

## 错误处理

所有 Get 方法都返回 `(value, error)`，错误类型如下：

```go
var (
    ErrNotFound       = errors.New("cfg: key not found")       // 键不存在且未提供默认值
    ErrTypeMismatch   = errors.New("cfg: type mismatch")       // 默认值类型不匹配或传了多个默认值
    ErrNotInitialized = errors.New("cfg: provider not initialized") // 全局 Provider 未初始化
)
```

### 使用模式

```go
// 模式1：严格处理错误（推荐用于关键配置）
host, err := cfg.GetString("db.host")
if err != nil {
    return fmt.Errorf("db.host is required: %w", err)
}

// 模式2：使用默认值（配置项可选时）
port, _ := cfg.GetInt("db.port", 5432)  // 不存在则使用 5432

// 模式3：检查特定错误
timeout, err := cfg.GetDuration("server.timeout")
if errors.Is(err, cfg.ErrNotFound) {
    // 使用自定义逻辑设置默认值
    timeout = calculateDefaultTimeout()
}
```

## 注意事项

1. **必须先调用 Init**: 使用全局函数前必须先调用 `cfg.Init()`，否则返回 `ErrNotInitialized`
2. **Init 只执行一次**: 多次调用 `Init` 只有第一次生效
3. **错误处理**: `New` 和 `Init` 返回的错误必须处理
4. **线程安全**: Provider 是线程安全的，可以在多个 goroutine 中并发使用
5. **默认值不修改内部状态**: 传入的默认值仅在当前调用中返回，不会写入配置

## 常见陷阱

### Unmarshal 不会清空已有字段
```go
var cfg Config
cfg.Name = "default"

// 如果配置文件中没有 name 字段，cfg.Name 仍保持 "default"
provider.Unmarshal(&cfg)
```

### 环境变量字符串解析
```go
// .env 文件: PORT=8080
// 环境变量: export MYAPP_PORT=9090

// GetInt 可以正确解析
port := provider.GetInt("port")  // 8080 或 9090（取决于优先级）

// 但 GetString 会返回原始字符串
portStr := provider.GetString("port")  // "8080" 或 "9090"
```

### Sub 后 Unmarshal 的区别
```go
// 方式1：反序列化整个配置（包含层级）
var fullConfig Config  // 包含 app, db, server
provider.Unmarshal(&fullConfig)

// 方式2：只反序列化子配置（扁平化）
var dbConfig DB  // 只包含 db 下的字段
dbProvider := provider.Sub("db")
dbProvider.Unmarshal(&dbConfig)
// 注意：此时 dbConfig 的字段直接对应 db.*，不是 db.db.*
```

### 配置键不存在时的行为
```go
// Get 方法返回 error，需要用默认值或处理错误
host, err := provider.GetString("db.nonexistent")
if errors.Is(err, cfg.ErrNotFound) {
    // 键不存在且未提供默认值
}

// 使用默认值时不会返回错误
port, _ := provider.GetInt("db.nonexistent", 3306)  // 返回 3306

// Exists 用于检查键是否存在
if provider.Exists("db.password") {
    // 键存在（即使值为空字符串）
}

// 嵌套键不存在时，Sub 返回 EmptyProvider（安全链式调用）
db := provider.Sub("nonexistent")
host, _ := db.GetString("host")  // 返回 ("", ErrNotFound)，不会 panic
```
