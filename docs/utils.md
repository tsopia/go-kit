# 工具函数 (utils)

常用工具函数集合，提供字符串处理、时间操作、加密解密、文件操作等实用功能。

## 🚀 特性

- ✅ 字符串处理和验证
- ✅ 时间操作和格式化
- ✅ 加密解密功能
- ✅ 文件操作工具
- ✅ 网络工具函数
- ✅ 数据结构操作
- ✅ 并发安全

## 📖 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/tsopia/go-kit/utils"
)

func main() {
    // 字符串工具
    fmt.Println(utils.IsEmpty(""))           // true
    fmt.Println(utils.TruncateString("hello world", 5)) // "hello"
    
    // 时间工具
    fmt.Println(utils.FormatTime(time.Now())) // "2023-01-01 12:00:00"
    fmt.Println(utils.GetTimestamp())         // 1672531200
    
    // 加密工具
    hashed := utils.HashPassword("password")
    fmt.Println(utils.CheckPassword("password", hashed)) // true
    
    // 文件工具
    exists := utils.FileExists("config.yml")
    fmt.Println(exists) // true/false
}
```

### 工具函数分类

```go
// 字符串工具
utils.IsEmpty("")                    // 检查字符串是否为空
utils.TruncateString("hello", 3)     // 截断字符串
utils.GenerateRandomString(10)       // 生成随机字符串
utils.CamelToSnake("userName")       // 驼峰转下划线
utils.SnakeToCamel("user_name")      // 下划线转驼峰

// 时间工具
utils.FormatTime(time.Now())         // 格式化时间
utils.ParseTime("2023-01-01")       // 解析时间
utils.GetTimestamp()                 // 获取时间戳
utils.IsExpired(someTime)           // 检查是否过期

// 加密工具
utils.HashPassword("password")       // 哈希密码
utils.CheckPassword("pass", hash)    // 验证密码
utils.GenerateToken()               // 生成令牌
utils.EncryptText("secret")         // 加密文本
utils.DecryptText(encrypted)        // 解密文本

// 文件工具
utils.FileExists("file.txt")        // 检查文件是否存在
utils.ReadFile("file.txt")          // 读取文件
utils.WriteFile("file.txt", data)   // 写入文件
utils.CreateDir("path")             // 创建目录

// 网络工具
utils.GetLocalIP()                  // 获取本地IP
utils.IsValidIP("192.168.1.1")     // 验证IP地址
utils.IsValidEmail("test@example.com") // 验证邮箱
utils.IsValidURL("https://example.com") // 验证URL
```

## 🔧 API 参考

### 字符串工具

#### 基本字符串操作

```go
// 检查字符串是否为空
func IsEmpty(s string) bool

// 检查字符串是否为空白
func IsBlank(s string) bool

// 截断字符串
func TruncateString(s string, maxLen int) string

// 生成随机字符串
func GenerateRandomString(length int) string

// 生成UUID
func GenerateUUID() string

// 驼峰转下划线
func CamelToSnake(s string) string

// 下划线转驼峰
func SnakeToCamel(s string) string

// 首字母大写
func Capitalize(s string) string

// 首字母小写
func Uncapitalize(s string) string
```

#### 字符串验证

```go
// 验证邮箱格式
func IsValidEmail(email string) bool

// 验证手机号格式
func IsValidPhone(phone string) bool

// 验证身份证号
func IsValidIDCard(idCard string) bool

// 验证URL格式
func IsValidURL(url string) bool

// 验证IP地址
func IsValidIP(ip string) bool

// 验证域名
func IsValidDomain(domain string) bool
```

#### 字符串转换

```go
// 字符串转整数
func StringToInt(s string) (int, error)

// 字符串转浮点数
func StringToFloat(s string) (float64, error)

// 字符串转布尔值
func StringToBool(s string) (bool, error)

// 整数转字符串
func IntToString(i int) string

// 浮点数转字符串
func FloatToString(f float64) string

// 布尔值转字符串
func BoolToString(b bool) string
```

### 时间工具

#### 时间格式化

```go
// 格式化时间
func FormatTime(t time.Time) string

// 格式化时间戳
func FormatTimestamp(timestamp int64) string

// 解析时间字符串
func ParseTime(timeStr string) (time.Time, error)

// 获取当前时间戳
func GetTimestamp() int64

// 获取当前时间戳（毫秒）
func GetTimestampMillis() int64

