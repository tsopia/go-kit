# 日志系统 (pkg/logger)

基于Zap的高性能结构化日志系统，支持多种输出格式、文件轮转、上下文追踪等企业级特性。

## 🚀 特性

- ✅ 基于Zap的高性能日志
- ✅ 支持JSON、Console、Text多种输出格式
- ✅ 自动文件轮转和压缩
- ✅ 上下文追踪（trace_id、request_id）
- ✅ 结构化日志和字段支持
- ✅ 采样和限流
- ✅ 钩子函数支持
- ✅ 线程安全

## 📖 快速开始

### 基本使用

```go
package main

import (
    "go-kit/pkg/logger"
)

func main() {
    // 创建默认日志记录器
    log := logger.New()
    
    // 基本日志记录
    log.Info("应用启动", "port", 8080)
    log.Debug("调试信息", "user_id", 123)
    log.Warn("警告信息", "memory_usage", "85%")
    log.Error("错误信息", "error", "连接失败")
    
    // 格式化日志
    log.Infof("用户 %s 登录成功", "张三")
    log.Errorf("处理请求失败: %v", err)
}
```

### 环境配置

```go
// 开发环境
logger.SetupDevelopment()

// 生产环境
logger.SetupProduction()

// 自定义配置
logger.SetupWithOptions(logger.Options{
    Level:            logger.InfoLevel,
    Format:           logger.FormatJSON,
    TimeFormat:       time.RFC3339,
    EnableFileOutput: true,
    Rotate: &logger.RotateConfig{
        Filename:   "logs/app.log",
        MaxSize:    100,    // MB
        MaxBackups: 10,
        MaxAge:     30,     // 天
        Compress:   true,
    },
})
```

## 🔧 API 参考

### 创建日志记录器

#### New
创建默认日志记录器

```go
log := logger.New()
```

#### NewWithOptions
使用自定义选项创建日志记录器

```go
log := logger.NewWithOptions(logger.Options{
    Level:            logger.InfoLevel,
    Format:           logger.FormatJSON,
    TimeFormat:       time.RFC3339,
    Caller:           true,
    Stacktrace:       true,
    EnableFileOutput: true,
    Rotate: &logger.RotateConfig{
        Filename:   "logs/app.log",
        MaxSize:    100,
        MaxBackups: 10,
        MaxAge:     30,
        Compress:   true,
    },
})
```

#### 预定义配置

```go
// 开发环境配置
log := logger.NewDevelopment()

// 生产环境配置
log := logger.NewProduction()

// 空操作日志记录器（用于测试）
log := logger.NewNop()
```

### 日志级别

```go
const (
    DebugLevel logger.Level = iota - 1
    InfoLevel
    WarnLevel
    ErrorLevel
    FatalLevel
)

// 设置日志级别
log.SetLevel(logger.DebugLevel)

// 检查级别是否启用
if log.IsEnabled(logger.DebugLevel) {
    log.Debug("调试信息")
}
```

### 日志格式

```go
const (
    FormatJSON    logger.Format = "json"    // JSON格式
    FormatConsole logger.Format = "console"  // 控制台格式（带颜色）
    FormatText    logger.Format = "text"     // 文本格式（不带颜色）
)
```

### 基本日志方法

```go
// 结构化日志
log.Info("用户登录", "user_id", 123, "ip", "192.168.1.1")
log.Debug("处理请求", "method", "GET", "path", "/api/users")
log.Warn("内存使用率高", "usage", "85%", "threshold", "80%")
log.Error("数据库连接失败", "error", err, "host", "localhost")

// 格式化日志
log.Infof("用户 %s 登录成功", username)
log.Debugf("处理请求 %s %s", method, path)
log.Warnf("内存使用率: %s", usage)
log.Errorf("处理失败: %v", err)

// 致命错误（会调用os.Exit(1)）
log.Fatal("应用启动失败", "error", err)
log.Fatalf("配置错误: %v", err)

// Panic（会panic）
log.Panic("严重错误", "error", err)
log.Panicf("系统错误: %v", err)
```

### 上下文支持

#### 从Context创建日志记录器

