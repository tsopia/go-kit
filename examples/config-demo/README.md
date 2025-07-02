# Go-Kit 配置系统演示

这个示例演示了如何使用 Go-Kit 的配置加载功能，基于 Viper 库实现。

## 功能特性

- ✅ **默认配置文件路径**：自动在项目根目录查找 `config.yml`
- ✅ **自定义配置文件路径**：支持指定任意配置文件路径
- ✅ **环境变量覆盖**：环境变量优先于配置文件中的值
- ✅ **自动前缀检测**：基于 `APP_NAME` 环境变量自动启用前缀模式
- ✅ **多种配置格式**：支持 YAML、JSON、TOML 等格式
- ✅ **结构体绑定**：直接解析到 Go 结构体
- ✅ **优先级保证**：当配置文件和环境变量都有 `app_name` 时，环境变量优先级最高

## 运行演示

### 1. 基本演示
```bash
cd examples/config-demo
go run main.go
```

### 2. 测试环境变量覆盖（无前缀模式）
```bash
# 不设置 APP_NAME，使用无前缀环境变量
export SERVER_HOST="从环境变量设置的主机"
export SERVER_PORT="9999"
export DATABASE_HOST="env-database-host"

# 运行程序
go run main.go
```

### 3. 测试 APP_NAME 自动前缀
```bash
# 设置 APP_NAME 启用前缀模式
export APP_NAME="myapp"
export MYAPP_SERVER_HOST="前缀模式主机"
export MYAPP_SERVER_PORT="7777"

# 运行程序
go run main.go
```

### 4. 测试 APP_NAME 优先级
```bash
# 设置 APP_NAME 和对应的前缀环境变量
export APP_NAME="priority"
export PRIORITY_APP_NAME="环境变量优先的应用名"

# 运行程序 - 环境变量会覆盖配置文件中的 app.name
go run main.go
```

## 配置文件示例

### config.yml (默认配置)
```yaml
app:
  name: "Go-Kit 示例应用"
  version: "1.0.0"
  port: 8080
  environment: "development"
  debug: false

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "password"
  dbname: "gokit_db"
  sslmode: "disable"
```

## 使用方法

### 1. 定义配置结构体
```go
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
}
```

### 2. 加载配置

#### 使用默认路径
```go
var cfg AppConfig
err := config.LoadConfig(&cfg)
if err != nil {
    log.Fatal(err)
}
```

#### 使用自定义路径
```go
var cfg AppConfig
err := config.LoadConfig(&cfg, "path/to/custom-config.yml")
if err != nil {
    log.Fatal(err)
}
```

## 环境变量映射规则

### 无前缀模式（未设置 APP_NAME）

配置键中的点（`.`）会被替换为下划线（`_`）并转换为大写：

| 配置键 | 环境变量 |
|--------|----------|
| `app.name` | `APP_NAME` |
| `app.port` | `APP_PORT` |
| `database.host` | `DATABASE_HOST` |
| `database.port` | `DATABASE_PORT` |
| `logging.level` | `LOGGING_LEVEL` |

### 前缀模式（设置了 APP_NAME）

当设置 `APP_NAME` 环境变量时，会自动启用前缀模式：

| APP_NAME值 | 前缀 | 配置键 | 环境变量 |
|------------|------|--------|----------|
| `myapp` | `MYAPP_` | `app.name` | `MYAPP_APP_NAME` |
| `myapp` | `MYAPP_` | `app.port` | `MYAPP_APP_PORT` |
| `myapp` | `MYAPP_` | `database.host` | `MYAPP_DATABASE_HOST` |
| `prod` | `PROD_` | `app.name` | `PROD_APP_NAME` |

### APP_NAME 的特殊作用

1. **前缀控制**：`APP_NAME` 的值决定环境变量前缀
2. **自身覆盖**：在前缀模式下，`{PREFIX}_APP_NAME` 环境变量会覆盖配置文件中的 `app.name`
3. **优先级最高**：环境变量始终优先于配置文件中的值

#### 示例：
```bash
# 配置文件中: app.name = "My App"
export APP_NAME="myapp"           # 启用前缀 MYAPP_
export MYAPP_APP_NAME="Prod App"  # 覆盖 app.name 的值

# 结果: app.name = "Prod App"
```

## 配置文件搜索路径

函数会按以下顺序搜索配置文件：
1. 指定的自定义路径（如果提供）
2. 当前工作目录 (`.`)
3. `./configs` 目录
4. `./config` 目录
5. 项目根目录

## 支持的配置文件格式

- YAML (`.yml`, `.yaml`)
- JSON (`.json`)
- TOML (`.toml`)
- HCL (`.hcl`)
- INI (`.ini`)
- Properties (`.properties`)

## 错误处理

函数提供详细的错误信息：
- 配置文件未找到
- 配置文件格式错误
- 结构体解析失败

```go
err := config.LoadConfig(&cfg)
if err != nil {
    log.Printf("配置加载失败: %v", err)
    // 处理错误...
}
```

## 最佳实践

### 1. 开发环境
```bash
# 不设置 APP_NAME，使用简单的环境变量
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
export DEBUG="true"
```

### 2. 生产环境
```bash
# 设置 APP_NAME 避免环境变量冲突
export APP_NAME="myapp"
export MYAPP_APP_NAME="生产应用"
export MYAPP_DATABASE_HOST="prod-db-server"
export MYAPP_DATABASE_PASSWORD="secure-password"
```

### 3. 配置验证
```go
var cfg AppConfig
err := config.LoadConfig(&cfg)
if err != nil {
    log.Fatal("配置加载失败:", err)
}

// 验证必需配置
if cfg.Database.Host == "" {
    log.Fatal("数据库主机未设置")
}
if cfg.App.Port <= 0 {
    log.Fatal("应用端口配置无效")
}
```

## API 简化

新的配置系统将原来的两个函数合并为一个：

```go
// ❌ 旧版本（已删除）
config.LoadConfig(&cfg)
config.LoadConfigWithEnvPrefix(&cfg, "MYAPP")

// ✅ 新版本（统一API）
config.LoadConfig(&cfg)                    // 自动检测前缀
config.LoadConfig(&cfg, "custom-path.yml") // 自定义路径 + 自动检测前缀
```

APP_NAME 环境变量的存在与否决定了使用哪种模式，无需额外的函数参数。 