// 获取当前时间戳（纳秒）
func GetTimestampNanos() int64
```

#### 时间计算

```go
// 检查时间是否过期
func IsExpired(t time.Time, duration time.Duration) bool

// 计算时间差
func TimeDiff(t1, t2 time.Time) time.Duration

// 添加时间
func AddTime(t time.Time, duration time.Duration) time.Time

// 减去时间
func SubTime(t time.Time, duration time.Duration) time.Time

// 获取时间范围
func GetTimeRange(start, end time.Time) []time.Time
```

#### 时间验证

```go
// 验证时间格式
func IsValidTimeFormat(timeStr, format string) bool

// 验证日期格式
func IsValidDateFormat(dateStr string) bool

// 验证时间戳
func IsValidTimestamp(timestamp int64) bool

// 检查是否为工作日
func IsWorkday(t time.Time) bool

// 检查是否为周末
func IsWeekend(t time.Time) bool
```

### 加密工具

#### 密码处理

```go
// 哈希密码
func HashPassword(password string) (string, error)

// 验证密码
func CheckPassword(password, hash string) bool

// 生成盐值
func GenerateSalt() string

// 哈希字符串
func HashString(s string) string

// MD5哈希
func MD5Hash(s string) string

// SHA256哈希
func SHA256Hash(s string) string
```

#### 加密解密

```go
// 加密文本
func EncryptText(text, key string) (string, error)

// 解密文本
func DecryptText(encryptedText, key string) (string, error)

// 生成密钥
func GenerateKey() (string, error)

// 生成令牌
func GenerateToken() string

// 生成JWT令牌
func GenerateJWT(payload map[string]interface{}, secret string) (string, error)

// 验证JWT令牌
func ValidateJWT(token, secret string) (map[string]interface{}, error)
```

### 文件工具

#### 文件操作

```go
// 检查文件是否存在
func FileExists(path string) bool

// 读取文件内容
func ReadFile(path string) ([]byte, error)

// 写入文件
func WriteFile(path string, data []byte) error

// 追加写入文件
func AppendFile(path string, data []byte) error

// 删除文件
func DeleteFile(path string) error

// 复制文件
func CopyFile(src, dst string) error

// 移动文件
func MoveFile(src, dst string) error
```

#### 目录操作

```go
// 创建目录
func CreateDir(path string) error

// 创建多级目录
func CreateDirs(path string) error

// 删除目录
func DeleteDir(path string) error

// 列出目录内容
func ListDir(path string) ([]string, error)

// 获取文件大小
func GetFileSize(path string) (int64, error)

// 获取文件信息
func GetFileInfo(path string) (os.FileInfo, error)
```

#### 文件验证

```go
// 检查是否为目录
func IsDir(path string) bool

// 检查是否为文件
func IsFile(path string) bool

// 检查文件权限
func HasPermission(path string, mode os.FileMode) bool

// 验证文件扩展名
func HasExtension(path, ext string) bool

// 获取文件扩展名
func GetFileExtension(path string) string

// 获取文件名（不含扩展名）
func GetFileNameWithoutExt(path string) string
```

### 网络工具

#### IP地址处理

```go
// 获取本地IP地址
func GetLocalIP() string

// 获取公网IP地址
func GetPublicIP() (string, error)

// 验证IP地址格式
func IsValidIP(ip string) bool

// 验证IPv4地址
func IsValidIPv4(ip string) bool

// 验证IPv6地址
func IsValidIPv6(ip string) bool

// IP地址转整数
func IPToInt(ip string) (uint32, error)

// 整数转IP地址
func IntToIP(ipInt uint32) string
```

#### 网络验证

```go
// 验证邮箱格式
func IsValidEmail(email string) bool

// 验证URL格式
func IsValidURL(url string) bool

// 验证域名格式
func IsValidDomain(domain string) bool

// 验证端口号
func IsValidPort(port int) bool

// 验证MAC地址
func IsValidMAC(mac string) bool

// 验证HTTP状态码
func IsValidHTTPStatus(status int) bool
```

#### 网络请求

```go
// 发送HTTP GET请求
func HTTPGet(url string) ([]byte, error)

// 发送HTTP POST请求
func HTTPPost(url string, data []byte) ([]byte, error)

// 发送HTTP请求
func HTTPRequest(method, url string, data []byte, headers map[string]string) ([]byte, error)

// 检查URL是否可访问
func IsURLReachable(url string) bool

