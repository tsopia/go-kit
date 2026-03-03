// Package cfg provides configuration management functionality.
//
// Deprecated: This package is deprecated. Use github.com/tsopia/go-kit/cfg instead.
// The new cfg package located at the project root provides a cleaner API
// with better error handling and a unified constructor design.
package cfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const defaultConfigFileName = "config.yml"

var (
	ErrNotLoaded    = errors.New("cfg: configuration not loaded")
	ErrNotFound     = errors.New("cfg: key not found")
	ErrTypeMismatch = errors.New("cfg: type mismatch")
)

var (
	defaultManager *Manager
	defaultMutex   sync.RWMutex
)

// Manager 管理一个 viper 实例，提供类型安全的读取能力。
type Manager struct {
	v  *viper.Viper
	mu sync.RWMutex
}

// Load 创建配置管理器并可选地将配置绑定到结构体。
//
// 如果 target 不为 nil，会将配置解析到 target 指向的结构体中。
// 创建成功后会更新全局默认管理器，后续包级别的 Getter 将使用该实例。
func Load(target interface{}, filePath ...string) (*Manager, error) {
	v, err := createViperInstance(filePath...)
	if err != nil {
		return nil, err
	}

	if target != nil {
		if err := v.Unmarshal(target); err != nil {
			return nil, fmt.Errorf("cfg: unmarshal into struct failed: %w", err)
		}
	}

	manager := &Manager{v: v}
	defaultMutex.Lock()
	defaultManager = manager
	defaultMutex.Unlock()

	return manager, nil
}

// Default 返回已经初始化的默认管理器。
func Default() (*Manager, error) {
	defaultMutex.RLock()
	defer defaultMutex.RUnlock()

	if defaultManager == nil {
		return nil, ErrNotLoaded
	}
	return defaultManager, nil
}

// GetString 读取字符串配置，支持可选默认值。
func GetString(key string, defaultValue ...string) (string, error) {
	manager, err := Default()
	if err != nil {
		return "", err
	}
	return manager.GetString(key, defaultValue...)
}

// GetBool 读取布尔配置，支持可选默认值。
func GetBool(key string, defaultValue ...bool) (bool, error) {
	manager, err := Default()
	if err != nil {
		return false, err
	}
	return manager.GetBool(key, defaultValue...)
}

// GetInt 读取 int 配置，支持可选默认值。
func GetInt(key string, defaultValue ...int) (int, error) {
	manager, err := Default()
	if err != nil {
		return 0, err
	}
	return manager.GetInt(key, defaultValue...)
}

// GetInt64 读取 int64 配置，支持可选默认值。
func GetInt64(key string, defaultValue ...int64) (int64, error) {
	manager, err := Default()
	if err != nil {
		return 0, err
	}
	return manager.GetInt64(key, defaultValue...)
}

// GetFloat64 读取 float64 配置，支持可选默认值。
func GetFloat64(key string, defaultValue ...float64) (float64, error) {
	manager, err := Default()
	if err != nil {
		return 0, err
	}
	return manager.GetFloat64(key, defaultValue...)
}

// GetDuration 读取 time.Duration 配置，支持可选默认值。
func GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	manager, err := Default()
	if err != nil {
		return 0, err
	}
	return manager.GetDuration(key, defaultValue...)
}

// GetTime 读取 time.Time 配置，支持可选默认值。
func GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	manager, err := Default()
	if err != nil {
		return time.Time{}, err
	}
	return manager.GetTime(key, defaultValue...)
}

// GetStringSlice 读取字符串切片配置，支持可选默认值。
func GetStringSlice(key string, defaultValue ...[]string) ([]string, error) {
	manager, err := Default()
	if err != nil {
		return nil, err
	}
	return manager.GetStringSlice(key, defaultValue...)
}

// GetStringMap 读取字符串键的 map 配置，支持可选默认值。
func GetStringMap(key string, defaultValue ...map[string]interface{}) (map[string]interface{}, error) {
	manager, err := Default()
	if err != nil {
		return nil, err
	}
	return manager.GetStringMap(key, defaultValue...)
}

// GetStringMapString 读取 map[string]string 配置，支持可选默认值。
func GetStringMapString(key string, defaultValue ...map[string]string) (map[string]string, error) {
	manager, err := Default()
	if err != nil {
		return nil, err
	}
	return manager.GetStringMapString(key, defaultValue...)
}

// IsSet 判断配置项是否存在。
func IsSet(key string) (bool, error) {
	manager, err := Default()
	if err != nil {
		return false, err
	}
	return manager.IsSet(key)
}

