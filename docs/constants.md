# 常量定义 (constants)

共享常量和工具函数，提供项目中使用的通用常量和辅助函数。

## 🚀 特性

- ✅ 统一的常量定义
- ✅ 类型安全的常量
- ✅ 避免循环依赖
- ✅ 便于维护和扩展
- ✅ 支持国际化

## 📖 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/tsopia/go-kit/constants"
)

func main() {
    // 使用HTTP状态码常量
    fmt.Printf("OK: %d\n", constants.HTTPStatusOK)
    fmt.Printf("Not Found: %d\n", constants.HTTPStatusNotFound)
    
    // 使用HTTP方法常量
    fmt.Printf("GET: %s\n", constants.HTTPMethodGET)
    fmt.Printf("POST: %s\n", constants.HTTPMethodPOST)
    
    // 使用时间格式常量
    fmt.Printf("ISO格式: %s\n", constants.TimeFormatISO)
    fmt.Printf("RFC3339格式: %s\n", constants.TimeFormatRFC3339)
}
```

### 常量分类

```go
// HTTP相关常量
constants.HTTPStatusOK           // 200
constants.HTTPStatusCreated      // 201
constants.HTTPStatusBadRequest   // 400
constants.HTTPStatusUnauthorized // 401
constants.HTTPStatusNotFound     // 404
constants.HTTPStatusInternalServerError // 500

// HTTP方法常量
constants.HTTPMethodGET    // "GET"
constants.HTTPMethodPOST   // "POST"
constants.HTTPMethodPUT    // "PUT"
constants.HTTPMethodDELETE // "DELETE"
constants.HTTPMethodPATCH  // "PATCH"

// 时间格式常量
constants.TimeFormatISO     // "2006-01-02T15:04:05Z07:00"
constants.TimeFormatRFC3339 // "2006-01-02T15:04:05Z07:00"
constants.TimeFormatDate    // "2006-01-02"
constants.TimeFormatTime    // "15:04:05"

// 日志级别常量
constants.LogLevelDebug // "debug"
constants.LogLevelInfo  // "info"
constants.LogLevelWarn  // "warn"
constants.LogLevelError // "error"
constants.LogLevelFatal // "fatal"

// 环境常量
constants.EnvDevelopment // "development"
constants.EnvProduction  // "production"
constants.EnvTesting     // "testing"

// 数据库驱动常量
constants.DBDriverMySQL    // "mysql"
constants.DBDriverPostgres // "postgres"
constants.DBDriverSQLite   // "sqlite"

// 配置格式常量
constants.ConfigFormatYAML // "yaml"
constants.ConfigFormatJSON // "json"
constants.ConfigFormatTOML // "toml"
```

## 🔧 API 参考

### HTTP常量

#### HTTP状态码

```go
// 2xx 成功
HTTPStatusOK                  // 200
HTTPStatusCreated             // 201
HTTPStatusAccepted            // 202
HTTPStatusNoContent           // 204

// 3xx 重定向
HTTPStatusMovedPermanently   // 301
HTTPStatusFound              // 302
HTTPStatusNotModified        // 304

// 4xx 客户端错误
HTTPStatusBadRequest         // 400
HTTPStatusUnauthorized       // 401
HTTPStatusForbidden          // 403
HTTPStatusNotFound           // 404
HTTPStatusMethodNotAllowed   // 405
HTTPStatusConflict           // 409
HTTPStatusTooManyRequests    // 429

