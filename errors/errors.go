package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode 错误码设计
type ErrorCode struct {
	Code           int    `json:"code"`           // 数字错误码，用于API响应
	Name           string `json:"name,omitempty"` // 字符串名称，用于调试和日志
	DefaultMessage string `json:"-"`              // 默认消息，不序列化到JSON
}

// String 返回字符串表示
func (ec ErrorCode) String() string {
	if ec.Name != "" {
		return ec.Name
	}
	return fmt.Sprintf("ERROR_%d", ec.Code)
}

// MarshalJSON 自定义JSON序列化
func (ec ErrorCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(ec.Code)
}

// UnmarshalJSON 自定义JSON反序列化
func (ec *ErrorCode) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &ec.Code)
}

// Equal 比较两个错误码
func (ec ErrorCode) Equal(other ErrorCode) bool {
	return ec.Code == other.Code
}

// GetDefaultMessage 获取默认消息
func (ec ErrorCode) GetDefaultMessage() string {
	if ec.DefaultMessage != "" {
		return ec.DefaultMessage
	}
	return ec.Name
}

// 预定义错误码变量
var (
	// 系统级错误码 (1000-1999)
	CodeInternalServer = ErrorCode{
		Code:           1000,
		Name:           "INTERNAL_SERVER_ERROR",
		DefaultMessage: "内部服务器错误",
	}
	CodeInvalidParam = ErrorCode{
		Code:           1001,
		Name:           "INVALID_PARAM",
		DefaultMessage: "参数无效",
	}
	CodeNotFound = ErrorCode{
		Code:           1002,
		Name:           "NOT_FOUND",
		DefaultMessage: "资源不存在",
	}
	CodeUnauthorized = ErrorCode{
		Code:           1003,
		Name:           "UNAUTHORIZED",
		DefaultMessage: "未授权",
	}
	CodeForbidden = ErrorCode{
		Code:           1004,
		Name:           "FORBIDDEN",
		DefaultMessage: "访问被禁止",
	}
	CodeConflict = ErrorCode{
		Code:           1005,
		Name:           "CONFLICT",
		DefaultMessage: "资源冲突",
	}
	CodeTooManyRequests = ErrorCode{
		Code:           1006,
		Name:           "TOO_MANY_REQUESTS",
		DefaultMessage: "请求过多",
	}

	// 业务级错误码 (2000-2999)
	CodeUserNotFound = ErrorCode{
		Code:           2000,
		Name:           "USER_NOT_FOUND",
		DefaultMessage: "用户不存在",
	}
	CodeUserExists = ErrorCode{
		Code:           2001,
		Name:           "USER_EXISTS",
		DefaultMessage: "用户已存在",
	}
	CodeInvalidPassword = ErrorCode{
		Code:           2002,
		Name:           "INVALID_PASSWORD",
		DefaultMessage: "密码无效",
	}
	CodeTokenExpired = ErrorCode{
		Code:           2003,
		Name:           "TOKEN_EXPIRED",
		DefaultMessage: "令牌已过期",
	}
	CodeTokenInvalid = ErrorCode{
		Code:           2004,
		Name:           "TOKEN_INVALID",
		DefaultMessage: "令牌无效",
	}

	// 数据库错误码 (3000-3999)
	CodeDatabaseError = ErrorCode{
		Code:           3000,
		Name:           "DATABASE_ERROR",
		DefaultMessage: "数据库错误",
	}
	CodeRecordNotFound = ErrorCode{
		Code:           3001,
		Name:           "RECORD_NOT_FOUND",
		DefaultMessage: "记录不存在",
	}
	CodeDuplicateKey = ErrorCode{
		Code:           3002,
		Name:           "DUPLICATE_KEY",
		DefaultMessage: "数据重复",
	}
	CodeForeignKeyViolation = ErrorCode{
		Code:           3003,
		Name:           "FOREIGN_KEY_VIOLATION",
		DefaultMessage: "外键约束违反",
	}

	// 外部服务错误码 (4000-4999)
	CodeExternalServiceError = ErrorCode{
		Code:           4000,
		Name:           "EXTERNAL_SERVICE_ERROR",
		DefaultMessage: "外部服务错误",
	}
	CodeNetworkError = ErrorCode{
		Code:           4001,
		Name:           "NETWORK_ERROR",
		DefaultMessage: "网络错误",
	}
	CodeTimeoutError = ErrorCode{
		Code:           4002,
		Name:           "TIMEOUT_ERROR",
		DefaultMessage: "请求超时",
	}
)

