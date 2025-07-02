package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Validator 验证器接口
type Validator interface {
	Validate(value interface{}) error
}

// Converter 转换器接口
type Converter interface {
	Convert(value interface{}) (interface{}, error)
}

// Encoder 编码器接口
type Encoder interface {
	Encode(data interface{}) (string, error)
	Decode(data string, v interface{}) error
}

// === 字符串工具 ===

// IsEmpty 检查字符串是否为空或只包含空白字符
func IsEmpty(str string) bool {
	return strings.TrimSpace(str) == ""
}

// IsNotEmpty 检查字符串是否不为空
func IsNotEmpty(str string) bool {
	return !IsEmpty(str)
}

// TruncateString 截断字符串到指定长度
func TruncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}

	runes := []rune(str)
	if len(runes) <= length {
		return str
	}

	return string(runes[:length]) + "..."
}

// CamelToSnake 驼峰命名转蛇形命名
func CamelToSnake(str string) string {
	var result strings.Builder
	for i, char := range str {
		if unicode.IsUpper(char) && i > 0 {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(char))
	}
	return result.String()
}

// SnakeToCamel 蛇形命名转驼峰命名
func SnakeToCamel(str string) string {
	words := strings.Split(str, "_")
	var result strings.Builder

	for i, word := range words {
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else {
			result.WriteString(strings.Title(word))
		}
	}
	return result.String()
}

// MaskString 掩码字符串敏感信息
func MaskString(str string, start, end int, mask rune) string {
	runes := []rune(str)
	length := len(runes)

	if start < 0 || end < 0 || start >= length || end >= length || start > end {
		return str
	}

	for i := start; i <= end; i++ {
		runes[i] = mask
	}

	return string(runes)
}

// RandomString 生成随机字符串
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result)
}

// === 数字工具 ===

// SafeParseInt 安全解析整数，失败时返回默认值
func SafeParseInt(str string, defaultValue int) int {
	if value, err := strconv.Atoi(str); err == nil {
		return value
	}
	return defaultValue
}

// SafeParseFloat 安全解析浮点数，失败时返回默认值
func SafeParseFloat(str string, defaultValue float64) float64 {
	if value, err := strconv.ParseFloat(str, 64); err == nil {
		return value
	}
	return defaultValue
}

// InRange 检查数字是否在指定范围内
func InRange(value, min, max int) bool {
	return value >= min && value <= max
}

// ClampInt 将整数限制在指定范围内
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampFloat 将浮点数限制在指定范围内
func ClampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// === 时间工具 ===

// FormatDuration 格式化时间间隔为人类可读格式
func FormatDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
	return fmt.Sprintf("%.1fd", duration.Hours()/24)
}

// ParseTimeWithFormats 尝试使用多种格式解析时间
func ParseTimeWithFormats(timeStr string, formats []string) (time.Time, error) {
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", timeStr)
}

// IsBusinessDay 检查是否为工作日
func IsBusinessDay(t time.Time) bool {
	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

// NextBusinessDay 获取下一个工作日
func NextBusinessDay(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	for !IsBusinessDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// === 文件工具 ===

// EnsureDir 确保目录存在，不存在则创建
func EnsureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// IsValidPath 检查路径是否有效
func IsValidPath(path string) bool {
	_, err := filepath.Abs(path)
	return err == nil
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		return ext[1:] // 去掉点号
	}
	return ""
}

// === 加密工具 ===

// MD5Hash 计算MD5哈希
func MD5Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash 计算SHA256哈希
func SHA256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateSecureToken 生成安全令牌
func GenerateSecureToken(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)

	for i := range token {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		token[i] = charset[randomIndex.Int64()]
	}

	return string(token)
}

// === 验证工具 ===

// IsValidEmail 验证电子邮件地址
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidURL 验证URL
func IsValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

// IsValidPhone 验证电话号码（简单版本）
func IsValidPhone(phone string) bool {
	// 简单的电话号码验证，只检查数字和常见符号
	phoneRegex := regexp.MustCompile(`^[+]?[\d\s\-\(\)]+$`)
	return phoneRegex.MatchString(phone) && len(strings.ReplaceAll(strings.ReplaceAll(phone, " ", ""), "-", "")) >= 10
}

// IsStrongPassword 验证强密码
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

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

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// === JSON工具 ===

