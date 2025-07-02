# 错误处理 (pkg/errors)

统一的错误码系统和错误包装，提供类型安全的错误处理机制。

## 🚀 特性

- ✅ 统一的错误码系统
- ✅ 支持错误包装和上下文
- ✅ 类型安全的错误检查
- ✅ 结构化错误信息
- ✅ 堆栈跟踪支持
- ✅ JSON序列化支持

## 📖 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "go-kit/pkg/errors"
)

func main() {
    // 创建错误
    err := errors.New(errors.CodeInvalidParam, "参数无效")
    
    // 包装现有错误
    dbErr := fmt.Errorf("数据库连接失败")
    wrappedErr := errors.Wrap(dbErr, errors.CodeDatabaseError, "用户查询失败")
    
    // 检查错误类型
    if errors.IsInvalidParam(err) {
        fmt.Println("参数错误")
    }
    
    if errors.IsDatabaseError(wrappedErr) {
        fmt.Println("数据库错误")
    }
}
```

### 错误码系统

```go
// 系统级错误码 (1000-1999)
errors.CodeInternalServer    // 内部服务器错误
errors.CodeInvalidParam     // 参数无效
errors.CodeNotFound         // 资源不存在
errors.CodeUnauthorized     // 未授权
errors.CodeForbidden        // 访问被禁止
errors.CodeConflict         // 资源冲突
errors.CodeTooManyRequests  // 请求过多

// 业务级错误码 (2000-2999)
errors.CodeUserNotFound     // 用户不存在
errors.CodeUserExists       // 用户已存在
errors.CodeInvalidPassword  // 密码无效
errors.CodeTokenExpired     // 令牌已过期
errors.CodeTokenInvalid     // 令牌无效

// 数据库错误码 (3000-3999)
errors.CodeDatabaseError    // 数据库错误
errors.CodeRecordNotFound   // 记录不存在
errors.CodeDuplicateKey     // 数据重复
errors.CodeForeignKeyViolation // 外键约束违反

// 外部服务错误码 (4000-4999)
errors.CodeExternalServiceError // 外部服务错误
errors.CodeNetworkError     // 网络错误
errors.CodeTimeoutError     // 请求超时
```

## 🔧 API 参考

### 创建错误

#### New
创建新的错误

```go
// 基本错误
err := errors.New(errors.CodeInvalidParam, "参数无效")

// 带详细信息的错误
err := errors.NewWithDetails(errors.CodeDatabaseError, "数据库操作失败", "连接超时")
```

#### Wrap
包装现有错误

```go
// 包装错误
originalErr := fmt.Errorf("原始错误")
wrappedErr := errors.Wrap(originalErr, errors.CodeDatabaseError, "数据库操作失败")

// 包装并添加详细信息
wrappedErr := errors.WrapWithDetails(originalErr, errors.CodeDatabaseError, 
    "数据库操作失败", "连接超时")
```

#### 格式化错误

```go
// 格式化错误
err := errors.Newf(errors.CodeInvalidParam, "用户 %s 不存在", username)

// 格式化包装错误
wrappedErr := errors.Wrapf(originalErr, errors.CodeDatabaseError, 
    "查询用户 %s 失败", userID)
```

### 错误检查

#### 基本检查函数

```go
// 检查错误类型
if errors.IsInternalServer(err) {
    // 处理内部服务器错误
}

if errors.IsInvalidParam(err) {
    // 处理参数错误
}

if errors.IsNotFound(err) {
    // 处理未找到错误
}

if errors.IsUnauthorized(err) {
    // 处理未授权错误
}

if errors.IsForbidden(err) {
    // 处理禁止访问错误
}

if errors.IsConflict(err) {
    // 处理冲突错误
}

if errors.IsTooManyRequests(err) {
    // 处理请求过多错误
}
```

#### 业务错误检查

```go
// 用户相关错误
if errors.IsUserNotFound(err) {
    // 处理用户不存在
}

if errors.IsUserExists(err) {
    // 处理用户已存在
}

if errors.IsInvalidPassword(err) {
    // 处理密码无效
}

if errors.IsTokenExpired(err) {
    // 处理令牌过期
}

