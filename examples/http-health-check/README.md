# HTTP Server 健康检查功能演示

这个示例展示了 Go-Kit HTTP Server 的健康检查功能，包括基本健康检查、带管理器的健康检查、自定义路径等。

## 🚀 功能特性

- ✅ 默认健康检查接口
- ✅ 健康检查管理器
- ✅ 内置健康检查器（数据库、HTTP、自定义）
- ✅ 自定义健康检查路径
- ✅ 日志集成
- ✅ 上下文追踪

## 📖 运行示例

```bash
# 进入示例目录
cd examples/http-health-check

# 运行示例
go run main.go
```

## 🔧 示例说明

### 1. 基本健康检查

- **服务器**: http://localhost:8080
- **健康检查**: http://localhost:8080/health
- **特性**: 默认启用，返回简单的健康状态

```bash
curl http://localhost:8080/health
```

**响应示例:**
```json
{
    "status": "healthy",
    "timestamp": 1640995200,
    "version": "1.0.0"
}
```

### 2. 带管理器的健康检查

- **服务器**: http://localhost:8081
- **健康检查**: http://localhost:8081/health
- **特性**: 包含多个检查器，显示详细状态

```bash
curl http://localhost:8081/health
```

**响应示例:**
```json
{
    "status": "unhealthy",
    "timestamp": 1640995200,
    "version": "1.0.0",
    "uptime": 3600,
    "checks": {
        "database": {
            "status": "healthy"
        },
        "redis": {
            "status": "healthy"
        },
        "basic_service": {
            "status": "healthy"
        },
        "external_api": {
            "status": "unhealthy",
            "error": "外部API服务不可用"
        }
    },
    "error": "service unavailable"
}
```

### 3. 自定义健康检查路径

- **服务器**: http://localhost:8082
- **健康检查**: http://localhost:8082/api/v1/health
- **特性**: 自定义健康检查路径

```bash
curl http://localhost:8082/api/v1/health
```

### 4. 禁用健康检查

- **服务器**: http://localhost:8083
- **健康检查**: http://localhost:8083/health (返回404)
- **特性**: 完全禁用健康检查功能

```bash
curl http://localhost:8083/health
# 返回 404 Not Found
```

### 5. 带日志集成

- **服务器**: http://localhost:8084
- **健康检查**: http://localhost:8084/health
- **特性**: 集成日志和上下文追踪

```bash
curl http://localhost:8084/
```

## 🏗️ 代码示例

### 基本使用

```go
// 创建服务器（默认启用健康检查）
server := httpserver.NewServer(nil)

// 添加业务路由
server.GET("/", handler)

// 启动服务器
server.Run()
```

### 带管理器的健康检查

```go
// 创建健康检查管理器
manager := httpserver.NewHealthCheckManager("1.0.0")

// 添加检查器
manager.AddChecker(httpserver.NewDatabaseHealthChecker("mysql", db))
manager.AddChecker(httpserver.NewHTTPHealthChecker("api", "http://api.example.com/health", 5*time.Second))
manager.AddChecker(httpserver.NewCustomHealthChecker("custom", func(ctx context.Context) error {
    // 自定义检查逻辑
    return nil
}))

// 创建服务器并启用管理器
server := httpserver.NewServer(&httpserver.Config{
    EnableHealthCheck: true,
    HealthCheckPath: "/health",
})
server.EnableHealthCheckWithManager(manager)
```

### 自定义配置

```go
server := httpserver.NewServer(&httpserver.Config{
    Host:            "0.0.0.0",
    Port:            8080,
    EnableHealthCheck: true,
    HealthCheckPath: "/api/v1/health", // 自定义路径
    HealthCheckPort: 0,                // 使用主端口
})
```

## 🔍 健康检查器类型

### 数据库健康检查器

```go
// 适用于任何实现了Ping()方法的数据库连接
manager.AddChecker(httpserver.NewDatabaseHealthChecker("mysql", mysqlDB))
manager.AddChecker(httpserver.NewDatabaseHealthChecker("redis", redisDB))
```

### HTTP服务健康检查器

```go
// 检查外部HTTP服务的健康状态
manager.AddChecker(httpserver.NewHTTPHealthChecker(
    "payment_service", 
    "http://payment.example.com/health", 
    5*time.Second,
))
```

### 自定义健康检查器

```go
// 实现自定义检查逻辑
manager.AddChecker(httpserver.NewCustomHealthChecker("file_system", func(ctx context.Context) error {
    // 检查磁盘空间
    if diskUsage > 90 {
        return fmt.Errorf("磁盘使用率过高: %.1f%%", diskUsage)
    }
    return nil
}))
```

## 📊 健康检查响应格式

### 健康状态

```json
{
    "status": "healthy",
    "timestamp": 1640995200,
    "version": "1.0.0",
    "uptime": 3600,
    "checks": {
        "database": {
            "status": "healthy"
        },
        "redis": {
            "status": "healthy"
        }
    }
}
```

### 不健康状态

```json
{
    "status": "unhealthy",
    "timestamp": 1640995200,
    "version": "1.0.0",
    "uptime": 3600,
    "checks": {
        "database": {
            "status": "unhealthy",
            "error": "connection timeout"
        }
    },
    "error": "service unavailable"
}
```

## 🧪 测试

运行示例后，可以使用以下命令测试各个端点：

```bash
# 测试基本健康检查
curl -s http://localhost:8080/health | jq

# 测试带管理器的健康检查
curl -s http://localhost:8081/health | jq

# 测试自定义路径健康检查
curl -s http://localhost:8082/api/v1/health | jq

# 测试禁用健康检查（应该返回404）
curl -s http://localhost:8083/health

# 测试带日志的请求
curl -s http://localhost:8084/ | jq
```

## 📚 相关文档

- [HTTP Server 文档](../../docs/httpserver.md)
- [Logger 文档](../../docs/logger.md)
- [返回首页](../../README.md)