```go
import (
    "context"
    "go-kit/pkg/logger"
    "go-kit/pkg/httpserver"
)

func userHandler(c *gin.Context) {
    // 从Gin Context提取request context
    ctx := httpserver.ContextFromGin(c)
    
    // 创建带上下文的日志记录器
    log := logger.FromContext(ctx)
    
    // 所有日志自动包含trace_id和request_id
    log.Info("开始处理用户请求")
    log.Debug("请求详情", "user_id", userID)
    log.Error("用户不存在", "user_id", userID)
}
```

#### 手动设置上下文

```go
// 创建带上下文的日志记录器
ctx := context.WithValue(context.Background(), "user_id", "123")
log := logger.WithContext(ctx)

// 或者手动添加字段
log := logger.With("user_id", "123", "session_id", "abc")
log.Info("用户操作", "action", "login")
```

### 字段操作

```go
// 添加字段
log := logger.With("user_id", 123, "session_id", "abc")

// 添加错误字段
log := logger.WithError(err)

// 添加多个字段
log := logger.WithFields(map[string]interface{}{
    "user_id":    123,
    "session_id": "abc",
    "ip":         "192.168.1.1",
})

// 命名日志记录器
log := logger.Named("user-service")
log.Info("用户服务启动")
```

### 文件轮转

```go
log := logger.NewWithOptions(logger.Options{
    EnableFileOutput: true,
    Rotate: &logger.RotateConfig{
        Filename:   "logs/app.log",  // 日志文件路径
        MaxSize:    100,              // 单个文件最大大小（MB）
        MaxBackups: 10,               // 最大备份文件数
        MaxAge:     30,               // 最大保留天数
        Compress:   true,             // 是否压缩
        LocalTime:  true,             // 使用本地时间
    },
})
```

### 采样配置

```go
log := logger.NewWithOptions(logger.Options{
    Sampling: &logger.SamplingConfig{
        Initial:    100,              // 初始采样数量
        Thereafter: 10,               // 后续采样数量
        Tick:       1 * time.Second, // 采样周期
    },
})
```

## 🏗️ 最佳实践

### 1. 日志级别使用

```go
// Debug - 详细的调试信息
log.Debug("SQL查询", "query", sql, "params", params)

// Info - 重要的业务事件
log.Info("用户注册", "user_id", userID, "email", email)

// Warn - 警告信息，不影响系统运行
log.Warn("数据库连接池使用率高", "usage", "90%")

// Error - 错误信息，需要关注
log.Error("数据库连接失败", "error", err, "host", host)

// Fatal - 致命错误，程序无法继续运行
log.Fatal("配置文件不存在", "file", configFile)
```

### 2. 结构化日志

```go
// ✅ 好的做法 - 使用结构化字段
log.Info("用户登录",
    "user_id", userID,
    "ip", clientIP,
    "user_agent", userAgent,
    "login_method", "password",
)

// ❌ 不好的做法 - 字符串拼接
log.Info(fmt.Sprintf("用户 %d 从 %s 登录", userID, clientIP))
```

### 3. 错误日志

```go
// ✅ 好的做法 - 包含错误和上下文
log.Error("数据库查询失败",
    "error", err,
    "query", sql,
    "params", params,
    "user_id", userID,
)

// ❌ 不好的做法 - 只有错误信息
log.Error("数据库查询失败", "error", err)
```

### 4. 性能敏感场景

```go
// 使用采样减少日志量
log := logger.NewWithOptions(logger.Options{
    Sampling: &logger.SamplingConfig{
        Initial:    100,
        Thereafter: 10,
        Tick:       1 * time.Second,
    },
})

// 检查级别避免不必要的计算
if log.IsEnabled(logger.DebugLevel) {
    expensiveData := calculateExpensiveData()
    log.Debug("调试信息", "data", expensiveData)
}
```

### 5. 上下文追踪

```go
// 在HTTP处理器中
func userHandler(c *gin.Context) {
    ctx := httpserver.ContextFromGin(c)
    log := logger.FromContext(ctx)
    
    // 所有日志自动包含trace_id和request_id
    log.Info("开始处理用户请求")
    
    user, err := getUser(userID)
    if err != nil {
        log.Error("获取用户失败", "error", err, "user_id", userID)
        return
    }
    
    log.Info("用户获取成功", "user_id", userID, "user_name", user.Name)
}

// 在后台任务中
func backgroundTask(ctx context.Context) {
    log := logger.FromContext(ctx)
    
    log.Info("开始后台任务")
    
    // 任务处理...
    
    log.Info("后台任务完成")
}
```