if errors.IsTokenInvalid(err) {
    // 处理令牌无效
}
```

#### 数据库错误检查

```go
// 数据库错误
if errors.IsDatabaseError(err) {
    // 处理数据库错误
}

if errors.IsRecordNotFound(err) {
    // 处理记录不存在
}

if errors.IsDuplicateKey(err) {
    // 处理数据重复
}

if errors.IsForeignKeyViolation(err) {
    // 处理外键约束违反
}
```

#### 外部服务错误检查

```go
// 外部服务错误
if errors.IsExternalServiceError(err) {
    // 处理外部服务错误
}

if errors.IsNetworkError(err) {
    // 处理网络错误
}

if errors.IsTimeoutError(err) {
    // 处理超时错误
}
```

### 错误信息获取

#### 获取错误码

```go
code := errors.GetCode(err)
fmt.Printf("错误码: %d\n", code.Code)
fmt.Printf("错误名称: %s\n", code.Name)
```

#### 获取错误消息

```go
// 获取错误消息
message := err.(*errors.Error).GetMessage()

// 获取错误详情
details := err.(*errors.Error).Details

// 获取错误上下文
context := errors.GetContext(err)
```

#### 错误解包

```go
// 解包错误
originalErr := errors.Unwrap(err)

// 检查错误类型
if errors.Is(err, someError) {
    // 处理特定错误
}
```

### 错误上下文

#### 添加上下文信息

```go
err := errors.New(errors.CodeDatabaseError, "数据库操作失败").
    WithContext("user_id", userID).
    WithContext("operation", "create_user").
    WithContext("table", "users")
```

#### 添加详细信息

```go
err := errors.New(errors.CodeDatabaseError, "数据库操作失败").
    WithDetails("连接超时，重试3次后仍然失败")
```

#### 设置自定义消息

```go
err := errors.New(errors.CodeDatabaseError, "数据库操作失败").
    WithMessage("创建用户失败")
```

### 堆栈跟踪

```go
// 添加堆栈跟踪
err := errors.New(errors.CodeInternalServer, "内部错误").WithStack()

// 获取堆栈信息
stack := errors.GetStack(err)
if stack != "" {
    fmt.Printf("堆栈跟踪:\n%s\n", stack)
}
```

## 🏗️ 最佳实践

### 1. 错误定义

#### 定义自定义错误码

```go
// 定义业务错误码
var (
    CodeOrderNotFound = errors.NewErrorCode(5000, "ORDER_NOT_FOUND", "订单不存在")
    CodeOrderExpired  = errors.NewErrorCode(5001, "ORDER_EXPIRED", "订单已过期")
    CodePaymentFailed = errors.NewErrorCode(5002, "PAYMENT_FAILED", "支付失败")
)

