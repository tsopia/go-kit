# 数据库配置 - 指针类型示例

这个示例展示了如何使用指针类型来解决 bool 字段默认值的问题。

## 问题背景

在 Go 中，bool 类型的零值是 `false`，这导致无法区分以下两种情况：
1. 用户明确设置 `false`
2. 用户没有设置，使用默认值

## 解决方案

使用指针类型 `*bool` 来解决这个问题：

```go
type Config struct {
    // 其他字段...
    IgnoreRecordNotFoundError *bool `mapstructure:"ignore_record_not_found_error" json:"ignore_record_not_found_error" yaml:"ignore_record_not_found_error"`
    ParameterizedQueries      *bool `mapstructure:"parameterized_queries" json:"parameterized_queries" yaml:"parameterized_queries"`
}
```

## 使用方法

### 1. 使用默认值（推荐）

```go
config := &database.Config{
    Driver:   "sqlite",
    Database: ":memory:",
    LogLevel: "info",
    // IgnoreRecordNotFoundError 和 ParameterizedQueries 未设置
    // 将使用默认值 true
}
```

### 2. 明确设置值

```go
trueVal := true
falseVal := false

config := &database.Config{
    Driver:                    "sqlite",
    Database:                  ":memory:",
    LogLevel:                  "info",
    IgnoreRecordNotFoundError: &falseVal,  // 明确设置为 false
    ParameterizedQueries:      &trueVal,   // 明确设置为 true
}
```

### 3. 混合设置

```go
trueVal := true

config := &database.Config{
    Driver:                    "sqlite",
    Database:                  ":memory:",
    LogLevel:                  "info",
    IgnoreRecordNotFoundError: &trueVal,   // 明确设置为 true
    // ParameterizedQueries 未设置，将使用默认值 true
}
```

## 优势

1. **明确区分**: 可以清楚地区分用户设置的值和默认值
2. **向后兼容**: 现有的配置文件无需修改，未设置的字段会使用默认值
3. **灵活性**: 用户可以明确控制每个配置项
4. **类型安全**: 编译时检查，避免运行时错误

## 默认值

- `IgnoreRecordNotFoundError`: `true` (忽略记录未找到错误)
- `ParameterizedQueries`: `true` (使用参数化查询)

## 运行示例

```bash
go run main.go
```

输出示例：
```
=== 示例1: 使用默认值 ===
数据库连接成功，使用默认配置

=== 示例2: 明确设置配置值 ===
数据库连接成功，使用明确设置的配置

=== 示例3: 混合设置 ===
数据库连接成功，混合配置

=== 示例4: 使用SetLogger设置自定义日志记录器 ===
自定义日志记录器设置成功

=== 测试数据库操作 ===
2025-07-17 15:53:27     INFO    database/gorm_adapter.go:116    SQL执行 {"elapsed": 0.000096765, "rows": -1, "sql": "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=\"users\""}
2025-07-17 15:53:27     INFO    database/gorm_adapter.go:116    SQL执行 {"elapsed": 0.000788561, "rows": 0, "sql": "CREATE TABLE `users` (`id` integer PRIMARY KEY AUTOINCREMENT,`name` text NOT NULL,`email` text NOT NULL,`age` integer DEFAULT 0,`created_at` datetime,`updated_at` datetime)"}
用户创建成功: ID=1, Name=张三, Email=zhangsan@example.com
查询用户成功: ID=1, Name=张三, Email=zhangsan@example.com
连接池统计: 打开连接=1, 空闲连接=1

所有测试完成！
```

## SetLogger 功能（通用设计）

### 设计理念

采用通用的 `LogClient` 接口设计，不耦合特定的日志包，让用户可以传入自己的日志客户端：

```go
// LogClient 通用日志客户端接口
type LogClient interface {
    Info(ctx context.Context, msg string, fields ...interface{})
    Warn(ctx context.Context, msg string, fields ...interface{})
    Error(ctx context.Context, msg string, fields ...interface{})
    Debug(ctx context.Context, msg string, fields ...interface{})
}
```

### 使用项目的 logger 包

```go
// 创建项目的logger
kitLogger := kitlogger.NewWithOptions(kitlogger.Options{
    Level:      kitlogger.InfoLevel,
    Format:     kitlogger.FormatConsole,
    TimeFormat: "2006-01-02 15:04:05",
    Caller:     true,
    Stacktrace: false,
})

// 创建项目的logger适配器
kitAdapter := database.NewKitLoggerAdapter(kitLogger)

// 创建GORM日志配置
gormConfig := gormlogger.Config{
    SlowThreshold:             200 * time.Millisecond,
    LogLevel:                  gormlogger.Info,
    IgnoreRecordNotFoundError: false,
    ParameterizedQueries:      true,
    Colorful:                  false,
}

// 创建GORM适配器并设置
adapter := database.NewGormLoggerAdapter(kitAdapter, gormConfig)
db.SetLogger(adapter)
```

### 使用自定义日志客户端

```go
// 自定义日志客户端
type CustomLogger struct {
    prefix string
}

func (c *CustomLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
    fmt.Printf("%s INFO: %s\n", c.prefix, msg)
}

func (c *CustomLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
    fmt.Printf("%s WARN: %s\n", c.prefix, msg)
}

func (c *CustomLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
    fmt.Printf("%s ERROR: %s\n", c.prefix, msg)
}

func (c *CustomLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
    fmt.Printf("%s DEBUG: %s\n", c.prefix, msg)
}

// 使用自定义日志客户端
customLogger := &CustomLogger{prefix: "[CUSTOM]"}
adapter := database.NewGormLoggerAdapter(customLogger, gormConfig)
db.SetLogger(adapter)
```

### 优势

1. **解耦设计**: 不耦合特定的日志包，用户可以传入任何实现 `LogClient` 接口的日志客户端
2. **统一接口**: 提供统一的 `LogClient` 接口，便于扩展和测试
3. **灵活配置**: 支持各种日志格式和输出方式
4. **性能监控**: 记录SQL执行时间和影响行数
5. **慢查询检测**: 自动检测和警告慢查询
6. **上下文支持**: 支持context传递，便于分布式追踪 