### 6. 日志配置

#### 开发环境

```go
func setupDevelopmentLogger() {
    logger.SetupWithOptions(logger.Options{
        Level:      logger.DebugLevel,
        Format:     logger.FormatConsole,
        TimeFormat: "2006-01-02 15:04:05",
        Caller:     true,
        Stacktrace: true,
    })
}
```

#### 生产环境

```go
func setupProductionLogger() {
    logger.SetupWithOptions(logger.Options{
        Level:            logger.InfoLevel,
        Format:           logger.FormatJSON,
        TimeFormat:       time.RFC3339,
        Caller:           false,
        Stacktrace:       false,
        EnableFileOutput: true,
        Rotate: &logger.RotateConfig{
            Filename:   "logs/app.log",
            MaxSize:    100,
            MaxBackups: 10,
            MaxAge:     30,
            Compress:   true,
            LocalTime:  true,
        },
    })
}
```

### 7. 自定义钩子

```go
// 创建自定义钩子
func metricsHook(entry zapcore.Entry) error {
    // 记录指标
    metrics.IncCounter("log_entries_total", map[string]string{
        "level": entry.Level.String(),
    })
    return nil
}

// 使用钩子
log := logger.NewWithOptions(logger.Options{
    Hooks: []logger.Hook{metricsHook},
})
```

## 🧪 测试

### 单元测试

```go
func TestLogger(t *testing.T) {
    // 创建测试日志记录器
    log := logger.NewNop()
    
    // 测试基本日志方法
    log.Info("测试信息", "key", "value")
    log.Error("测试错误", "error", "test error")
    
    // 测试上下文
    ctx := context.WithValue(context.Background(), "test_key", "test_value")
    logWithCtx := logger.WithContext(ctx)
    logWithCtx.Info("带上下文的日志")
}
```

### 集成测试

```go
func TestLoggerWithFile(t *testing.T) {
    // 创建临时日志文件
    tempDir := t.TempDir()
    logFile := filepath.Join(tempDir, "test.log")
    
    log := logger.NewWithOptions(logger.Options{
        EnableFileOutput: true,
        Rotate: &logger.RotateConfig{
            Filename: logFile,
        },
    })
    
    // 写入日志
    log.Info("测试日志", "test", true)
    
    // 同步日志
    log.Sync()
    
    // 检查日志文件
    content, err := os.ReadFile(logFile)
    if err != nil {
        t.Fatalf("读取日志文件失败: %v", err)
    }
    
    if !strings.Contains(string(content), "测试日志") {
        t.Error("日志文件中未找到预期内容")
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 日志文件权限问题

```bash
# 确保日志目录存在且有写权限
mkdir -p logs
chmod 755 logs

# 检查文件权限
ls -la logs/
```

#### 2. 日志文件过大

```go
// 调整轮转配置
log := logger.NewWithOptions(logger.Options{
    EnableFileOutput: true,
    Rotate: &logger.RotateConfig{
        Filename:   "logs/app.log",
        MaxSize:    50,     // 减小文件大小
        MaxBackups: 5,      // 减少备份数量
        MaxAge:     7,      // 减少保留天数
        Compress:   true,   // 启用压缩
    },
})
```

#### 3. 性能问题

```go
// 使用采样减少日志量
log := logger.NewWithOptions(logger.Options{
    Sampling: &logger.SamplingConfig{
        Initial:    100,
        Thereafter: 10,
        Tick:       1 * time.Second,
    },
})

// 检查日志级别避免不必要的计算
if log.IsEnabled(logger.DebugLevel) {
    expensiveData := calculateExpensiveData()
    log.Debug("调试信息", "data", expensiveData)
}
```

### 调试技巧

```go
// 1. 检查日志级别
level := log.GetLevel()
fmt.Printf("当前日志级别: %s\n", level)

// 2. 检查是否启用某个级别
if log.IsEnabled(logger.DebugLevel) {
    fmt.Println("Debug级别已启用")
}

// 3. 同步日志缓冲区
log.Sync()

// 4. 获取底层Zap记录器
zapLogger := log.GetZap()
sugarLogger := log.GetSugar()
```

## 📚 相关链接

- [Zap官方文档](https://github.com/uber-go/zap)
- [示例项目](./examples/logger-context-demo/)
- [返回首页](../README.md) 