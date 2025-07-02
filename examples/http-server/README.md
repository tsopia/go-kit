# HTTP 服务器最小化封装示例

本示例展示了**真正最小化封装**的 HTTP 服务器设计，彻底避免了过度封装问题。

## 🎯 设计原则

**"零强制，全控制"** - 用户对每个中间件、配置都有完全控制权。

## 📊 简化对比

### ❌ 原版本（过度封装）
```go
server := httpserver.NewServer()
server.SetPort(8080)
server.SetHost("0.0.0.0")
server.SetMode("debug")
server.AddMiddleware(middleware)
server.GET("/users", handler)  // 无意义包装
```

### ❌ 中间版本（仍有封装）
```go
server := httpserver.NewServer()
server.Use(middleware)         // 重复的便利方法
// 强制添加 gin.Recovery()     // 用户无法控制
```

### ✅ 当前版本（最小化封装）
```go
server := httpserver.NewServer(nil)
engine := server.Engine()

// 用户完全控制，想要什么中间件自己加
engine.Use(gin.Logger())       // 可选
engine.Use(gin.Recovery())     // 可选
engine.Use(TraceIDMiddleware()) // 可选
engine.GET("/users", handler)  // 直接使用 Gin
```

## 🔧 核心特性

### 1. **零强制策略**
- ❌ 不强制添加任何中间件（连 `gin.Recovery()` 都不强制）
- ❌ 不强制设置 Gin 模式
- ❌ 不提供重复的便利方法
- ✅ 用户想要什么就加什么

### 2. **纯净的 Gin 引擎**
```go
server := httpserver.NewServer(nil)
engine := server.Engine() // 获得纯净的 gin.New()

// 从零开始构建你的中间件栈
if needsLogging {
    engine.Use(gin.Logger())
}
if needsRecovery {
    engine.Use(gin.Recovery())
}
```

### 3. **结构化配置**
```go
config := &httpserver.Config{
    Host:            "127.0.0.1",
    Port:            8080,
    ReadTimeout:     10 * time.Second,
    WriteTimeout:    10 * time.Second,
    IdleTimeout:     60 * time.Second,
    MaxHeaderBytes:  1 << 20,
    ShutdownTimeout: 10 * time.Second,
}
```

### 4. **可选中间件库**
提供常用中间件函数，但**完全可选**：
- `TraceIDMiddleware()` - 请求追踪
- `RequestIDMiddleware()` - 请求唯一标识
- `CORSMiddleware()` - 跨域支持

## 🚀 使用示例

### 完整功能服务器
```go
server := httpserver.NewServer(nil)
engine := server.Engine()

// 手动添加需要的中间件
engine.Use(gin.Logger())
engine.Use(gin.Recovery())
engine.Use(httpserver.TraceIDMiddleware())
engine.Use(httpserver.CORSMiddleware())

engine.GET("/api/users", handler)
server.Start()
```

### 极简服务器
```go
server := httpserver.NewServer(&httpserver.Config{Port: 8080})
engine := server.Engine()

// 不添加任何中间件，最小开销
engine.GET("/ping", func(c *gin.Context) {
    c.JSON(200, gin.H{"message": "pong"})
})
server.Start()
```

### 自定义中间件服务器
```go
server := httpserver.NewServer(nil)
engine := server.Engine()

// 用户自己控制 Gin 模式
gin.SetMode(gin.ReleaseMode)

// 只添加必要的中间件
engine.Use(httpserver.TraceIDMiddleware())
engine.Use(customAuthMiddleware())
```

## 📈 架构优势

### 1. **真正的最小化封装**
- 只封装**服务器生命周期管理**（启动、关闭）
- 只封装**HTTP 服务器配置**（超时、端口等）
- **不封装任何 Gin 功能**

### 2. **完全的用户控制**
- 中间件：用户决定加哪些，顺序如何
- 配置：用户决定每个参数的值
- 模式：用户自己控制 `gin.SetMode()`

### 3. **零学习成本**
- 直接使用 Gin 的原生 API
- 不需要学习额外的封装方法
- 完全的 Gin 生态系统兼容性

### 4. **性能最优**
- 无中间层开销
- 用户可以构建最优的中间件栈
- 支持极简部署（0 中间件）

## 🧪 运行测试

```bash
cd examples/http-server
go run main.go
```

**可以看到三种不同的部署模式：**

1. **默认服务器** (localhost:8080) - 6 个处理器
   - gin.Logger + gin.Recovery + TraceID + RequestID + CORS + 业务处理器

2. **自定义服务器** (localhost:9000) - 2 个处理器  
   - TraceID + 业务处理器

3. **极简服务器** (localhost:9001) - 1 个处理器
   - 只有业务处理器，无任何中间件

## 📋 测试端点

```bash
# 测试完整中间件服务器
curl http://localhost:8080/health

# 测试极简服务器（无中间件）
curl http://localhost:9001/minimal

# 测试 Trace ID 传递
curl -H "X-Trace-ID: custom-trace-123" http://localhost:8080/trace
```

## 🎖️ 设计哲学

这个设计遵循了几个重要原则：

1. **"不做假设"** - 不假设用户需要什么中间件
2. **"不限制选择"** - 用户可以使用任何 Gin 功能
3. **"不增加复杂性"** - 封装的代码比原生代码更简单
4. **"不重复造轮子"** - 直接暴露成熟的 Gin API

### Server 只做两件事：
1. **管理 HTTP 服务器生命周期**（启动、关闭、配置）
2. **提供可选的中间件函数**（不强制使用）

### 用户完全控制：
- 中间件的选择和顺序
- Gin 的所有功能和配置
- 性能优化策略

这就是**真正的最小化封装** - 只封装有价值的抽象，暴露所有必要的控制权。 