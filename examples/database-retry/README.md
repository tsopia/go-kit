# Database 连接重试机制演示

这个示例演示了 go-kit 数据库包的连接重试机制功能。

## 🎯 功能特性

### 智能重试策略
- **指数退避**: 每次重试的延迟时间呈指数增长
- **最大延迟限制**: 防止重试间隔过长
- **抖动机制**: 避免雷群效应，分散重试时间
- **可配置重试次数**: 支持1-100次重试

### 用户无感知设计
- **默认启用**: 当配置重试次数>1时自动启用
- **合理默认值**: 提供生产环境可用的默认配置
- **透明重试**: 用户无需修改现有代码

## 📋 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `RetryEnabled` | bool | true | 是否启用重试 |
| `RetryMaxAttempts` | int | 3 | 最大重试次数 |
| `RetryInitialDelay` | time.Duration | 1s | 初始重试延迟 |
| `RetryMaxDelay` | time.Duration | 30s | 最大重试延迟 |
| `RetryBackoffFactor` | float64 | 2.0 | 退避因子 |
| `RetryJitterEnabled` | bool | true | 是否启用抖动 |

## 🚀 使用示例

### 1. 默认配置（推荐）
```go
config := &database.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    Username: "root",
    Password: "password",
    Database: "test",
    // 自动使用默认重试配置
}
```

### 2. 自定义重试策略
```go
config := &database.Config{
    Driver:   "mysql",
    Host:     "localhost",
    Port:     3306,
    Username: "root",
    Password: "password",
    Database: "test",
    
    // 自定义重试策略
    RetryEnabled:       true,
    RetryMaxAttempts:   5,
    RetryInitialDelay:  500 * time.Millisecond,
    RetryMaxDelay:      10 * time.Second,
    RetryBackoffFactor: 1.5,
    RetryJitterEnabled: true,
}
```

### 3. 禁用重试
```go
config := &database.Config{
    Driver:   "sqlite",
    Database: ":memory:",
    
    // 禁用重试
    RetryConfigured: true,
    RetryEnabled:    false,
}
```

## 📊 重试延迟计算

重试延迟采用指数退避算法：

```
延迟 = 初始延迟 × 退避因子^(重试次数-1)
```

以默认配置为例：
- 第1次重试: 1s
- 第2次重试: 2s  
- 第3次重试: 4s
- 第4次重试: 8s
- 第5次重试: 16s
- 第6次重试: 30s (达到最大延迟)

## 🔧 运行演示

```bash
# 运行演示程序
go run examples/database-retry/main.go

# 运行相关测试
go test ./database -v -run TestRetry
```

## 🎨 设计理念

1. **用户无感知**: 现有代码无需修改即可享受重试功能
2. **生产就绪**: 提供经过验证的默认配置
3. **高度可配置**: 支持各种复杂的重试场景
4. **性能优化**: 智能抖动机制避免系统过载

## 📝 注意事项

- 重试机制仅在初始连接时生效
- 建议在生产环境中启用重试功能
- 可以通过日志观察重试过程
- 重试次数过多可能延长启动时间

## 🤝 适用场景

- 数据库服务重启
- 网络不稳定
- 容器化环境启动顺序问题
- 云服务临时不可用
- 负载均衡器故障转移 