// GetString 读取字符串配置，支持可选默认值。
func (m *Manager) GetString(key string, defaultValue ...string) (string, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return "", err
	}

	result, castErr := cast.ToStringE(value)
	if castErr != nil {
		return "", fmt.Errorf("%w: key %s expects string", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetBool 读取布尔配置，支持可选默认值。
func (m *Manager) GetBool(key string, defaultValue ...bool) (bool, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return false, err
	}

	result, castErr := cast.ToBoolE(value)
	if castErr != nil {
		return false, fmt.Errorf("%w: key %s expects bool", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetInt 读取 int 配置，支持可选默认值。
func (m *Manager) GetInt(key string, defaultValue ...int) (int, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return 0, err
	}

	result, castErr := cast.ToIntE(value)
	if castErr != nil {
		return 0, fmt.Errorf("%w: key %s expects int", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetInt64 读取 int64 配置，支持可选默认值。
func (m *Manager) GetInt64(key string, defaultValue ...int64) (int64, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return 0, err
	}

	result, castErr := cast.ToInt64E(value)
	if castErr != nil {
		return 0, fmt.Errorf("%w: key %s expects int64", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetFloat64 读取 float64 配置，支持可选默认值。
func (m *Manager) GetFloat64(key string, defaultValue ...float64) (float64, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return 0, err
	}

	result, castErr := cast.ToFloat64E(value)
	if castErr != nil {
		return 0, fmt.Errorf("%w: key %s expects float64", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetDuration 读取 time.Duration 配置，支持可选默认值。
func (m *Manager) GetDuration(key string, defaultValue ...time.Duration) (time.Duration, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return 0, err
	}

	result, castErr := cast.ToDurationE(value)
	if castErr != nil {
		return 0, fmt.Errorf("%w: key %s expects duration", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetTime 读取 time.Time 配置，支持可选默认值。
func (m *Manager) GetTime(key string, defaultValue ...time.Time) (time.Time, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return time.Time{}, err
	}

	result, castErr := cast.ToTimeE(value)
	if castErr != nil {
		return time.Time{}, fmt.Errorf("%w: key %s expects time", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetStringSlice 读取字符串切片配置，支持可选默认值。
func (m *Manager) GetStringSlice(key string, defaultValue ...[]string) ([]string, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return nil, err
	}

	result, castErr := cast.ToStringSliceE(value)
	if castErr != nil {
		return nil, fmt.Errorf("%w: key %s expects string slice", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetStringMap 读取字符串键的 map 配置，支持可选默认值。
func (m *Manager) GetStringMap(key string, defaultValue ...map[string]interface{}) (map[string]interface{}, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return nil, err
	}

	result, castErr := cast.ToStringMapE(value)
	if castErr != nil {
		return nil, fmt.Errorf("%w: key %s expects string map", ErrTypeMismatch, key)
	}
	return result, nil
}

// GetStringMapString 读取 map[string]string 配置，支持可选默认值。
func (m *Manager) GetStringMapString(key string, defaultValue ...map[string]string) (map[string]string, error) {
	value, err := m.value(key)
	if errors.Is(err, ErrNotFound) && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	if err != nil {
		return nil, err
	}

	result, castErr := cast.ToStringMapStringE(value)
	if castErr != nil {
		return nil, fmt.Errorf("%w: key %s expects string map", ErrTypeMismatch, key)
	}
	return result, nil
}

// IsSet 判断配置项是否存在。
func (m *Manager) IsSet(key string) (bool, error) {
	if m == nil || m.v == nil {
		return false, ErrNotLoaded
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.IsSet(key), nil
}

func (m *Manager) value(key string) (interface{}, error) {
	if m == nil || m.v == nil {
		return nil, ErrNotLoaded
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.v.IsSet(key) {
		return nil, ErrNotFound
	}

	return m.v.Get(key), nil
}

func createViperInstance(filePath ...string) (*viper.Viper, error) {
	v := viper.New()

	configPath, err := resolveConfigPath(filePath...)
	if err != nil {
		return nil, err
	}

	if ext := strings.TrimPrefix(filepath.Ext(configPath), "."); ext != "" {
		v.SetConfigType(ext)
	}
	v.SetConfigFile(configPath)

	if appName := os.Getenv("APP_NAME"); appName != "" {
		v.SetEnvPrefix(strings.ToUpper(appName))
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.AllowEmptyEnv(true)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("cfg: config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("cfg: read config failed: %w", err)
	}

	return v, nil
}

func resolveConfigPath(filePath ...string) (string, error) {
	for _, path := range filePath {
		if strings.TrimSpace(path) == "" {
			continue
		}

		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("cfg: config file not found: %s", path)
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cfg: cannot determine working directory: %w", err)
	}

	searchPaths := make([]string, 0, 9)
	seen := map[string]struct{}{}
	current := wd
	for i := 0; i < 3 && current != "" && current != "/"; i++ {
		candidates := []string{current, filepath.Join(current, "configs"), filepath.Join(current, "config")}
		for _, dir := range candidates {
			if _, ok := seen[dir]; ok {
				continue
			}
			seen[dir] = struct{}{}
			searchPaths = append(searchPaths, dir)
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	candidates := []string{defaultConfigFileName, ".env"}
	for _, candidate := range candidates {
		for _, dir := range searchPaths {
			candidatePath := filepath.Join(dir, candidate)
			if info, err := os.Stat(candidatePath); err == nil && !info.IsDir() {
				return candidatePath, nil
			}
		}
	}

	return "", fmt.Errorf("cfg: unable to locate config file")
}
