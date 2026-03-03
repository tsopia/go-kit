package cfg

import "errors"

// 预定义错误
var (
	// ErrNotFound 表示配置键不存在
	ErrNotFound = errors.New("cfg: key not found")
	// ErrTypeMismatch 表示类型不匹配
	ErrTypeMismatch = errors.New("cfg: type mismatch")
	// ErrNotInitialized 表示 Provider 未初始化
	ErrNotInitialized = errors.New("cfg: provider not initialized")
)
