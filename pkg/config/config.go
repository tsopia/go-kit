package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultConfigFileName = "config.yml"
)

var (
	// 全局viper实例，用于GetClient和便利函数
	globalViper   *viper.Viper
	globalMutex   sync.RWMutex
	isInitialized bool
)

// Cleanup 清理全局配置状态，释放相关资源
//
// 使用场景:
//   - ✅ 应用程序关闭时清理资源
//   - ✅ 在需要重新初始化配置时使用
//   - ✅ 微服务重启时的资源清理
//
// 示例:
//
//	defer config.Cleanup() // 应用退出时清理
//
//	// 重新初始化配置
//	config.Cleanup()
//	err := config.LoadConfig(&newCfg)
func Cleanup() {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalViper = nil
	isInitialized = false
}

// ResetGlobalState 重置全局配置状态（主要用于测试）
//
// 注意: 这是一个内部函数，生产代码请使用 Cleanup()
func ResetGlobalState() {
	Cleanup() // 复用公开的清理逻辑
}

// LoadConfig 加载配置文件并将其解析到提供的结构体中
//
// 参数:
//   - config: 指向要填充的配置结构体的指针
//   - filePath: 可选的自定义配置文件路径（如果为空，则使用默认路径 config.yml）
//
// 功能特性:
//   - 默认在项目根目录查找 config.yml 文件
//   - 支持自定义配置文件路径
//   - 环境变量优先于配置文件中的值
//   - 自动支持 YAML, JSON, TOML 等格式
//   - 环境变量名自动转换（如 app.name -> APP_NAME）
//   - 自动从 APP_NAME 环境变量获取前缀（如果设置了 APP_NAME=myapp，则环境变量为 MYAPP_*）
//
// 环境变量前缀规则:
//   - 如果设置了 APP_NAME 环境变量，则使用其值作为前缀
//   - 例如：APP_NAME=myapp 时，配置键 app.port 对应环境变量 MYAPP_APP_PORT
//   - 如果未设置 APP_NAME，则直接使用无前缀的环境变量（如 APP_PORT）
//   - 当配置文件和环境变量都有 app_name 时，环境变量优先级最高
//
// 返回:
//   - error: 如果加载或解析过程中出现错误
//
// 使用场景:
//   - ✅ 推荐用于大多数应用（约80%的使用场景）
//   - ✅ 适合配置结构相对固定的应用
//   - ✅ 提供类型安全和编译时检查
//
// 示例:
//
//	type AppConfig struct {
//	    App struct {
//	        Name string `mapstructure:"name"`
//	        Port int    `mapstructure:"port"`
//	    } `mapstructure:"app"`
//	}
//
//	// 基础用法
//	var config AppConfig
//	err := LoadConfig(&config) // 使用默认路径和自动前缀检测
//
//	// 自定义配置文件路径
//	err := LoadConfig(&config, "custom/config.yml")
//
//	// 环境变量示例:
//	// 不使用前缀: APP_NAME未设置时
//	//   export APP_PORT=8080
//	// 使用前缀: export APP_NAME=myapp 时
//	//   export MYAPP_APP_PORT=8080
func LoadConfig(config interface{}, filePath ...string) error {
	v, err := createViperInstanceWithError(filePath...)
	if err != nil {
		return err
	}

	// 解析配置到结构体
	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("解析配置到结构体失败: %w", err)
	}

	// 同时初始化全局viper实例供其他函数使用
	globalMutex.Lock()
	globalViper = v
	isInitialized = true
	globalMutex.Unlock()

	return nil
}