// 使用自定义错误码
func getOrder(orderID string) (*Order, error) {
    order, err := db.GetOrder(orderID)
    if err != nil {
        return nil, errors.Wrap(err, CodeOrderNotFound, "获取订单失败")
    }
    
    if order.IsExpired() {
        return nil, errors.New(CodeOrderExpired, "订单已过期")
    }
    
    return order, nil
}
```

### 2. 错误处理

#### HTTP处理器中的错误处理

```go
func userHandler(c *gin.Context) {
    userID := c.Param("id")
    
    user, err := getUser(userID)
    if err != nil {
        // 根据错误类型返回不同的HTTP状态码
        switch {
        case errors.IsUserNotFound(err):
            c.JSON(http.StatusNotFound, gin.H{
                "error": "用户不存在",
                "code":  errors.GetCode(err).Code,
            })
            return
            
        case errors.IsUnauthorized(err):
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "未授权访问",
                "code":  errors.GetCode(err).Code,
            })
            return
            
        case errors.IsDatabaseError(err):
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "服务器内部错误",
                "code":  errors.GetCode(err).Code,
            })
            return
            
        default:
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "未知错误",
                "code":  errors.GetCode(err).Code,
            })
            return
        }
    }
    
    c.JSON(http.StatusOK, user)
}
```

#### 服务层错误处理

```go
func (s *UserService) CreateUser(user *User) error {
    // 验证用户数据
    if err := s.validateUser(user); err != nil {
        return errors.Wrap(err, errors.CodeInvalidParam, "用户数据验证失败")
    }
    
    // 检查用户是否已存在
    exists, err := s.userRepo.ExistsByEmail(user.Email)
    if err != nil {
        return errors.Wrap(err, errors.CodeDatabaseError, "检查用户是否存在失败")
    }
    
    if exists {
        return errors.New(errors.CodeUserExists, "用户已存在")
    }
    
    // 创建用户
    if err := s.userRepo.Create(user); err != nil {
        return errors.Wrap(err, errors.CodeDatabaseError, "创建用户失败").
            WithContext("email", user.Email)
    }
    
    return nil
}
```

### 3. 错误日志

#### 结构化错误日志

```go
func logError(err error, logger *logger.Logger) {
    code := errors.GetCode(err)
    context := errors.GetContext(err)
    stack := errors.GetStack(err)
    
    logger.Error("操作失败",
        "error_code", code.Code,
        "error_name", code.Name,
        "error_message", err.Error(),
        "context", context,
        "stack", stack,
    )
}
```

#### 错误分类日志

```go
func logErrorByType(err error, logger *logger.Logger) {
    switch {
    case errors.IsDatabaseError(err):
        logger.Error("数据库错误", "error", err)
        
    case errors.IsNetworkError(err):
        logger.Error("网络错误", "error", err)
        
    case errors.IsInvalidParam(err):
        logger.Warn("参数错误", "error", err)
        
    case errors.IsNotFound(err):
        logger.Info("资源不存在", "error", err)
        
    default:
        logger.Error("未知错误", "error", err)
    }
}
```

### 4. 错误恢复

#### 错误恢复机制

```go
func withRecovery(fn func() error) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("程序panic: %v", r)
        }
    }()
    
    return fn()
}

func safeOperation() error {
    return withRecovery(func() error {
        // 可能panic的操作
        return nil
    })
}
```

#### 重试机制

```go
func withRetry(operation func() error, maxRetries int) error {
    var lastErr error
    
    for i := 0; i <= maxRetries; i++ {
        if err := operation(); err != nil {
            lastErr = err
            
            // 检查是否应该重试
            if !shouldRetry(err) {
                return err
            }
            
            if i < maxRetries {
                time.Sleep(time.Duration(i+1) * time.Second)
                continue
            }
        }
        
        return nil
    }
    
    return errors.Wrap(lastErr, errors.CodeTimeoutError, "操作重试失败")
}

func shouldRetry(err error) bool {
    return errors.IsNetworkError(err) || 
           errors.IsTimeoutError(err) ||
           errors.IsExternalServiceError(err)
}
```

### 5. 错误监控

#### 错误指标收集

```go
type ErrorMetrics struct {
    errorCounter *prometheus.CounterVec
}

func (m *ErrorMetrics) RecordError(err error) {
    code := errors.GetCode(err)
    context := errors.GetContext(err)
    
    labels := map[string]string{
        "error_code": fmt.Sprintf("%d", code.Code),
        "error_name": code.Name,
    }
    
    // 添加上下文标签
    for key, value := range context {
        if str, ok := value.(string); ok {
            labels[key] = str
        }
    }
    
    m.errorCounter.With(labels).Inc()
}