// 获取URL状态码
func GetURLStatusCode(url string) (int, error)
```

### 数据结构工具

#### 切片操作

```go
// 检查切片是否包含元素
func SliceContains(slice []interface{}, item interface{}) bool

// 切片去重
func SliceUnique(slice []interface{}) []interface{}

// 切片过滤
func SliceFilter(slice []interface{}, predicate func(interface{}) bool) []interface{}

// 切片映射
func SliceMap(slice []interface{}, mapper func(interface{}) interface{}) []interface{}

// 切片排序
func SliceSort(slice []interface{}, less func(interface{}, interface{}) bool) []interface{}

// 切片分页
func SlicePaginate(slice []interface{}, page, size int) []interface{}
```

#### 映射操作

```go
// 检查映射是否包含键
func MapHasKey(m map[string]interface{}, key string) bool

// 获取映射值，如果不存在则返回默认值
func MapGetOrDefault(m map[string]interface{}, key string, defaultValue interface{}) interface{}

// 合并映射
func MergeMaps(maps ...map[string]interface{}) map[string]interface{}

// 映射键列表
func MapKeys(m map[string]interface{}) []string

// 映射值列表
func MapValues(m map[string]interface{}) []interface{}

// 映射过滤
func MapFilter(m map[string]interface{}, predicate func(string, interface{}) bool) map[string]interface{}
```

#### 集合操作

```go
// 集合交集
func SetIntersection(sets ...[]interface{}) []interface{}

// 集合并集
func SetUnion(sets ...[]interface{}) []interface{}

// 集合差集
func SetDifference(set1, set2 []interface{}) []interface{}

// 检查集合是否相等
func SetEqual(set1, set2 []interface{}) bool

// 检查集合是否包含
func SetContains(set, subset []interface{}) bool
```

## 🏗️ 最佳实践

### 1. 字符串处理

#### 安全的字符串操作

```go
// ✅ 安全的字符串截断
func SafeTruncateString(s string, maxLen int) string {
    if maxLen <= 0 {
        return ""
    }
    
    if len(s) <= maxLen {
        return s
    }
    
    // 避免截断UTF-8字符
    runes := []rune(s)
    if len(runes) <= maxLen {
        return string(runes)
    }
    
    return string(runes[:maxLen-3]) + "..."
}

// ✅ 安全的字符串转换
func SafeStringToInt(s string) (int, error) {
    if utils.IsEmpty(s) {
        return 0, fmt.Errorf("字符串为空")
    }
    
    return strconv.Atoi(strings.TrimSpace(s))
}

// ✅ 验证字符串格式
func ValidateEmail(email string) error {
    if utils.IsEmpty(email) {
        return fmt.Errorf("邮箱不能为空")
    }
    
    if !utils.IsValidEmail(email) {
        return fmt.Errorf("邮箱格式无效")
    }
    
    return nil
}
```

### 2. 时间处理

#### 统一的时间格式

```go
// 定义时间格式常量
const (
    TimeFormatDefault = "2006-01-02 15:04:05"
    TimeFormatDate    = "2006-01-02"
    TimeFormatTime    = "15:04:05"
    TimeFormatISO     = "2006-01-02T15:04:05Z07:00"
)

// ✅ 统一的时间格式化
func FormatTimeStandard(t time.Time) string {
    return t.Format(TimeFormatDefault)
}

// ✅ 时区安全的时间处理
func ParseTimeWithLocation(timeStr, location string) (time.Time, error) {
    loc, err := time.LoadLocation(location)
    if err != nil {
        return time.Time{}, err
    }
    
    t, err := time.Parse(TimeFormatDefault, timeStr)
    if err != nil {
        return time.Time{}, err
    }
    
    return t.In(loc), nil
}

// ✅ 时间范围验证
func ValidateTimeRange(start, end time.Time) error {
    if start.After(end) {
        return fmt.Errorf("开始时间不能晚于结束时间")
    }
    
    if utils.IsExpired(start, 24*time.Hour) {
        return fmt.Errorf("开始时间不能是过去的时间")
    }
    
    return nil
}
```

### 3. 加密安全

#### 安全的密码处理

```go
// ✅ 使用强密码策略
func ValidatePassword(password string) error {
    if len(password) < 8 {
        return fmt.Errorf("密码长度至少8位")
    }
    
    hasUpper := false
    hasLower := false
    hasDigit := false
    hasSpecial := false
    
    for _, char := range password {
        switch {
        case unicode.IsUpper(char):
            hasUpper = true
        case unicode.IsLower(char):
            hasLower = true
        case unicode.IsDigit(char):
            hasDigit = true
        case unicode.IsPunct(char) || unicode.IsSymbol(char):
            hasSpecial = true
        }
    }
    
    if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
        return fmt.Errorf("密码必须包含大小写字母、数字和特殊字符")
    }
    
    return nil
}