// GetClient 获取配置的viper客户端实例，提供完整的viper功能
//
// 返回已配置好的viper实例，支持：
//   - 动态配置获取
//   - 配置文件监听和热更新
//   - 复杂的配置操作和验证
//   - 所有viper原生功能
//
// 使用场景:
//   - ✅ 需要动态配置访问的高级应用
//   - ✅ 第三方库集成
//   - ✅ 配置热更新和监听
//   - ✅ 复杂的配置层级操作
//
// 返回:
//   - *viper.Viper: 配置好的viper实例
//   - error: 如果初始化过程中出现错误
//
// 注意: 如果之前没有调用过LoadConfig，会自动初始化配置
//
// 示例:
//
//	client, err := config.GetClient()
//	if err != nil {
//	    log.Fatal("获取配置客户端失败:", err)
//	}
//
//	// 动态获取配置
//	value := client.GetString("dynamic.feature.flag")
//
//	// 监听配置变化
//	client.OnConfigChange(func(e fsnotify.Event) {
//	    fmt.Println("配置文件发生变化:", e.Name)
//	})
//	client.WatchConfig()
//
//	// 设置配置项（测试环境）
//	client.Set("app.debug", true)
func GetClient(filePath ...string) (*viper.Viper, error) {
	globalMutex.RLock()
	if isInitialized && globalViper != nil {
		defer globalMutex.RUnlock()
		return globalViper, nil
	}
	globalMutex.RUnlock()

	// 如果未初始化，创建新实例
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if !isInitialized {
		v, err := createViperInstanceWithError(filePath...)
		if err != nil {
			return nil, fmt.Errorf("初始化配置客户端失败: %w", err)
		}
		globalViper = v
		isInitialized = true
	}

	return globalViper, nil
}

// MustGetClient 获取配置客户端，如果失败则panic
//
// 这是GetClient的panic版本，适用于配置初始化失败应该终止程序的场景
//
// 使用场景:
//   - ✅ 应用启动阶段，配置加载失败应该终止程序
//   - ✅ 简化错误处理的场景
//
// 示例:
//
//	client := config.MustGetClient()
//	value := client.GetString("app.name")
func MustGetClient(filePath ...string) *viper.Viper {
	client, err := GetClient(filePath...)
	if err != nil {
		panic(fmt.Sprintf("获取配置客户端失败: %v", err))
	}
	return client
}

