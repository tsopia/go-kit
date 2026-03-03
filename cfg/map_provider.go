package cfg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
)

// mapProvider 是基于 map 的 Provider 实现
// 主要用于测试，不受全局环境变量影响
type mapProvider struct {
	data map[string]any
}

// newMapProvider 从 map 创建 Provider
// 主要用于测试场景，提供一个独立的配置实例
func newMapProvider(m map[string]any) Provider {
	// 深拷贝 map，避免外部修改影响内部状态
	data := make(map[string]any, len(m))
	for k, v := range m {
		data[k] = deepCopyValue(v)
	}
	return &mapProvider{data: data}
}

// deepCopyValue 深拷贝一个值
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		newMap := make(map[string]any, len(val))
		for k, v := range val {
			newMap[k] = deepCopyValue(v)
		}
		return newMap
	case map[any]any:
		newMap := make(map[string]any, len(val))
		for k, v := range val {
			if keyStr, ok := k.(string); ok {
				newMap[keyStr] = deepCopyValue(v)
			}
		}
		return newMap
	case []any:
		newSlice := make([]any, len(val))
		for i, v := range val {
			newSlice[i] = deepCopyValue(v)
		}
		return newSlice
	default:
		return v
	}
}

// Get 获取原始值
func (p *mapProvider) Get(key string) any {
	return p.getValue(key)
}

// GetString 获取字符串值
func (p *mapProvider) GetString(key string, defaultValue ...string) (string, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return "", err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return "", ErrNotFound
	}
	val := p.getValue(key)
	str, err := cast.ToStringE(val)
	if err != nil {
		return "", fmt.Errorf("%w: key %q expects string", ErrTypeMismatch, key)
	}
	return str, nil
}

// GetBool 获取布尔值
func (p *mapProvider) GetBool(key string, defaultValue ...bool) (bool, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return false, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return false, ErrNotFound
	}
	val := p.getValue(key)
	b, err := cast.ToBoolE(val)
	if err != nil {
		return false, fmt.Errorf("%w: key %q expects bool", ErrTypeMismatch, key)
	}
	return b, nil
}

