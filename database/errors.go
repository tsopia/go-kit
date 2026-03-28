package database

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrMissingClient     = errors.New("database: client not configured")
	ErrMissingDriver     = errors.New("数据库驱动不能为空")
	ErrUnsupportedDriver = errors.New("不支持的数据库驱动")
	ErrMissingHost       = errors.New("数据库主机不能为空")
	ErrInvalidPort       = errors.New("数据库端口无效")
	ErrMissingUsername   = errors.New("数据库用户名不能为空")
	ErrMissingDatabase   = errors.New("数据库名不能为空")
	ErrMissingDBPath     = errors.New("SQLite数据库路径不能为空")
	ErrInvalidLogLevel   = errors.New("无效的日志级别")
	ErrInvalidCharset    = errors.New("无效的字符集")
	ErrInvalidSSLMode    = errors.New("无效的SSL模式")
	ErrInvalidConnPool   = errors.New("连接池配置无效")
	ErrInvalidTimeout    = errors.New("超时配置无效")
	ErrConnectionFailed  = errors.New("数据库连接失败")
	ErrTransactionFailed = errors.New("事务执行失败")
	ErrQueryFailed       = errors.New("查询执行失败")
	ErrMigrationFailed   = errors.New("数据库迁移失败")
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeConnection ErrorType = iota
	ErrorTypeValidation
	ErrorTypeQuery
	ErrorTypeTransaction
	ErrorTypeMigration
)

// DatabaseError 数据库错误结构
type DatabaseError struct {
	Type      ErrorType
	Operation string
	Err       error
	Context   map[string]interface{}
}

// Error 实现error接口
func (e *DatabaseError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("数据库错误 [%s]: %v (上下文: %v)", e.Operation, e.Err, e.Context)
	}
	return fmt.Sprintf("数据库错误 [%s]: %v", e.Operation, e.Err)
}

// Unwrap 支持errors.Unwrap
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// Is 支持errors.Is
func (e *DatabaseError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewDatabaseError 创建数据库错误
func NewDatabaseError(errorType ErrorType, operation string, err error) *DatabaseError {
	return &DatabaseError{
		Type:      errorType,
		Operation: operation,
		Err:       err,
		Context:   make(map[string]interface{}),
	}
}

// WithContext 添加错误上下文
func (e *DatabaseError) WithContext(key string, value interface{}) *DatabaseError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Type == ErrorTypeConnection
	}

	return errors.Is(err, ErrConnectionFailed)
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Type == ErrorTypeValidation
	}

	// 检查是否为我们定义的验证错误
	return errors.Is(err, ErrMissingDriver) ||
		errors.Is(err, ErrUnsupportedDriver) ||
		errors.Is(err, ErrMissingHost) ||
		errors.Is(err, ErrInvalidPort) ||
		errors.Is(err, ErrMissingUsername) ||
		errors.Is(err, ErrMissingDatabase) ||
		errors.Is(err, ErrMissingDBPath) ||
		errors.Is(err, ErrInvalidLogLevel) ||
		errors.Is(err, ErrInvalidCharset) ||
		errors.Is(err, ErrInvalidSSLMode) ||
		errors.Is(err, ErrInvalidConnPool) ||
		errors.Is(err, ErrInvalidTimeout)
}