// 5xx 服务器错误
HTTPStatusInternalServerError // 500
HTTPStatusNotImplemented     // 501
HTTPStatusBadGateway         // 502
HTTPStatusServiceUnavailable // 503
```

#### HTTP方法

```go
HTTPMethodGET    // "GET"
HTTPMethodPOST   // "POST"
HTTPMethodPUT    // "PUT"
HTTPMethodDELETE // "DELETE"
HTTPMethodPATCH  // "PATCH"
HTTPMethodHEAD   // "HEAD"
HTTPMethodOPTIONS // "OPTIONS"
```

#### HTTP头部

```go
HTTPHeaderContentType     // "Content-Type"
HTTPHeaderAuthorization   // "Authorization"
HTTPHeaderAccept         // "Accept"
HTTPHeaderUserAgent      // "User-Agent"
HTTPHeaderXRequestID     // "X-Request-ID"
HTTPHeaderXForwardedFor  // "X-Forwarded-For"
```

### 时间常量

#### 时间格式

```go
TimeFormatISO     // "2006-01-02T15:04:05Z07:00"
TimeFormatRFC3339 // "2006-01-02T15:04:05Z07:00"
TimeFormatDate    // "2006-01-02"
TimeFormatTime    // "15:04:05"
TimeFormatDateTime // "2006-01-02 15:04:05"
TimeFormatUnix    // "1136214245"
```

#### 时间间隔

```go
TimeSecond // time.Second
TimeMinute // time.Minute
TimeHour   // time.Hour
TimeDay    // 24 * time.Hour
TimeWeek   // 7 * 24 * time.Hour
TimeMonth  // 30 * 24 * time.Hour
TimeYear   // 365 * 24 * time.Hour
```

### 日志常量

#### 日志级别

```go
LogLevelDebug // "debug"
LogLevelInfo  // "info"
LogLevelWarn  // "warn"
LogLevelError // "error"
LogLevelFatal // "fatal"
LogLevelPanic // "panic"
```

#### 日志格式

```go
LogFormatJSON    // "json"
LogFormatConsole // "console"
LogFormatText    // "text"
```

### 环境常量

#### 环境类型

```go
EnvDevelopment // "development"
EnvProduction  // "production"
EnvTesting     // "testing"
EnvStaging     // "staging"
```

#### 环境变量

```go
EnvAppName     // "APP_NAME"
EnvAppEnv      // "APP_ENV"
EnvAppDebug    // "APP_DEBUG"
EnvAppPort     // "APP_PORT"
EnvAppHost     // "APP_HOST"
```

### 数据库常量

#### 数据库驱动

```go
DBDriverMySQL    // "mysql"
DBDriverPostgres // "postgres"
DBDriverSQLite   // "sqlite"
DBDriverMongo    // "mongo"
```

#### 数据库配置

```go
DBConfigHost     // "host"
DBConfigPort     // "port"
DBConfigUser     // "user"
DBConfigPassword // "password"
DBConfigDatabase // "database"
DBConfigCharset  // "charset"
```

### 配置常量

#### 配置格式

```go
ConfigFormatYAML // "yaml"
ConfigFormatJSON // "json"
ConfigFormatTOML // "toml"
ConfigFormatHCL  // "hcl"
```

#### 配置文件

```go
ConfigFileDefault // "config.yml"
ConfigFileDev     // "config.dev.yml"
ConfigFileProd    // "config.prod.yml"
ConfigFileTest    // "config.test.yml"
```

### 错误常量

#### 错误码范围

```go
ErrorCodeSystemStart     // 1000
ErrorCodeSystemEnd       // 1999
ErrorCodeBusinessStart   // 2000
ErrorCodeBusinessEnd     // 2999
ErrorCodeDatabaseStart   // 3000
ErrorCodeDatabaseEnd     // 3999
ErrorCodeExternalStart   // 4000
ErrorCodeExternalEnd     // 4999
```

#### 错误类型

```go
ErrorTypeSystem    // "system"
ErrorTypeBusiness  // "business"
ErrorTypeDatabase  // "database"
ErrorTypeExternal  // "external"
ErrorTypeNetwork   // "network"
ErrorTypeTimeout   // "timeout"
```

### 工具函数

#### 时间工具

```go
// 获取当前时间戳
timestamp := constants.GetCurrentTimestamp()

// 格式化时间
formatted := constants.FormatTime(time.Now(), constants.TimeFormatISO)

// 解析时间
parsed, err := constants.ParseTime("2023-01-01T12:00:00Z", constants.TimeFormatISO)

// 检查时间是否过期
expired := constants.IsTimeExpired(someTime, 24*time.Hour)
```

#### 字符串工具

```go
// 生成UUID
uuid := constants.GenerateUUID()