// Error 自定义错误结构
type Error struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details string                 `json:"details,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
	Stack   string                 `json:"stack,omitempty"`
	Cause   error                  `json:"-"`
}

// Error 实现error接口
func (e *Error) Error() string {
	message := e.GetMessage()
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code.Name, message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code.Name, message)
}

// Unwrap 返回原始错误
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithStack 添加堆栈信息
func (e *Error) WithStack() *Error {
	if e.Stack == "" {
		e.Stack = getStack()
	}
	return e
}

// WithContext 添加上下文信息
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithDetails 添加详细信息
func (e *Error) WithDetails(details string) *Error {
	e.Details = details
	return e
}

// WithMessage 设置自定义消息
func (e *Error) WithMessage(message string) *Error {
	e.Message = message
	return e
}

// New 创建新的错误
func New(code ErrorCode, message ...string) *Error {
	err := &Error{Code: code}
	if len(message) > 0 && message[0] != "" {
		err.Message = message[0]
	}
	return err
}

// NewWithDetails 创建带详细信息的错误
func NewWithDetails(code ErrorCode, message string, details string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message ...string) *Error {
	wrapped := &Error{
		Code:  code,
		Cause: err,
	}
	if len(message) > 0 && message[0] != "" {
		wrapped.Message = message[0]
	}
	return wrapped
}

// WrapWithDetails 包装现有错误并添加详细信息
func WrapWithDetails(err error, code ErrorCode, message string, details string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   err,
	}
}

// GetCode 获取错误码
func GetCode(err error) ErrorCode {
	if err == nil {
		return ErrorCode{} // Return an empty ErrorCode
	}

	if e, ok := err.(*Error); ok {
		return e.Code
	}

	// 递归检查包装的错误
	if wrapped := Unwrap(err); wrapped != nil {
		return GetCode(wrapped)
	}

	return CodeInternalServer
}

// GetMessage 获取实际的错误消息
func (e *Error) GetMessage() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code.GetDefaultMessage()
}

// Is 检查错误类型
func Is(err error, code ErrorCode) bool {
	return GetCode(err).Equal(code)
}

// IsInternalServer 检查是否为内部服务器错误
func IsInternalServer(err error) bool {
	return Is(err, CodeInternalServer)
}

// IsInvalidParam 检查是否为无效参数错误
func IsInvalidParam(err error) bool {
	return Is(err, CodeInvalidParam)
}

// IsNotFound 检查是否为未找到错误
func IsNotFound(err error) bool {
	return Is(err, CodeNotFound)
}

// IsUnauthorized 检查是否为未授权错误
func IsUnauthorized(err error) bool {
	return Is(err, CodeUnauthorized)
}

// IsForbidden 检查是否为禁止访问错误
func IsForbidden(err error) bool {
	return Is(err, CodeForbidden)
}

// IsConflict 检查是否为冲突错误
func IsConflict(err error) bool {
	return Is(err, CodeConflict)
}

// IsTooManyRequests 检查是否为请求过多错误
func IsTooManyRequests(err error) bool {
	return Is(err, CodeTooManyRequests)
}

// IsDatabaseError 检查是否为数据库错误
func IsDatabaseError(err error) bool {
	code := GetCode(err)
	return code.Equal(CodeDatabaseError) || code.Equal(CodeRecordNotFound) ||
		code.Equal(CodeDuplicateKey) || code.Equal(CodeForeignKeyViolation)
}

// IsExternalServiceError 检查是否为外部服务错误
func IsExternalServiceError(err error) bool {
	code := GetCode(err)
	return code.Equal(CodeExternalServiceError) || code.Equal(CodeNetworkError) ||
		code.Equal(CodeTimeoutError)
}

// Unwrap 解包错误
func Unwrap(err error) error {
	if e, ok := err.(*Error); ok {
		return e.Cause
	}
	return nil
}

// 预定义错误码映射 - 避免每次函数调用时重复构建
var codeMap = map[string]ErrorCode{
	"INTERNAL_SERVER_ERROR":  CodeInternalServer,
	"INVALID_PARAM":          CodeInvalidParam,
	"NOT_FOUND":              CodeNotFound,
	"UNAUTHORIZED":           CodeUnauthorized,
	"FORBIDDEN":              CodeForbidden,
	"CONFLICT":               CodeConflict,
	"TOO_MANY_REQUESTS":      CodeTooManyRequests,
	"USER_NOT_FOUND":         CodeUserNotFound,
	"USER_EXISTS":            CodeUserExists,
	"INVALID_PASSWORD":       CodeInvalidPassword,
	"TOKEN_EXPIRED":          CodeTokenExpired,
	"TOKEN_INVALID":          CodeTokenInvalid,
	"DATABASE_ERROR":         CodeDatabaseError,
	"RECORD_NOT_FOUND":       CodeRecordNotFound,
	"DUPLICATE_KEY":          CodeDuplicateKey,
	"FOREIGN_KEY_VIOLATION":  CodeForeignKeyViolation,
	"EXTERNAL_SERVICE_ERROR": CodeExternalServiceError,
	"NETWORK_ERROR":          CodeNetworkError,
	"TIMEOUT_ERROR":          CodeTimeoutError,
}

// StringToCode 根据字符串名称查找错误码
func StringToCode(s string) ErrorCode {
	if code, exists := codeMap[strings.ToUpper(s)]; exists {
		return code
	}

	// 如果没有找到，返回一个通用的错误码
	return ErrorCode{Code: 9999, Name: strings.ToUpper(s)}
}

// StringToCodeWithFound 根据字符串名称查找错误码，并返回是否找到
func StringToCodeWithFound(s string) (ErrorCode, bool) {
	if code, exists := codeMap[strings.ToUpper(s)]; exists {
		return code, true
	}
	return ErrorCode{Code: 9999, Name: strings.ToUpper(s)}, false
}

// NewErrorCode 创建自定义错误码
func NewErrorCode(code int, name string, defaultMessage ...string) ErrorCode {
	ec := ErrorCode{Code: code, Name: name}
	if len(defaultMessage) > 0 && defaultMessage[0] != "" {
		ec.DefaultMessage = defaultMessage[0]
	}
	return ec
}

// === 便利函数 - 只保留最常用的 ===

// InvalidParam 创建无效参数错误
func InvalidParam(message ...string) *Error {
	return New(CodeInvalidParam, message...)
}

// NotFound 创建未找到错误
func NotFound(message ...string) *Error {
	return New(CodeNotFound, message...)
}

// Internal 创建内部服务器错误
func Internal(message ...string) *Error {
	return New(CodeInternalServer, message...)
}

// === 辅助函数 ===

// getStack 获取堆栈信息
func getStack() string {
	return getStackWithBuffer(make([]byte, 4096))
}

// getStackWithBuffer 使用指定缓冲区获取堆栈信息（用于测试）
func getStackWithBuffer(buf []byte) string {
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// getStackOptimized 优化的堆栈获取（用于高频场景）
func getStackOptimized() string {
	const maxFrames = 32
	var pcs [maxFrames]uintptr
	n := runtime.Callers(3, pcs[:])

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		buf.WriteString(fmt.Sprintf("\n\t%s:%d", frame.File, frame.Line))
		if !more {
			break
		}
	}

	return buf.String()
}

// === 扩展的错误查询函数 ===

// GetContext 获取错误上下文
func GetContext(err error) map[string]interface{} {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e.Context
	}
	return nil
}

// GetStack 获取堆栈信息
func GetStack(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*Error); ok {
		return e.Stack
	}
	return ""
}

// === 标准库兼容性 ===

// 重新导出标准库函数
var (
	As             = stderrors.As
	StdIs          = stderrors.Is // 重命名避免冲突
	Join           = stderrors.Join
	StdUnwrap      = stderrors.Unwrap // 重命名避免冲突
	ErrUnsupported = stderrors.ErrUnsupported
)

// === 格式化支持 ===

// Newf 创建格式化的错误
func Newf(code ErrorCode, format string, args ...interface{}) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrapf 包装现有错误并格式化消息
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *Error {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}
