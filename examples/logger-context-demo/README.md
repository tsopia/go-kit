# 日志系统配置演示

本示例展示了Go-Kit日志系统的新配置方式，主要改进包括：

## 主要变化

### 1. 简化的构造函数
- **移除了多余的构造函数**：`NewFileLogger`、`NewMultiLogger`等
- **保留核心构造函数**：`New()`、`NewWithOptions()`、`NewDevelopment()`、`NewProduction()`

### 2. 统一的输出策略
- **始终输出到stdout**：确保日志在控制台可见
- **可选的文件输出**：通过`EnableFileOutput`控制是否同时写入文件
- **移除Output字段**：不再需要指定输出目标

### 3. 新的配置选项

```go
type Options struct {
    Level            Level                  // 日志级别
    Format           Format                 // 输出格式 (FormatJSON, FormatConsole, FormatText)
    TimeFormat       string                 // 时间格式
    Caller           bool                   // 是否显示调用者信息
    Stacktrace       bool                   // 是否显示堆栈跟踪
    EnableFileOutput bool                   // 是否启用文件输出 (新增)
    Sampling         *SamplingConfig        // 采样配置
    Rotate           *RotateConfig          // 日志轮转配置
    Fields           map[string]interface{} // 默认字段
    Hooks            []Hook                 // 钩子函数
}
```

### 4. 格式常量

新增了类型安全的格式常量，避免硬编码字符串：

```go
// 格式常量
const (
    FormatJSON    Format = "json"     // JSON格式，适合日志收集
    FormatConsole Format = "console"  // 控制台格式，带颜色
    FormatText    Format = "text"     // 文本格式，不带颜色
)

// 使用示例
logger := logger.NewWithOptions(logger.Options{
    Level:  logger.InfoLevel,
    Format: logger.FormatJSON,  // 使用常量而非字符串
})

// 格式解析
format := logger.ParseFormat("json")  // 返回 FormatJSON
formatStr := logger.FormatJSON.String()  // 返回 "json"
```

**格式说明：**
- `FormatJSON` - JSON格式输出，适合日志收集系统和生产环境
- `FormatConsole` - 控制台格式输出，带颜色，适合开发环境
- `FormatText` - 文本格式输出，不带颜色，适合文件输出

### 5. 默认日志文件路径配置

新增了可配置的默认日志文件路径功能：

```go
// 设置默认日志目录和文件名
logger.SetDefaultLogDir("app_logs")
logger.SetDefaultLogFile("application.log")

// 获取配置
fmt.Printf("默认日志路径: %s\n", logger.GetDefaultLogPath())

// 确保目录存在
if err := logger.EnsureLogDir(); err != nil {
    logger.Error("创建日志目录失败", "error", err)
}

// 测试环境清理（主要用于测试）
defer logger.CleanupLogFiles()
```

**路径配置函数：**
- `SetDefaultLogDir(dir string)` - 设置默认日志目录
- `SetDefaultLogFile(file string)` - 设置默认日志文件名
- `GetDefaultLogPath() string` - 获取完整的默认日志路径
- `EnsureLogDir() error` - 确保默认日志目录存在
- `CleanupLogFiles() error` - 清理默认日志文件（测试用）
- `CleanupLogFile(path string) error` - 清理指定日志文件

## 使用示例

### 基本配置（只输出到stdout）
```go
logger := logger.NewWithOptions(logger.Options{
    Level:      logger.InfoLevel,
    Format:     "console",
    TimeFormat: "2006-01-02 15:04:05",
    Caller:     true,
    Stacktrace: true,
})
```

### 生产环境配置（同时输出到stdout和文件）
```go
logger := logger.NewWithOptions(logger.Options{
    Level:            logger.InfoLevel,
    Format:           "json",
    TimeFormat:       time.RFC3339,
    Caller:           true,
    Stacktrace:       true,
    EnableFileOutput: true, // 启用文件输出
    Rotate: &logger.RotateConfig{
        Filename:   "logs/app.log",
        MaxSize:    100, // 100MB
        MaxBackups: 10,
        MaxAge:     30,  // 30天
        Compress:   true,
        LocalTime:  true,
    },
})
```

### 预设配置
```go
// 开发环境：只输出到stdout，debug级别
logger.SetupDevelopment()

// 生产环境：同时输出到stdout和文件
logger.SetupProduction()
```

## 运行示例

```bash
cd examples/logger-context-demo
go run main.go
```

示例将展示：
1. 基本配置的使用
2. 开发和生产环境的预设配置
3. 自定义配置的各种选项
4. 上下文支持和链式调用
5. 文件输出的配置

运行后，您将看到：
- 控制台输出的日志
- 在`logs/`目录下生成的日志文件（如果启用了文件输出） 