// 生成随机字符串
randomStr := constants.GenerateRandomString(16)

// 检查字符串是否为空
isEmpty := constants.IsEmptyString("")

// 截断字符串
truncated := constants.TruncateString("long string", 10)
```

#### 数字工具

```go
// 生成随机整数
randomInt := constants.GenerateRandomInt(1, 100)

// 检查数字是否在范围内
inRange := constants.IsInRange(5, 1, 10)

// 限制数字范围
limited := constants.LimitRange(15, 1, 10) // 返回10
```

#### 切片工具

```go
// 检查切片是否包含元素
contains := constants.SliceContains([]string{"a", "b", "c"}, "b")

// 去重切片
unique := constants.SliceUnique([]string{"a", "b", "a", "c"})

// 过滤切片
filtered := constants.SliceFilter([]int{1, 2, 3, 4, 5}, func(x int) bool {
    return x%2 == 0
})
```

#### 映射工具

```go
// 检查映射是否包含键
hasKey := constants.MapHasKey(map[string]int{"a": 1, "b": 2}, "a")

// 获取映射值，如果不存在则返回默认值
value := constants.MapGetOrDefault(map[string]int{"a": 1}, "b", 0)

// 合并映射
merged := constants.MergeMaps(map[string]int{"a": 1}, map[string]int{"b": 2})
```

## 🏗️ 最佳实践

### 1. 常量命名

#### 使用有意义的名称

```go
// ✅ 好的命名
const (
    HTTPStatusOK = 200
    HTTPStatusNotFound = 404
    TimeFormatISO = "2006-01-02T15:04:05Z07:00"
)

// ❌ 不好的命名
const (
    OK = 200
    NF = 404
    ISO = "2006-01-02T15:04:05Z07:00"
)
```

#### 使用前缀分组

```go
// HTTP相关常量
const (
    HTTPStatusOK = 200
    HTTPStatusCreated = 201
    HTTPStatusBadRequest = 400
)

// 时间相关常量
const (
    TimeFormatISO = "2006-01-02T15:04:05Z07:00"
    TimeFormatDate = "2006-01-02"
    TimeFormatTime = "15:04:05"
)
```

### 2. 常量组织

#### 按功能分组

```go
// HTTP常量
var (
    HTTPStatuses = map[string]int{
        "OK":                   200,
        "Created":              201,
        "BadRequest":           400,
        "Unauthorized":         401,
        "NotFound":             404,
        "InternalServerError":  500,
    }
    
    HTTPMethods = []string{
        "GET",
        "POST", 
        "PUT",
        "DELETE",
        "PATCH",
    }
)

// 时间常量
var (
    TimeFormats = map[string]string{
        "ISO":     "2006-01-02T15:04:05Z07:00",
        "RFC3339": "2006-01-02T15:04:05Z07:00",
        "Date":    "2006-01-02",
        "Time":    "15:04:05",
    }
)
```

### 3. 类型安全

#### 使用强类型常量

```go
// ✅ 使用强类型
type HTTPStatus int
const (
    HTTPStatusOK HTTPStatus = 200
    HTTPStatusNotFound HTTPStatus = 404
)

// ✅ 使用字符串常量
type LogLevel string
const (
    LogLevelDebug LogLevel = "debug"
    LogLevelInfo  LogLevel = "info"
    LogLevelError LogLevel = "error"
)

// ❌ 避免使用interface{}
const (
    SomeValue interface{} = "value"
)
```

### 4. 常量验证

#### 验证常量值

```go
// 验证HTTP状态码
func IsValidHTTPStatus(status int) bool {
    return status >= 100 && status <= 599
}

// 验证HTTP方法
func IsValidHTTPMethod(method string) bool {
    validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
    return constants.SliceContains(validMethods, method)
}

