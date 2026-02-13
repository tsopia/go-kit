package cfg

import (
	"fmt"
	"time"
)

// emptyProvider 是空 Provider 实现（Null Object Pattern）
// 对所有方法返回零值或错误，Exists 返回 false
// 这种设计允许安全的链式调用，调用方无需检查 nil
type emptyProvider struct{}

// emptyProviderInstance 是 emptyProvider 的单例实例
var emptyProviderInstance = &emptyProvider{}

// checkDefaultCount 检查默认值参数个数
func checkDefaultCountEmpty[T any](key string, defaults []T) error {
	if len(defaults) > 1 {
		return fmt.Errorf("%w: key %q expects at most one default value, got %d", ErrTypeMismatch, key, len(defaults))
	}
	return nil
}

// Get 返回 nil
func (p *emptyProvider) Get(string) any {
	return nil
}

// GetString 返回零值或默认值
func (p *emptyProvider) GetString(key string, defaultValue ...string) (string, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return "", err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return "", ErrNotFound
}

// GetBool 返回零值或默认值
func (p *emptyProvider) GetBool(key string, defaultValue ...bool) (bool, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return false, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return false, ErrNotFound
}

// GetInt 返回零值或默认值
func (p *emptyProvider) GetInt(key string, defaultValue ...int) (int, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetInt32 返回零值或默认值
func (p *emptyProvider) GetInt32(key string, defaultValue ...int32) (int32, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetInt64 返回零值或默认值
func (p *emptyProvider) GetInt64(key string, defaultValue ...int64) (int64, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetUint 返回零值或默认值
func (p *emptyProvider) GetUint(key string, defaultValue ...uint) (uint, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetUint8 返回零值或默认值
func (p *emptyProvider) GetUint8(key string, defaultValue ...uint8) (uint8, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetUint16 返回零值或默认值
func (p *emptyProvider) GetUint16(key string, defaultValue ...uint16) (uint16, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetUint32 返回零值或默认值
func (p *emptyProvider) GetUint32(key string, defaultValue ...uint32) (uint32, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetUint64 返回零值或默认值
func (p *emptyProvider) GetUint64(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetFloat64 返回零值或默认值
func (p *emptyProvider) GetFloat64(key string, defaultValue ...float64) (float64, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetDuration 返回零值或默认值
func (p *emptyProvider) GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetTime 返回零值或默认值
func (p *emptyProvider) GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return time.Time{}, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return time.Time{}, ErrNotFound
}

// GetSizeInBytes 返回零值或默认值
func (p *emptyProvider) GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCountEmpty(key, defaultValue); err != nil {
		return 0, err
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return 0, ErrNotFound
}

// GetStringSlice 返回错误
func (p *emptyProvider) GetStringSlice(string) ([]string, error) {
	return nil, ErrNotFound
}

// GetIntSlice 返回错误
func (p *emptyProvider) GetIntSlice(string) ([]int, error) {
	return nil, ErrNotFound
}

// GetStringMap 返回错误
func (p *emptyProvider) GetStringMap(string) (map[string]any, error) {
	return nil, ErrNotFound
}

// Exists 返回 false
func (p *emptyProvider) Exists(string) bool {
	return false
}

// Unmarshal 返回 nil（无错误）
func (p *emptyProvider) Unmarshal(any) error {
	return nil
}

// Sub 返回自身，支持链式调用
// 例如：cfg.Sub("db").Sub("pool").GetInt("max_open") 不会 panic
func (p *emptyProvider) Sub(string) Provider {
	return p
}

// EmptyProvider 返回空 Provider 实例
// 用于测试或需要空实现的场景
func EmptyProvider() Provider {
	return emptyProviderInstance
}