// ✅ 安全的密码哈希
func HashPasswordSecure(password string) (string, error) {
    if err := ValidatePassword(password); err != nil {
        return "", err
    }
    
    return utils.HashPassword(password)
}

// ✅ 安全的令牌生成
func GenerateSecureToken() string {
    token := utils.GenerateToken()
    // 添加额外的随机性
    timestamp := utils.GetTimestamp()
    return fmt.Sprintf("%s_%d", token, timestamp)
}
```

### 4. 文件操作

#### 安全的文件操作

```go
// ✅ 安全的文件读取
func ReadFileSafely(path string) ([]byte, error) {
    if !utils.FileExists(path) {
        return nil, fmt.Errorf("文件不存在: %s", path)
    }
    
    if utils.IsDir(path) {
        return nil, fmt.Errorf("路径是目录: %s", path)
    }
    
    return utils.ReadFile(path)
}

// ✅ 安全的文件写入
func WriteFileSafely(path string, data []byte) error {
    // 创建目录
    dir := filepath.Dir(path)
    if err := utils.CreateDirs(dir); err != nil {
        return fmt.Errorf("创建目录失败: %v", err)
    }
    
    // 写入临时文件
    tempPath := path + ".tmp"
    if err := utils.WriteFile(tempPath, data); err != nil {
        return fmt.Errorf("写入临时文件失败: %v", err)
    }
    
    // 原子性重命名
    if err := os.Rename(tempPath, path); err != nil {
        utils.DeleteFile(tempPath) // 清理临时文件
        return fmt.Errorf("重命名文件失败: %v", err)
    }
    
    return nil
}

// ✅ 文件备份
func BackupFile(path string) error {
    if !utils.FileExists(path) {
        return fmt.Errorf("文件不存在: %s", path)
    }
    
    backupPath := path + ".backup"
    return utils.CopyFile(path, backupPath)
}
```

### 5. 网络工具

#### 网络连接检查

```go
// ✅ 检查网络连接
func CheckNetworkConnectivity() error {
    // 检查DNS解析
    if _, err := net.LookupHost("google.com"); err != nil {
        return fmt.Errorf("DNS解析失败: %v", err)
    }
    
    // 检查HTTP连接
    if _, err := utils.HTTPGet("https://httpbin.org/get"); err != nil {
        return fmt.Errorf("HTTP连接失败: %v", err)
    }
    
    return nil
}

// ✅ 获取网络信息
func GetNetworkInfo() map[string]interface{} {
    info := make(map[string]interface{})
    
    // 本地IP
    info["local_ip"] = utils.GetLocalIP()
    
    // 公网IP
    if publicIP, err := utils.GetPublicIP(); err == nil {
        info["public_ip"] = publicIP
    }
    
    // 主机名
    if hostname, err := os.Hostname(); err == nil {
        info["hostname"] = hostname
    }
    
    return info
}

// ✅ 网络延迟测试
func TestNetworkLatency(url string) (time.Duration, error) {
    start := time.Now()
    
    _, err := utils.HTTPGet(url)
    if err != nil {
        return 0, err
    }
    
    return time.Since(start), nil
}
```

### 6. 数据结构操作

#### 类型安全的切片操作

```go
// ✅ 类型安全的切片操作
func SliceContainsString(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func SliceContainsInt(slice []int, item int) bool {
    for _, i := range slice {
        if i == item {
            return true
        }
    }
    return false
}

// ✅ 切片去重
func SliceUniqueString(slice []string) []string {
    seen := make(map[string]bool)
    result := make([]string, 0)
    
    for _, item := range slice {
        if !seen[item] {
            seen[item] = true
            result = append(result, item)
        }
    }
    
    return result
}

// ✅ 切片分页
func SlicePaginateString(slice []string, page, size int) []string {
    if page <= 0 || size <= 0 {
        return []string{}
    }
    
    start := (page - 1) * size
    end := start + size
    
    if start >= len(slice) {
        return []string{}
    }
    
    if end > len(slice) {
        end = len(slice)
    }
    
    return slice[start:end]
}
```

### 7. 错误处理

#### 工具函数错误处理

```go
// ✅ 包装工具函数错误
func SafeReadFile(path string) ([]byte, error) {
    data, err := utils.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("读取文件失败 [%s]: %v", path, err)
    }
    return data, nil
}