// 验证日志级别
func IsValidLogLevel(level string) bool {
    validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
    return constants.SliceContains(validLevels, level)
}
```

### 5. 国际化支持

#### 多语言常量

```go
// 错误消息常量
var ErrorMessages = map[string]map[string]string{
    "zh": {
        "InvalidParam":     "参数无效",
        "NotFound":         "资源不存在",
        "Unauthorized":     "未授权",
        "InternalError":    "服务器内部错误",
    },
    "en": {
        "InvalidParam":     "Invalid parameter",
        "NotFound":         "Resource not found",
        "Unauthorized":     "Unauthorized",
        "InternalError":    "Internal server error",
    },
}

// 获取本地化消息
func GetLocalizedMessage(key, lang string) string {
    if messages, exists := ErrorMessages[lang]; exists {
        if message, exists := messages[key]; exists {
            return message
        }
    }
    // 返回英文作为默认值
    if messages, exists := ErrorMessages["en"]; exists {
        if message, exists := messages[key]; exists {
            return message
        }
    }
    return key
}
```

### 6. 配置常量

#### 环境相关常量

```go
// 根据环境获取配置
func GetConfigByEnv(env string) map[string]interface{} {
    switch env {
    case constants.EnvDevelopment:
        return map[string]interface{}{
            "debug": true,
            "log_level": constants.LogLevelDebug,
        }
    case constants.EnvProduction:
        return map[string]interface{}{
            "debug": false,
            "log_level": constants.LogLevelInfo,
        }
    case constants.EnvTesting:
        return map[string]interface{}{
            "debug": true,
            "log_level": constants.LogLevelDebug,
        }
    default:
        return map[string]interface{}{
            "debug": false,
            "log_level": constants.LogLevelInfo,
        }
    }
}
```

### 7. 工具函数使用

#### 时间处理

```go
// 格式化当前时间
func GetCurrentTimeFormatted() string {
    return constants.FormatTime(time.Now(), constants.TimeFormatISO)
}

// 检查时间是否在指定范围内
func IsTimeInRange(t time.Time, start, end time.Time) bool {
    return t.After(start) && t.Before(end)
}

// 获取时间差
func GetTimeDifference(t1, t2 time.Time) time.Duration {
    return t2.Sub(t1)
}
```

#### 字符串处理

```go
// 生成唯一标识符
func GenerateUniqueID() string {
    return constants.GenerateUUID()
}

// 安全地截断字符串
func SafeTruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return constants.TruncateString(s, maxLen-3) + "..."
}

// 检查字符串是否为有效邮箱
func IsValidEmail(email string) bool {
    // 简单的邮箱验证
    return strings.Contains(email, "@") && strings.Contains(email, ".")
}
```

## 🧪 测试

### 单元测试

```go
func TestConstants(t *testing.T) {
    // 测试HTTP状态码
    if constants.HTTPStatusOK != 200 {
        t.Errorf("期望HTTPStatusOK = 200，实际 = %d", constants.HTTPStatusOK)
    }
    
    if constants.HTTPStatusNotFound != 404 {
        t.Errorf("期望HTTPStatusNotFound = 404，实际 = %d", constants.HTTPStatusNotFound)
    }
    
    // 测试HTTP方法
    if constants.HTTPMethodGET != "GET" {
        t.Errorf("期望HTTPMethodGET = 'GET'，实际 = '%s'", constants.HTTPMethodGET)
    }
    
    // 测试时间格式
    if constants.TimeFormatISO != "2006-01-02T15:04:05Z07:00" {
        t.Errorf("期望TimeFormatISO = '2006-01-02T15:04:05Z07:00'，实际 = '%s'", constants.TimeFormatISO)
    }
}