// GetStringWithDefault 获取字符串配置项，支持默认值
//
// 参数:
//   - key: 配置键（支持嵌套，如 "app.name"）
//   - defaultValue: 默认值
//
// 使用场景:
//   - ✅ 简单的动态配置获取
//   - ✅ 需要默认值的配置项
//
// 返回:
//   - string: 配置值或默认值
//   - error: 如果获取配置客户端失败
//
// 示例:
//
//	appName, err := config.GetStringWithDefault("app.name", "默认应用名")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GetStringWithDefault(key, defaultValue string) (string, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetString(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetStringWithDefault 获取字符串配置项，如果失败则panic
func MustGetStringWithDefault(key, defaultValue string) string {
	value, err := GetStringWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetIntWithDefault 获取整数配置项，支持默认值
func GetIntWithDefault(key string, defaultValue int) (int, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetInt(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetIntWithDefault 获取整数配置项，如果失败则panic
func MustGetIntWithDefault(key string, defaultValue int) int {
	value, err := GetIntWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetBoolWithDefault 获取布尔配置项，支持默认值
func GetBoolWithDefault(key string, defaultValue bool) (bool, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetBool(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetBoolWithDefault 获取布尔配置项，如果失败则panic
func MustGetBoolWithDefault(key string, defaultValue bool) bool {
	value, err := GetBoolWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetIntWithValidation 获取整数配置项并进行范围验证
//
// 参数:
//   - key: 配置键
//   - defaultValue: 默认值
//   - min, max: 有效范围（包含边界）
//
// 返回:
//   - value: 配置值（如果超出范围则返回默认值）
//   - isValid: 是否在有效范围内
//   - error: 如果获取配置客户端失败
//
// 使用场景:
//   - ✅ 需要范围验证的数值配置
//   - ✅ 端口号、连接数等有限制的配置
//
// 示例:
//
//	port, valid, err := config.GetIntWithValidation("app.port", 8080, 1, 65535)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if !valid {
//	    log.Warn("端口配置超出有效范围，使用默认值")
//	}
func GetIntWithValidation(key string, defaultValue, min, max int) (int, bool, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, false, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	value := client.GetInt(key)
	globalMutex.Unlock()

	if value < min || value > max {
		return defaultValue, false, nil
	}
	return value, true, nil
}

// GetStringSliceWithDefault 获取字符串切片配置项，支持默认值
func GetStringSliceWithDefault(key string, defaultValue []string) ([]string, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetStringSlice(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetStringSliceWithDefault 获取字符串切片配置项，如果失败则panic
func MustGetStringSliceWithDefault(key string, defaultValue []string) []string {
	value, err := GetStringSliceWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetFloat64WithDefault 获取浮点数配置项，支持默认值
func GetFloat64WithDefault(key string, defaultValue float64) (float64, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetFloat64(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetFloat64WithDefault 获取浮点数配置项，如果失败则panic
func MustGetFloat64WithDefault(key string, defaultValue float64) float64 {
	value, err := GetFloat64WithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetInt64WithDefault 获取64位整数配置项，支持默认值
func GetInt64WithDefault(key string, defaultValue int64) (int64, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetInt64(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetInt64WithDefault 获取64位整数配置项，如果失败则panic
func MustGetInt64WithDefault(key string, defaultValue int64) int64 {
	value, err := GetInt64WithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetUintWithDefault 获取无符号整数配置项，支持默认值
func GetUintWithDefault(key string, defaultValue uint) (uint, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetUint(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetUintWithDefault 获取无符号整数配置项，如果失败则panic
func MustGetUintWithDefault(key string, defaultValue uint) uint {
	value, err := GetUintWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetUint64WithDefault 获取64位无符号整数配置项，支持默认值
func GetUint64WithDefault(key string, defaultValue uint64) (uint64, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetUint64(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetUint64WithDefault 获取64位无符号整数配置项，如果失败则panic
func MustGetUint64WithDefault(key string, defaultValue uint64) uint64 {
	value, err := GetUint64WithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetDurationWithDefault 获取时间间隔配置项，支持默认值
func GetDurationWithDefault(key string, defaultValue time.Duration) (time.Duration, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetDuration(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetDurationWithDefault 获取时间间隔配置项，如果失败则panic
func MustGetDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	value, err := GetDurationWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetTimeWithDefault 获取时间配置项，支持默认值
func GetTimeWithDefault(key string, defaultValue time.Time) (time.Time, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetTime(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetTimeWithDefault 获取时间配置项，如果失败则panic
func MustGetTimeWithDefault(key string, defaultValue time.Time) time.Time {
	value, err := GetTimeWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetStringMapWithDefault 获取字符串映射配置项，支持默认值
func GetStringMapWithDefault(key string, defaultValue map[string]interface{}) (map[string]interface{}, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetStringMap(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetStringMapWithDefault 获取字符串映射配置项，如果失败则panic
func MustGetStringMapWithDefault(key string, defaultValue map[string]interface{}) map[string]interface{} {
	value, err := GetStringMapWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetStringMapStringWithDefault 获取字符串到字符串的映射配置项，支持默认值
func GetStringMapStringWithDefault(key string, defaultValue map[string]string) (map[string]string, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetStringMapString(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetStringMapStringWithDefault 获取字符串到字符串的映射配置项，如果失败则panic
func MustGetStringMapStringWithDefault(key string, defaultValue map[string]string) map[string]string {
	value, err := GetStringMapStringWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetStringMapStringSliceWithDefault 获取字符串到字符串切片的映射配置项，支持默认值
func GetStringMapStringSliceWithDefault(key string, defaultValue map[string][]string) (map[string][]string, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetStringMapStringSlice(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetStringMapStringSliceWithDefault 获取字符串到字符串切片的映射配置项，如果失败则panic
func MustGetStringMapStringSliceWithDefault(key string, defaultValue map[string][]string) map[string][]string {
	value, err := GetStringMapStringSliceWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetSizeInBytesWithDefault 获取字节大小配置项，支持默认值
func GetSizeInBytesWithDefault(key string, defaultValue int) (uint, error) {
	client, err := GetClient()
	if err != nil {
		return uint(defaultValue), err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	result := client.GetSizeInBytes(key)
	globalMutex.Unlock()

	return result, nil
}

// MustGetSizeInBytesWithDefault 获取字节大小配置项，如果失败则panic
func MustGetSizeInBytesWithDefault(key string, defaultValue int) uint {
	value, err := GetSizeInBytesWithDefault(key, defaultValue)
	if err != nil {
		panic(fmt.Sprintf("获取配置失败: %v", err))
	}
	return value
}

// GetFloat64WithValidation 获取浮点数配置项并进行范围验证
func GetFloat64WithValidation(key string, defaultValue, min, max float64) (float64, bool, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, false, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	value := client.GetFloat64(key)
	globalMutex.Unlock()

	if value < min || value > max {
		return defaultValue, false, nil
	}
	return value, true, nil
}

// GetDurationWithValidation 获取时间间隔配置项并进行范围验证
func GetDurationWithValidation(key string, defaultValue, min, max time.Duration) (time.Duration, bool, error) {
	client, err := GetClient()
	if err != nil {
		return defaultValue, false, err
	}

	// 使用全局锁保护viper操作，确保线程安全
	globalMutex.Lock()
	client.SetDefault(key, defaultValue)
	value := client.GetDuration(key)
	globalMutex.Unlock()

	if value < min || value > max {
		return defaultValue, false, nil
	}
	return value, true, nil
}

// IsSet 检查配置项是否已设置
//
// 使用场景:
//   - ✅ 条件配置加载
//   - ✅ 可选功能的开关检查
//
// 示例:
//
//	if exists, err := config.IsSet("features.advanced_mode"); err == nil && exists {
//	    // 启用高级功能
//	}
func IsSet(key string) (bool, error) {
	client, err := GetClient()
	if err != nil {
		return false, err
	}

	// IsSet 是只读操作，使用读锁
	globalMutex.RLock()
	result := client.IsSet(key)
	globalMutex.RUnlock()

	return result, nil
}

// AllKeys 获取所有配置键
//
// 使用场景:
//   - ✅ 配置调试和诊断
//   - ✅ 动态配置探索
//
// 示例:
//
//	keys, err := config.AllKeys()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("当前配置项: %v\n", keys)
func AllKeys() ([]string, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	// AllKeys 是只读操作，使用读锁
	globalMutex.RLock()
	result := client.AllKeys()
	globalMutex.RUnlock()

	return result, nil
}

// createViperInstanceWithError 创建并配置viper实例，返回错误（用于LoadConfig和GetClient）
func createViperInstanceWithError(filePath ...string) (*viper.Viper, error) {
	v := viper.New()

	// 确定配置文件路径
	var configPath string
	if len(filePath) > 0 && filePath[0] != "" {
		configPath = filePath[0]
	} else {
		// 默认使用项目根目录的 config.yml
		configPath = DefaultConfigFileName
	}

	// 解析文件路径和名称
	dir := filepath.Dir(configPath)
	filename := filepath.Base(configPath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")

	// 设置配置文件路径和名称
	if dir != "." {
		v.AddConfigPath(dir)
	} else {
		// 添加常见的配置文件搜索路径
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("./config")

		// 获取工作目录并添加为搜索路径
		if pwd, err := os.Getwd(); err == nil {
			v.AddConfigPath(pwd)
		}
	}

	v.SetConfigName(name)
	if ext != "" {
		v.SetConfigType(ext)
	}

	// 自动从 APP_NAME 环境变量获取前缀
	appName := os.Getenv("APP_NAME")
	if appName != "" {
		// 设置环境变量前缀
		v.SetEnvPrefix(strings.ToUpper(appName))
	}

	// 启用环境变量支持
	v.AutomaticEnv()

	// 设置环境变量前缀分隔符
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 允许环境变量覆盖配置文件中的值
	v.AllowEmptyEnv(true)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果是找不到配置文件的错误，提供更友好的错误信息
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("配置文件未找到: %s。请确保配置文件存在于正确的路径", configPath)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	return v, nil
}

// createViperInstance 创建并配置viper实例（内部函数，保持向后兼容）
func createViperInstance(filePath ...string) *viper.Viper {
	v, err := createViperInstanceWithError(filePath...)
	if err != nil {
		// 对于向后兼容，输出警告并返回空配置的viper实例
		fmt.Printf("警告: 配置初始化失败: %v\n", err)
		// 返回一个基本的viper实例，至少能处理环境变量
		v = viper.New()
		v.AutomaticEnv()
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}
	return v
}
