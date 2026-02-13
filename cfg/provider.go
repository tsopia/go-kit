package cfg

import "time"

// Provider 定义配置提供者的接口
type Provider interface {
	// Get 获取原始值
	Get(key string) any

	// 基础类型获取，支持可选默认值，返回 (值, 错误)
	// 不传默认值时，key 不存在返回 ErrNotFound
	// 传默认值时，key 不存在返回默认值
	GetString(key string, defaultValue ...string) (string, error)
	GetBool(key string, defaultValue ...bool) (bool, error)
	GetInt(key string, defaultValue ...int) (int, error)
	GetInt32(key string, defaultValue ...int32) (int32, error)
	GetInt64(key string, defaultValue ...int64) (int64, error)
	GetUint(key string, defaultValue ...uint) (uint, error)
	GetUint8(key string, defaultValue ...uint8) (uint8, error)
	GetUint16(key string, defaultValue ...uint16) (uint16, error)
	GetUint32(key string, defaultValue ...uint32) (uint32, error)
	GetUint64(key string, defaultValue ...uint64) (uint64, error)
	GetFloat64(key string, defaultValue ...float64) (float64, error)
	GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error)
	GetTime(key string, defaultValue ...time.Time) (time.Time, error)

	// 特殊类型
	GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error)

	// 切片类型（不支持默认值参数，因为 API 难用，通过 Exists 判断）
	GetStringSlice(key string) ([]string, error)
	GetIntSlice(key string) ([]int, error)

	// Map 类型
	GetStringMap(key string) (map[string]any, error)

	// 检查与操作
	Exists(key string) bool
	Unmarshal(dst any) error
	Sub(key string) Provider
}