func SafeParseTime(timeStr string) (time.Time, error) {
    t, err := utils.ParseTime(timeStr)
    if err != nil {
        return time.Time{}, fmt.Errorf("解析时间失败 [%s]: %v", timeStr, err)
    }
    return t, nil
}

// ✅ 批量操作错误处理
func BatchProcessFiles(paths []string, processor func([]byte) error) error {
    var errors []string
    
    for _, path := range paths {
        data, err := utils.ReadFile(path)
        if err != nil {
            errors = append(errors, fmt.Sprintf("[%s]: %v", path, err))
            continue
        }
        
        if err := processor(data); err != nil {
            errors = append(errors, fmt.Sprintf("[%s]: %v", path, err))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("批量处理失败:\n%s", strings.Join(errors, "\n"))
    }
    
    return nil
}
```

## 🧪 测试

### 单元测试

```go
func TestStringUtils(t *testing.T) {
    // 测试字符串截断
    tests := []struct {
        input    string
        maxLen   int
        expected string
    }{
        {"hello world", 5, "hello"},
        {"测试", 1, "测"},
        {"", 10, ""},
    }
    
    for _, tt := range tests {
        result := utils.TruncateString(tt.input, tt.maxLen)
        if result != tt.expected {
            t.Errorf("TruncateString(%s, %d) = %s, 期望 %s", 
                tt.input, tt.maxLen, result, tt.expected)
        }
    }
    
    // 测试邮箱验证
    emailTests := []struct {
        email    string
        expected bool
    }{
        {"test@example.com", true},
        {"invalid-email", false},
        {"", false},
    }
    
    for _, tt := range emailTests {
        result := utils.IsValidEmail(tt.email)
        if result != tt.expected {
            t.Errorf("IsValidEmail(%s) = %t, 期望 %t", 
                tt.email, result, tt.expected)
        }
    }
}

func TestTimeUtils(t *testing.T) {
    // 测试时间格式化
    now := time.Now()
    formatted := utils.FormatTime(now)
    
    if utils.IsEmpty(formatted) {
        t.Error("格式化时间不应该为空")
    }
    
    // 测试时间戳
    timestamp := utils.GetTimestamp()
    if timestamp <= 0 {
        t.Error("时间戳应该大于0")
    }
    
    // 测试时间过期检查
    past := time.Now().Add(-1 * time.Hour)
    if !utils.IsExpired(past, 30*time.Minute) {
        t.Error("过去的时间应该被认为是过期的")
    }
}

func TestFileUtils(t *testing.T) {
    // 创建临时文件
    tempFile := filepath.Join(t.TempDir(), "test.txt")
    testData := []byte("test content")
    
    // 测试文件写入
    err := utils.WriteFile(tempFile, testData)
    if err != nil {
        t.Fatalf("写入文件失败: %v", err)
    }
    
    // 测试文件存在检查
    if !utils.FileExists(tempFile) {
        t.Error("文件应该存在")
    }
    
    // 测试文件读取
    data, err := utils.ReadFile(tempFile)
    if err != nil {
        t.Fatalf("读取文件失败: %v", err)
    }
    
    if string(data) != string(testData) {
        t.Errorf("读取的数据不匹配，期望 %s，实际 %s", 
            string(testData), string(data))
    }
}
```

### 集成测试

```go
func TestNetworkUtils(t *testing.T) {
    // 测试本地IP获取
    localIP := utils.GetLocalIP()
    if utils.IsEmpty(localIP) {
        t.Error("本地IP不应该为空")
    }
    
    if !utils.IsValidIP(localIP) {
        t.Errorf("本地IP格式无效: %s", localIP)
    }
    
    // 测试URL验证
    validURLs := []string{
        "https://example.com",
        "http://localhost:8080",
        "ftp://ftp.example.com",
    }
    
    for _, url := range validURLs {
        if !utils.IsValidURL(url) {
            t.Errorf("URL应该有效: %s", url)
        }
    }
    
    invalidURLs := []string{
        "not-a-url",
        "http://",
        "https://",
    }
    
    for _, url := range invalidURLs {
        if utils.IsValidURL(url) {
            t.Errorf("URL应该无效: %s", url)
        }
    }
}

func TestCryptoUtils(t *testing.T) {
    password := "testpassword123"
    
    // 测试密码哈希
    hash, err := utils.HashPassword(password)
    if err != nil {
        t.Fatalf("密码哈希失败: %v", err)
    }
    
    if utils.IsEmpty(hash) {
        t.Error("密码哈希不应该为空")
    }
    
    // 测试密码验证
    if !utils.CheckPassword(password, hash) {
        t.Error("密码验证应该成功")
    }
    
    // 测试错误密码
    if utils.CheckPassword("wrongpassword", hash) {
        t.Error("错误密码验证应该失败")
    }
    
    // 测试令牌生成
    token1 := utils.GenerateToken()
    token2 := utils.GenerateToken()
    
    if token1 == token2 {
        t.Error("生成的令牌应该不同")
    }
    
    if utils.IsEmpty(token1) || utils.IsEmpty(token2) {
        t.Error("生成的令牌不应该为空")
    }
}
```

## 🔍 故障排除

### 常见问题

#### 1. 字符串编码问题

```go
// ❌ 可能导致乱码的截断
func BadTruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] // 可能截断UTF-8字符
}