// GetInt 获取整数值
func (p *mapProvider) GetInt(key string, defaultValue ...int) (int, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToIntE(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects int", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetInt32 获取 int32 值
func (p *mapProvider) GetInt32(key string, defaultValue ...int32) (int32, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToInt32E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects int32", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetInt64 获取 int64 值
func (p *mapProvider) GetInt64(key string, defaultValue ...int64) (int64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToInt64E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects int64", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetUint 获取无符号整数值
func (p *mapProvider) GetUint(key string, defaultValue ...uint) (uint, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToUintE(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects uint", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetUint8 获取 uint8 值
func (p *mapProvider) GetUint8(key string, defaultValue ...uint8) (uint8, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToUint8E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects uint8", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetUint16 获取 uint16 值
func (p *mapProvider) GetUint16(key string, defaultValue ...uint16) (uint16, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToUint16E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects uint16", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetUint32 获取 uint32 值
func (p *mapProvider) GetUint32(key string, defaultValue ...uint32) (uint32, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToUint32E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects uint32", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetUint64 获取 uint64 值
func (p *mapProvider) GetUint64(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	i, err := cast.ToUint64E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects uint64", ErrTypeMismatch, key)
	}
	return i, nil
}

// GetFloat64 获取 float64 值
func (p *mapProvider) GetFloat64(key string, defaultValue ...float64) (float64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	f, err := cast.ToFloat64E(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects float64", ErrTypeMismatch, key)
	}
	return f, nil
}

// GetDuration 获取时间间隔值
func (p *mapProvider) GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	d, err := cast.ToDurationE(val)
	if err != nil {
		return 0, fmt.Errorf("%w: key %q expects duration", ErrTypeMismatch, key)
	}
	return d, nil
}

// GetTime 获取时间值
func (p *mapProvider) GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return time.Time{}, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return time.Time{}, ErrNotFound
	}
	val := p.getValue(key)
	t, err := cast.ToTimeE(val)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: key %q expects time", ErrTypeMismatch, key)
	}
	return t, nil
}

// GetSizeInBytes 获取字节大小
func (p *mapProvider) GetSizeInBytes(key string, defaultValue ...uint64) (uint64, error) {
	if err := checkDefaultCount(key, defaultValue); err != nil {
		return 0, err
	}
	if !p.Exists(key) {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 0, ErrNotFound
	}
	val := p.getValue(key)
	switch v := val.(type) {
	case string:
		return parseSizeInBytes(v), nil
	case int:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float64:
		return uint64(v), nil
	default:
		return 0, fmt.Errorf("%w: key %q expects size string or number", ErrTypeMismatch, key)
	}
}

// parseSizeInBytes 解析字节大小字符串
func parseSizeInBytes(sizeStr string) uint64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	sizeStrUpper := strings.ToUpper(sizeStr)

	var numStr string
	var unit string
	for i, c := range sizeStrUpper {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = strings.TrimSpace(sizeStrUpper[i:])
			break
		}
	}

	if numStr == "" {
		return 0
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	var multiplier uint64 = 1
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "PB", "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return uint64(num * float64(multiplier))
}

// GetStringSlice 获取字符串切片
func (p *mapProvider) GetStringSlice(key string) ([]string, error) {
	if !p.Exists(key) {
		return nil, ErrNotFound
	}
	val := p.getValue(key)
	slice, err := cast.ToStringSliceE(val)
	if err != nil {
		return nil, fmt.Errorf("%w: key %q expects string slice", ErrTypeMismatch, key)
	}
	return slice, nil
}

// GetIntSlice 获取整数切片
func (p *mapProvider) GetIntSlice(key string) ([]int, error) {
	if !p.Exists(key) {
		return nil, ErrNotFound
	}
	val := p.getValue(key)
	switch v := val.(type) {
	case []int:
		return v, nil
	case []any:
		result := make([]int, 0, len(v))
		for _, item := range v {
			if i, err := cast.ToIntE(item); err == nil {
				result = append(result, i)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%w: key %q expects int slice", ErrTypeMismatch, key)
	}
}

// GetStringMap 获取字符串到任意类型的映射
func (p *mapProvider) GetStringMap(key string) (map[string]any, error) {
	if !p.Exists(key) {
		return nil, ErrNotFound
	}
	val := p.getValue(key)
	m, err := cast.ToStringMapE(val)
	if err != nil {
		return nil, fmt.Errorf("%w: key %q expects string map", ErrTypeMismatch, key)
	}
	return m, nil
}

// Exists 检查键是否存在
func (p *mapProvider) Exists(key string) bool {
	return p.getValue(key) != nil
}

// Unmarshal 将配置反序列化到目标结构体
func (p *mapProvider) Unmarshal(dst any) error {
	if dst == nil {
		return fmt.Errorf("unmarshal config: destination cannot be nil")
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           dst,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return fmt.Errorf("unmarshal config: create decoder: %w", err)
	}
	if err := decoder.Decode(p.data); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}

// Sub 获取子配置
func (p *mapProvider) Sub(key string) Provider {
	val := p.getValue(key)
	if val == nil {
		return emptyProviderInstance
	}

	switch m := val.(type) {
	case map[string]any:
		return &mapProvider{data: m}
	case map[any]any:
		strMap := make(map[string]any, len(m))
		for k, v := range m {
			if keyStr, ok := k.(string); ok {
				strMap[keyStr] = v
			}
		}
		return &mapProvider{data: strMap}
	default:
		return emptyProviderInstance
	}
}

// getValue 通过点分隔的路径获取值
func (p *mapProvider) getValue(key string) any {
	if key == "" {
		return nil
	}

	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return nil
	}

	val, exists := p.data[parts[0]]
	if !exists {
		return nil
	}

	for _, part := range parts[1:] {
		switch m := val.(type) {
		case map[string]any:
			val, exists = m[part]
			if !exists {
				return nil
			}
		case map[any]any:
			found := false
			for k, v := range m {
				if keyStr, ok := k.(string); ok && keyStr == part {
					val = v
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		default:
			return nil
		}
	}

	return val
}
