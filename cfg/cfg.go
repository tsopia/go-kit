package cfg

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// initViper 初始化 viper 实例
func initViper(prefix string, configPath string, hasConfig bool) (*viper.Viper, error) {
	v := viper.New()

	// 配置环境变量行为
	if prefix != "" {
		v.SetEnvPrefix(prefix)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 如果找到了配置文件，读取它
	if hasConfig && configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	return v, nil
}

// New 创建一个新的 Provider 实例
// 自动按以下顺序查找配置：
// 1. 如果传入了 path 参数，直接使用该路径
// 2. 自动查找当前目录及父目录中的 config.*（支持 .yml/.yaml/.json/.toml/.env）
// 3. 环境变量（优先级最高，覆盖配置文件的值）
//
// 使用示例：
//   cfg.New()                                    // 自动查找配置
//   cfg.New("./config/app.yaml")                 // 指定路径
func New(path ...string) (Provider, error) {
	return NewWithPrefix("", path...)
}

// NewWithPrefix 创建带环境变量前缀的 Provider
//
// 使用示例：
//   cfg.NewWithPrefix("MYAPP")                   // 自动查找，带前缀
//   cfg.NewWithPrefix("MYAPP", "./config.yaml")  // 指定路径，带前缀
func NewWithPrefix(prefix string, path ...string) (Provider, error) {
	// 确定配置文件路径
	var configPath string
	var found bool
	var err error

	if len(path) > 0 && path[0] != "" {
		// 显式指定了路径
		configPath, found, err = resolveExplicitPath(path[0])
		if err != nil {
			return nil, fmt.Errorf("resolve config path: %w", err)
		}
	} else {
		// 自动查找配置文件
		configPath, found, err = findConfigFileAuto()
		if err != nil {
			return nil, fmt.Errorf("find config file: %w", err)
		}
	}

	// 初始化 viper
	v, err := initViper(prefix, configPath, found)
	if err != nil {
		return nil, fmt.Errorf("init viper: %w", err)
	}

	return newViperProvider(v), nil
}

// NewFromMap 从 map 创建 Provider（主要用于测试）
// 创建一个独立的、不受环境变量影响的 Provider 实例
func NewFromMap(m map[string]any) Provider {
	return newMapProvider(m)
}