// ✅ 安全的UTF-8截断
func SafeTruncateString(s string, maxLen int) string {
    runes := []rune(s)
    if len(runes) <= maxLen {
        return string(runes)
    }
    return string(runes[:maxLen])
}
```

#### 2. 时间时区问题

```go
// ❌ 忽略时区的时间处理
func BadParseTime(timeStr string) (time.Time, error) {
    return time.Parse("2006-01-02 15:04:05", timeStr)
}

// ✅ 考虑时区的时间处理
func SafeParseTime(timeStr string) (time.Time, error) {
    return time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
}
```

#### 3. 文件权限问题

```go
// ❌ 不检查文件权限
func BadWriteFile(path string, data []byte) error {
    return ioutil.WriteFile(path, data, 0644)
}

// ✅ 检查文件权限
func SafeWriteFile(path string, data []byte) error {
    // 检查目录权限
    dir := filepath.Dir(path)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("创建目录失败: %v", err)
        }
    }
    
    return ioutil.WriteFile(path, data, 0644)
}
```

### 性能优化

```go
// 1. 使用对象池减少内存分配
var stringBuilderPool = sync.Pool{
    New: func() interface{} {
        return &strings.Builder{}
    },
}

func EfficientStringConcat(strings []string) string {
    builder := stringBuilderPool.Get().(*strings.Builder)
    defer func() {
        builder.Reset()
        stringBuilderPool.Put(builder)
    }()
    
    for _, s := range strings {
        builder.WriteString(s)
    }
    
    return builder.String()
}

// 2. 缓存常用计算结果
var timeFormatCache = make(map[string]string)
var timeFormatMutex sync.RWMutex

func CachedFormatTime(t time.Time, format string) string {
    timeFormatMutex.RLock()
    if cached, exists := timeFormatCache[format]; exists {
        timeFormatMutex.RUnlock()
        return t.Format(cached)
    }
    timeFormatMutex.RUnlock()
    
    timeFormatMutex.Lock()
    defer timeFormatMutex.Unlock()
    
    // 双重检查
    if cached, exists := timeFormatCache[format]; exists {
        return t.Format(cached)
    }
    
    timeFormatCache[format] = format
    return t.Format(format)
}

// 3. 批量操作优化
func BatchProcessWithWorkerPool(items []string, processor func(string) error, workers int) error {
    if workers <= 0 {
        workers = runtime.NumCPU()
    }
    
    semaphore := make(chan struct{}, workers)
    var wg sync.WaitGroup
    var errors []error
    var errorMutex sync.Mutex
    
    for _, item := range items {
        wg.Add(1)
        go func(item string) {
            defer wg.Done()
            
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            if err := processor(item); err != nil {
                errorMutex.Lock()
                errors = append(errors, err)
                errorMutex.Unlock()
            }
        }(item)
    }
    
    wg.Wait()
    
    if len(errors) > 0 {
        return fmt.Errorf("批量处理失败: %v", errors)
    }
    
    return nil
}
```

## 📚 相关链接

- [返回首页](../README.md) 