func TestUtilityFunctions(t *testing.T) {
    // 测试UUID生成
    uuid1 := constants.GenerateUUID()
    uuid2 := constants.GenerateUUID()
    
    if uuid1 == uuid2 {
        t.Error("生成的UUID应该不同")
    }
    
    if len(uuid1) == 0 {
        t.Error("生成的UUID不应该为空")
    }
    
    // 测试随机字符串生成
    randomStr := constants.GenerateRandomString(10)
    if len(randomStr) != 10 {
        t.Errorf("期望随机字符串长度为10，实际 = %d", len(randomStr))
    }
    
    // 测试字符串截断
    truncated := constants.TruncateString("这是一个很长的字符串", 5)
    if len(truncated) > 5 {
        t.Errorf("截断后的字符串长度应该不超过5，实际 = %d", len(truncated))
    }
}
```

### 集成测试

```go
func TestConstantsIntegration(t *testing.T) {
    // 测试HTTP状态码验证
    validStatuses := []int{200, 201, 400, 404, 500}
    for _, status := range validStatuses {
        if !IsValidHTTPStatus(status) {
            t.Errorf("状态码 %d 应该是有效的", status)
        }
    }
    
    invalidStatuses := []int{0, 99, 600, 999}
    for _, status := range invalidStatuses {
        if IsValidHTTPStatus(status) {
            t.Errorf("状态码 %d 应该是无效的", status)
        }
    }
    
    // 测试HTTP方法验证
    validMethods := []string{"GET", "POST", "PUT", "DELETE"}
    for _, method := range validMethods {
        if !IsValidHTTPMethod(method) {
            t.Errorf("HTTP方法 %s 应该是有效的", method)
        }
    }
    
    invalidMethods := []string{"INVALID", "TEST", ""}
    for _, method := range invalidMethods {
        if IsValidHTTPMethod(method) {
            t.Errorf("HTTP方法 %s 应该是无效的", method)
        }
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 常量冲突

```go
// ❌ 避免在不同包中定义相同的常量名
package config
const StatusOK = 200

package http
const StatusOK = 200 // 冲突！

// ✅ 使用前缀避免冲突
package config
const ConfigStatusOK = 200

package http
const HTTPStatusOK = 200
```

#### 2. 常量类型不匹配

```go
// ❌ 类型不匹配
const StatusOK = 200
var status string = StatusOK // 编译错误

// ✅ 使用正确的类型
const StatusOK int = 200
var status int = StatusOK // 正确
```

#### 3. 常量值验证

```go
// 添加常量验证函数
func ValidateConstants() error {
    // 验证HTTP状态码
    if HTTPStatusOK != 200 {
        return fmt.Errorf("HTTPStatusOK 应该是 200，实际是 %d", HTTPStatusOK)
    }
    
    // 验证时间格式
    testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
    formatted := testTime.Format(TimeFormatISO)
    if !strings.Contains(formatted, "2023-01-01T12:00:00") {
        return fmt.Errorf("时间格式不正确: %s", formatted)
    }
    
    return nil
}
```

### 调试技巧

```go
// 1. 打印所有常量
func PrintAllConstants() {
    fmt.Printf("HTTP状态码:\n")
    fmt.Printf("  OK: %d\n", HTTPStatusOK)
    fmt.Printf("  NotFound: %d\n", HTTPStatusNotFound)
    fmt.Printf("  InternalServerError: %d\n", HTTPStatusInternalServerError)
    
    fmt.Printf("HTTP方法:\n")
    fmt.Printf("  GET: %s\n", HTTPMethodGET)
    fmt.Printf("  POST: %s\n", HTTPMethodPOST)
    fmt.Printf("  PUT: %s\n", HTTPMethodPUT)
    fmt.Printf("  DELETE: %s\n", HTTPMethodDELETE)
    
    fmt.Printf("时间格式:\n")
    fmt.Printf("  ISO: %s\n", TimeFormatISO)
    fmt.Printf("  Date: %s\n", TimeFormatDate)
    fmt.Printf("  Time: %s\n", TimeFormatTime)
}

// 2. 验证常量一致性
func ValidateConstantConsistency() error {
    // 检查HTTP状态码是否在有效范围内
    statusCodes := []int{HTTPStatusOK, HTTPStatusNotFound, HTTPStatusInternalServerError}
    for _, code := range statusCodes {
        if code < 100 || code > 599 {
            return fmt.Errorf("HTTP状态码 %d 超出有效范围", code)
        }
    }
    
    return nil
}
```

## 📚 相关链接

- [返回首页](../README.md) 