// PrettyJSON 美化JSON输出
func PrettyJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CompactJSON 压缩JSON输出
func CompactJSON(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// IsValidJSON 验证JSON格式
func IsValidJSON(jsonStr string) bool {
	var js interface{}
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}

// JSONToMap 将JSON字符串转换为map
func JSONToMap(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

// === 反射工具 ===

// IsNil 检查接口是否为nil
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}

	value := reflect.ValueOf(i)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// IsZero 检查值是否为零值
func IsZero(i interface{}) bool {
	if i == nil {
		return true
	}

	value := reflect.ValueOf(i)
	return value.IsZero()
}

// GetStructFields 获取结构体字段信息
func GetStructFields(obj interface{}) []string {
	var fields []string

	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return fields
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			fields = append(fields, field.Name)
		}
	}

	return fields
}

// CopyStruct 复制结构体（浅拷贝）
func CopyStruct(src, dst interface{}) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	if srcValue.Kind() == reflect.Ptr {
		srcValue = srcValue.Elem()
	}
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("目标必须是指针")
	}
	dstValue = dstValue.Elem()

	if srcValue.Type() != dstValue.Type() {
		return fmt.Errorf("源和目标类型不匹配")
	}

	dstValue.Set(srcValue)
	return nil
}

// === 切片工具 ===

// Contains 检查切片是否包含指定元素
func Contains(slice []interface{}, item interface{}) bool {
	for _, s := range slice {
		if reflect.DeepEqual(s, item) {
			return true
		}
	}
	return false
}

// ContainsString 检查字符串切片是否包含指定字符串
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsInt 检查整数切片是否包含指定整数
func ContainsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// UniqueStrings 去重字符串切片
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range slice {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// UniqueInts 去重整数切片
func UniqueInts(slice []int) []int {
	seen := make(map[int]bool)
	var result []int

	for _, num := range slice {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}

	return result
}

// === Map工具 ===

// MergeMaps 合并多个map
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}

// GetMapKeys 获取map的所有键
func GetMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetMapValues 获取map的所有值
func GetMapValues(m map[string]interface{}) []interface{} {
	values := make([]interface{}, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// === 条件工具 ===

// If 三元运算符
func If(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

// IfString 字符串三元运算符
func IfString(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

// IfInt 整数三元运算符
func IfInt(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}

// Coalesce 返回第一个非空值
func Coalesce(values ...interface{}) interface{} {
	for _, value := range values {
		if !IsNil(value) && !IsZero(value) {
			return value
		}
	}
	return nil
}

// CoalesceString 返回第一个非空字符串
func CoalesceString(values ...string) string {
	for _, value := range values {
		if IsNotEmpty(value) {
			return value
		}
	}
	return ""
}

// === 错误处理工具 ===

// Must 如果有错误则panic
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustReturn 如果有错误则panic，否则返回值
func MustReturn(value interface{}, err error) interface{} {
	Must(err)
	return value
}

// IgnoreError 忽略错误，只返回值
func IgnoreError(value interface{}, err error) interface{} {
	return value
}

// === 并发工具 ===

// Retry 重试执行函数
func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}

// RetryWithBackoff 带退避的重试
func RetryWithBackoff(attempts int, initialDelay time.Duration, backoffFactor float64, fn func() error) error {
	var err error
	delay := initialDelay

	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * backoffFactor)
		}
	}
	return err
}

// === 工具集合 ===

// Utils 工具集合
type Utils struct{}

// NewUtils 创建工具实例
func NewUtils() *Utils {
	return &Utils{}
}

// String 字符串工具方法
func (u *Utils) String() *StringUtils {
	return &StringUtils{}
}

// Number 数字工具方法
func (u *Utils) Number() *NumberUtils {
	return &NumberUtils{}
}

// Time 时间工具方法
func (u *Utils) Time() *TimeUtils {
	return &TimeUtils{}
}

// File 文件工具方法
func (u *Utils) File() *FileUtils {
	return &FileUtils{}
}

// Crypto 加密工具方法
func (u *Utils) Crypto() *CryptoUtils {
	return &CryptoUtils{}
}

// Validation 验证工具方法
func (u *Utils) Validation() *ValidationUtils {
	return &ValidationUtils{}
}

// JSON JSON工具方法
func (u *Utils) JSON() *JSONUtils {
	return &JSONUtils{}
}

// Reflect 反射工具方法
func (u *Utils) Reflect() *ReflectUtils {
	return &ReflectUtils{}
}

// 具体工具类型定义
type StringUtils struct{}
type NumberUtils struct{}
type TimeUtils struct{}
type FileUtils struct{}
type CryptoUtils struct{}
type ValidationUtils struct{}
type JSONUtils struct{}
type ReflectUtils struct{}

