package cfg

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// viperProvider 是基于 viper 的 Provider 实现
type viperProvider struct {
	v *viper.Viper
}

// newViperProvider 创建一个新的 viperProvider 实例
func newViperProvider(v *viper.Viper) *viperProvider {
	return &viperProvider{v: v}
}

// Get 获取原始值
func (p *viperProvider) Get(key string) any {
	return p.v.Get(key)
}

// checkDefaultCount 检查默认值参数个数
func checkDefaultCount[T any](key string, defaults []T) error {
	if len(defaults) > 1 {
		return fmt.Errorf("%w: key %q expects at most one default value, got %d", ErrTypeMismatch, key, len(defaults))
	}
	return nil
}

// GetString 获取字符串值
func (p *viperProvider) GetString(key string, defaultValue ...string) (string, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return "", err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return "", ErrNotFound
	}
	return p.v.GetString(key), nil
}

// GetBool 获取布尔值
func (p *viperProvider) GetBool(key string, defaultValue ...bool) (bool, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return false, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return false, ErrNotFound
	}
	return p.v.GetBool(key), nil
}

// GetInt 获取整数值
func (p *viperProvider) GetInt(key string, defaultValue ...int) (int, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetInt(key), nil
}

// GetInt32 获取 int32 值
func (p *viperProvider) GetInt32(key string, defaultValue ...int32) (int32, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetInt32(key), nil
}

// GetInt64 获取 int64 值
func (p *viperProvider) GetInt64(key string, defaultValue ...int64) (int64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetInt64(key), nil
}

// GetUint 获取无符号整数值
func (p *viperProvider) GetUint(key string, defaultValue ...uint) (uint, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetUint(key), nil
}

// GetUint8 获取 uint8 值
func (p *viperProvider) GetUint8(key string, defaultValue ...uint8) (uint8, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return uint8(p.v.GetUint(key)), nil
}

// GetUint16 获取 uint16 值
func (p *viperProvider) GetUint16(key string, defaultValue ...uint16) (uint16, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetUint16(key), nil
}

// GetUint32 获取 uint32 值
func (p *viperProvider) GetUint32(key string, defaultValue ...uint32) (uint32, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetUint32(key), nil
}

// GetUint64 获取 uint64 值
func (p *viperProvider) GetUint64(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetUint64(key), nil
}

// GetFloat64 获取 float64 值
func (p *viperProvider) GetFloat64(key string, defaultValue ...float64) (float64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetFloat64(key), nil
}

// GetDuration 获取时间间隔值
func (p *viperProvider) GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return p.v.GetDuration(key), nil
}

// GetTime 获取时间值
func (p *viperProvider) GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return time.Time{}, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return time.Time{}, ErrNotFound
	}
	return p.v.GetTime(key), nil
}

// GetSizeInBytes 获取字节大小（如 "1GB", "100MB"）
func (p *viperProvider) GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.v.IsSet(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	return parseSizeInBytes(p.v.GetString(key)), nil
}

// GetStringSlice 获取字符串切片
func (p *viperProvider) GetStringSlice(key string) ([]string, error) {
	if !p.v.IsSet(key) {
		return nil, ErrNotFound
	}
	return p.v.GetStringSlice(key), nil
}

// GetIntSlice 获取整数切片
func (p *viperProvider) GetIntSlice(key string) ([]int, error) {
	if !p.v.IsSet(key) {
		return nil, ErrNotFound
	}
	return p.v.GetIntSlice(key), nil
}

// GetStringMap 获取字符串到任意类型的映射
func (p *viperProvider) GetStringMap(key string) (map[string]any, error) {
	if !p.v.IsSet(key) {
		return nil, ErrNotFound
	}
	return p.v.GetStringMap(key), nil
}

// Exists 检查键是否存在
func (p *viperProvider) Exists(key string) bool {
	return p.v.IsSet(key)
}

// Unmarshal 将配置反序列化到目标结构体
func (p *viperProvider) Unmarshal(dst any) error {
	if err := p.v.Unmarshal(dst); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}

// Sub 获取子配置
// 如果子配置不存在，返回 emptyProvider（支持链式调用安全）
func (p *viperProvider) Sub(key string) Provider {
	sub := p.v.Sub(key)
	if sub == nil {
		return emptyProviderInstance
	}
	return newViperProvider(sub)
}
