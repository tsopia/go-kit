# Trace ID 和 Logger 联动功能演示

本示例演示了 Go-Kit 项目中 **架构重构后的 trace ID 管理**和**与 logger 包的完美联动**。

## 🏗️ 架构改进

### 问题
原本的设计中，trace ID 和 request ID 的常量定义在 `httpserver` 包中，这会导致：
- `logger` 包需要依赖 `httpserver` 包
- 不合理的依赖关系
- 其他包也可能需要这些常量，造成循环依赖

### 解决方案
创建了 **`pkg/constants`** 包专门存放各个模块都会用到的公共常量和方法：

```
pkg/constants/
├── trace.go          # 追踪相关的常量和工具函数
```

### 架构优势

1. **清晰的依赖关系**
   ```
   httpserver → constants ← logger
   ```
   
2. **避免循环依赖**
   - 所有包都可以安全地依赖 `constants` 包
   - `constants` 包不依赖任何业务包

3. **统一的常量管理**
   - 所有追踪相关的常量集中管理
   - 便于维护和扩展

## 🚀 核心功能

### 1. 常量定义
```go
// Context keys
const (
    TraceIDKey   = "trace_id"
    RequestIDKey = "request_id"
)

// HTTP headers
const (
    TraceIDHeader   = "X-Trace-ID"
    RequestIDHeader = "X-Request-ID"
)
```

### 2. 工具函数
```go
// ID 生成
constants.GenerateID()

// Context 操作
constants.WithTraceID(ctx, traceID)
constants.WithRequestID(ctx, requestID)
constants.WithTraceAndRequestID(ctx, traceID, requestID)

// 从 Context 提取
constants.TraceIDFromContext(ctx)
constants.RequestIDFromContext(ctx)
```

### 3. Logger 自动联动
当使用 `logger.FromContext(ctx)` 创建 logger 时，会自动提取 context 中的追踪信息并添加到所有日志中：

```go
ctx = constants.WithTraceAndRequestID(ctx, traceID, requestID)
logger := logger.FromContext(ctx)
logger.Info("用户登录") 
// 输出: {"level":"info","msg":"用户登录","trace_id":"abc123","request_id":"def456"}
```

## 🧪 运行演示

```bash
cd examples/trace-test
go run main.go
```

**预期输出：**
- 基础日志：无追踪信息
- 追踪日志：自动包含 `trace_id` 和 `request_id` 字段
- ID验证：提取的ID与原始ID完全匹配

## 📈 实际应用

### HTTP 中间件中的使用
```go
func TraceIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader(constants.TraceIDHeader)
        if traceID == "" {
            traceID = constants.GenerateID()
        }
        
        // 设置到 context 中
        ctx := constants.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}
```

### 业务代码中的使用
```go
func UserHandler(c *gin.Context) {
    // 从 request context 创建带追踪信息的 logger
    ctx := httpserver.ContextFromGin(c)
    logger := logger.FromContext(ctx)
    
    // 所有日志自动包含 trace_id 和 request_id
    logger.Info("开始处理用户请求")
    logger.Error("用户不存在", "user_id", userID)
}
```

## 🎯 设计原则

1. **单一职责**：`constants` 包只负责常量和基础工具函数
2. **无业务逻辑**：不包含任何业务相关的逻辑
3. **向后兼容**：logger 的 `DefaultContextExtractor` 支持多种格式的 key
4. **类型安全**：所有常量都有明确的类型和文档

这种设计完美解决了**常量共享**和**依赖管理**的问题，为整个项目提供了清晰、可维护的架构基础。 