// 使用错误指标
func (s *UserService) CreateUser(user *User) error {
    err := s.doCreateUser(user)
    if err != nil {
        s.errorMetrics.RecordError(err)
    }
    return err
}
```

### 6. 测试中的错误处理

#### 错误测试

```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        user    *User
        wantErr bool
        errCode errors.ErrorCode
    }{
        {
            name: "成功创建用户",
            user: &User{
                Name:     "张三",
                Email:    "zhangsan@example.com",
                Password: "password",
            },
            wantErr: false,
        },
        {
            name: "邮箱已存在",
            user: &User{
                Name:     "李四",
                Email:    "existing@example.com",
                Password: "password",
            },
            wantErr: true,
            errCode: errors.CodeUserExists,
        },
        {
            name: "邮箱格式无效",
            user: &User{
                Name:     "王五",
                Email:    "invalid-email",
                Password: "password",
            },
            wantErr: true,
            errCode: errors.CodeInvalidParam,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.CreateUser(tt.user)
            
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errCode.Code != 0 {
                    assert.True(t, errors.Is(err, tt.errCode))
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## 🧪 测试

### 单元测试

```go
func TestErrorCreation(t *testing.T) {
    // 测试基本错误创建
    err := errors.New(errors.CodeInvalidParam, "参数无效")
    
    if err == nil {
        t.Error("期望创建错误，但得到nil")
    }
    
    code := errors.GetCode(err)
    if code.Code != errors.CodeInvalidParam.Code {
        t.Errorf("期望错误码 %d，实际 %d", errors.CodeInvalidParam.Code, code.Code)
    }
}

func TestErrorWrapping(t *testing.T) {
    originalErr := fmt.Errorf("原始错误")
    wrappedErr := errors.Wrap(originalErr, errors.CodeDatabaseError, "数据库操作失败")
    
    // 检查包装的错误
    if !errors.IsDatabaseError(wrappedErr) {
        t.Error("期望是数据库错误")
    }
    
    // 检查原始错误
    unwrapped := errors.Unwrap(wrappedErr)
    if unwrapped != originalErr {
        t.Error("解包错误不匹配")
    }
}
```

### 集成测试

```go
func TestErrorInHTTPHandler(t *testing.T) {
    // 创建测试服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 模拟不同的错误情况
        switch r.URL.Query().Get("error") {
        case "not_found":
            err := errors.New(errors.CodeNotFound, "资源不存在")
            w.WriteHeader(http.StatusNotFound)
            w.Write([]byte(err.Error()))
            
        case "unauthorized":
            err := errors.New(errors.CodeUnauthorized, "未授权")
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(err.Error()))
            
        default:
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("success"))
        }
    }))
    defer server.Close()
    
    // 测试不同错误情况
    tests := []struct {
        name           string
        errorParam     string
        expectedStatus int
        expectedError  bool
    }{
        {"成功请求", "", 200, false},
        {"资源不存在", "not_found", 404, true},
        {"未授权", "unauthorized", 401, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            url := server.URL
            if tt.errorParam != "" {
                url += "?error=" + tt.errorParam
            }
            
            resp, err := http.Get(url)
            if err != nil {
                t.Fatalf("请求失败: %v", err)
            }
            
            if resp.StatusCode != tt.expectedStatus {
                t.Errorf("期望状态码 %d，实际 %d", tt.expectedStatus, resp.StatusCode)
            }
        })
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 错误类型检查失败

```go
// ❌ 错误的检查方式
if err == errors.CodeInvalidParam {
    // 这样比较是错误的
}

// ✅ 正确的检查方式
if errors.IsInvalidParam(err) {
    // 使用提供的检查函数
}

// 或者使用通用检查
if errors.Is(err, errors.CodeInvalidParam) {
    // 使用通用检查函数
}
```

#### 2. 错误包装丢失

```go
// ❌ 错误的包装方式
err := fmt.Errorf("包装错误: %w", originalErr)

// ✅ 正确的包装方式
err := errors.Wrap(originalErr, errors.CodeDatabaseError, "数据库操作失败")
```

#### 3. 错误上下文丢失

```go
// ❌ 错误的方式
err := errors.New(errors.CodeDatabaseError, "数据库错误")
err.Context["user_id"] = userID // 这样不会生效

// ✅ 正确的方式
err := errors.New(errors.CodeDatabaseError, "数据库错误").
    WithContext("user_id", userID)
```

### 调试技巧

```go
// 1. 打印错误详细信息
func debugError(err error) {
    fmt.Printf("错误: %v\n", err)
    
    code := errors.GetCode(err)
    fmt.Printf("错误码: %d\n", code.Code)
    fmt.Printf("错误名称: %s\n", code.Name)
    
    context := errors.GetContext(err)
    if len(context) > 0 {
        fmt.Printf("错误上下文: %v\n", context)
    }
    
    stack := errors.GetStack(err)
    if stack != "" {
        fmt.Printf("堆栈跟踪:\n%s\n", stack)
    }
}

// 2. 错误链追踪
func traceError(err error) {
    for err != nil {
        fmt.Printf("错误: %v\n", err)
        err = errors.Unwrap(err)
    }
}
```

## 📚 相关链接

- [示例项目](./examples/errors-demo/)
- [返回首页](../README.md) 