// StringUtils 方法
func (s *StringUtils) IsEmpty(str string) bool                { return IsEmpty(str) }
func (s *StringUtils) IsNotEmpty(str string) bool             { return IsNotEmpty(str) }
func (s *StringUtils) Truncate(str string, length int) string { return TruncateString(str, length) }
func (s *StringUtils) CamelToSnake(str string) string         { return CamelToSnake(str) }
func (s *StringUtils) SnakeToCamel(str string) string         { return SnakeToCamel(str) }
func (s *StringUtils) Mask(str string, start, end int, mask rune) string {
	return MaskString(str, start, end, mask)
}
func (s *StringUtils) Random(length int) string { return RandomString(length) }

// NumberUtils 方法
func (n *NumberUtils) SafeParseInt(str string, defaultValue int) int {
	return SafeParseInt(str, defaultValue)
}
func (n *NumberUtils) SafeParseFloat(str string, defaultValue float64) float64 {
	return SafeParseFloat(str, defaultValue)
}
func (n *NumberUtils) InRange(value, min, max int) bool           { return InRange(value, min, max) }
func (n *NumberUtils) ClampInt(value, min, max int) int           { return ClampInt(value, min, max) }
func (n *NumberUtils) ClampFloat(value, min, max float64) float64 { return ClampFloat(value, min, max) }

// TimeUtils 方法
func (t *TimeUtils) FormatDuration(duration time.Duration) string { return FormatDuration(duration) }
func (t *TimeUtils) ParseWithFormats(timeStr string, formats []string) (time.Time, error) {
	return ParseTimeWithFormats(timeStr, formats)
}
func (t *TimeUtils) IsBusinessDay(time time.Time) bool        { return IsBusinessDay(time) }
func (t *TimeUtils) NextBusinessDay(time time.Time) time.Time { return NextBusinessDay(time) }

// FileUtils 方法
func (f *FileUtils) EnsureDir(path string) error         { return EnsureDir(path) }
func (f *FileUtils) GetSize(path string) (int64, error)  { return GetFileSize(path) }
func (f *FileUtils) IsValidPath(path string) bool        { return IsValidPath(path) }
func (f *FileUtils) GetExtension(filename string) string { return GetFileExtension(filename) }

// CryptoUtils 方法
func (c *CryptoUtils) MD5(data string) string          { return MD5Hash(data) }
func (c *CryptoUtils) SHA256(data string) string       { return SHA256Hash(data) }
func (c *CryptoUtils) GenerateToken(length int) string { return GenerateSecureToken(length) }

// ValidationUtils 方法
func (v *ValidationUtils) IsEmail(email string) bool             { return IsValidEmail(email) }
func (v *ValidationUtils) IsURL(url string) bool                 { return IsValidURL(url) }
func (v *ValidationUtils) IsPhone(phone string) bool             { return IsValidPhone(phone) }
func (v *ValidationUtils) IsStrongPassword(password string) bool { return IsStrongPassword(password) }

// JSONUtils 方法
func (j *JSONUtils) Pretty(data interface{}) (string, error)              { return PrettyJSON(data) }
func (j *JSONUtils) Compact(data interface{}) (string, error)             { return CompactJSON(data) }
func (j *JSONUtils) IsValid(jsonStr string) bool                          { return IsValidJSON(jsonStr) }
func (j *JSONUtils) ToMap(jsonStr string) (map[string]interface{}, error) { return JSONToMap(jsonStr) }

// ReflectUtils 方法
func (r *ReflectUtils) IsNil(i interface{}) bool           { return IsNil(i) }
func (r *ReflectUtils) IsZero(i interface{}) bool          { return IsZero(i) }
func (r *ReflectUtils) GetFields(obj interface{}) []string { return GetStructFields(obj) }
func (r *ReflectUtils) Copy(src, dst interface{}) error    { return CopyStruct(src, dst) }

// 全局工具实例
var Default = NewUtils()

// 全局便捷函数
func String() *StringUtils         { return Default.String() }
func Number() *NumberUtils         { return Default.Number() }
func Time() *TimeUtils             { return Default.Time() }
func File() *FileUtils             { return Default.File() }
func Crypto() *CryptoUtils         { return Default.Crypto() }
func Validation() *ValidationUtils { return Default.Validation() }
func JSON() *JSONUtils             { return Default.JSON() }
func Reflect() *ReflectUtils       { return Default.Reflect() }
