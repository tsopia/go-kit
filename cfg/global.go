package cfg

import (
	"sync"
	"time"
)

var (
	global     Provider
	globalOnce sync.Once
	globalErr  error
	globalMu   sync.RWMutex
)

// Init 初始化全局配置实例
// 必须先调用 Init，否则包级便捷函数将返回零值
// 第一次调用后，后续调用将被忽略（返回第一次的错误）
//
// 使用示例：
//   cfg.Init()                           // 自动查找配置
//   cfg.Init("./config.yaml")            // 指定路径
func Init(path ...string) error {
	return InitWithPrefix("", path...)
}

// InitWithPrefix 使用环境变量前缀初始化全局配置
//
// 使用示例：
//   cfg.InitWithPrefix("MYAPP")                    // 自动查找，带前缀
//   cfg.InitWithPrefix("MYAPP", "./config.yaml")   // 指定路径，带前缀
func InitWithPrefix(prefix string, path ...string) error {
	globalOnce.Do(func() {
		global, globalErr = NewWithPrefix(prefix, path...)
	})
	return globalErr
}

// Default 获取全局 Provider 实例
// 如果 Init 尚未调用，返回空 Provider（emptyProvider）
func Default() Provider {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if global == nil {
		return emptyProviderInstance
	}
	return global
}

// IsInitialized 检查全局配置是否已初始化
func IsInitialized() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global != nil
}

// Get 从全局配置获取原始值
func Get(key string) any {
	return Default().Get(key)
}

// GetString 从全局配置获取字符串值
func GetString(key string, defaultValue ...string) (string, error) {
	return Default().GetString(key, defaultValue...)
}

// GetBool 从全局配置获取布尔值
func GetBool(key string, defaultValue ...bool) (bool, error) {
	return Default().GetBool(key, defaultValue...)
}

// GetInt 从全局配置获取整数值
func GetInt(key string, defaultValue ...int) (int, error) {
	return Default().GetInt(key, defaultValue...)
}

// GetInt32 从全局配置获取 int32 值
func GetInt32(key string, defaultValue ...int32) (int32, error) {
	return Default().GetInt32(key, defaultValue...)
}

// GetInt64 从全局配置获取 int64 值
func GetInt64(key string, defaultValue ...int64) (int64, error) {
	return Default().GetInt64(key, defaultValue...)
}

// GetUint 从全局配置获取无符号整数值
func GetUint(key string, defaultValue ...uint) (uint, error) {
	return Default().GetUint(key, defaultValue...)
}

// GetUint8 从全局配置获取 uint8 值
func GetUint8(key string, defaultValue ...uint8) (uint8, error) {
	return Default().GetUint8(key, defaultValue...)
}

// GetUint16 从全局配置获取 uint16 值
func GetUint16(key string, defaultValue ...uint16) (uint16, error) {
	return Default().GetUint16(key, defaultValue...)
}

// GetUint32 从全局配置获取 uint32 值
func GetUint32(key string, defaultValue ...uint32) (uint32, error) {
	return Default().GetUint32(key, defaultValue...)
}

// GetUint64 从全局配置获取 uint64 值
func GetUint64(key string, defaultValue ...uint64) (uint64, error) {
	return Default().GetUint64(key, defaultValue...)
}

// GetFloat64 从全局配置获取 float64 值
func GetFloat64(key string, defaultValue ...float64) (float64, error) {
	return Default().GetFloat64(key, defaultValue...)
}

// GetDuration 从全局配置获取时间间隔值
func GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	return Default().GetDuration(key, defaultValue...)
}

// GetTime 从全局配置获取时间值
func GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	return Default().GetTime(key, defaultValue...)
}

// GetSizeInBytes 从全局配置获取字节大小
func GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error) {
	return Default().GetSizeInBytes(key, defaultValue...)
}

// GetStringSlice 从全局配置获取字符串切片
func GetStringSlice(key string) ([]string, error) {
	return Default().GetStringSlice(key)
}

// GetIntSlice 从全局配置获取整数切片
func GetIntSlice(key string) ([]int, error) {
	return Default().GetIntSlice(key)
}

// GetStringMap 从全局配置获取字符串到任意类型的映射
func GetStringMap(key string) (map[string]any, error) {
	return Default().GetStringMap(key)
}

// Exists 检查全局配置中键是否存在
func Exists(key string) bool {
	return Default().Exists(key)
}

// Unmarshal 将全局配置反序列化到目标结构体
func Unmarshal(dst any) error {
	return Default().Unmarshal(dst)
}

// Sub 从全局配置获取子配置
func Sub(key string) Provider {
	return Default().Sub(key)
}
