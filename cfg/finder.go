package cfg

import (
	"fmt"
	"os"
	"path/filepath"
)

// 支持的配置文件扩展名（按优先级排序）
var supportedExts = []string{".yml", ".yaml", ".json", ".toml", ".env"}

// resolveExplicitPath 解析显式指定的路径
// 返回：绝对路径、是否找到、错误
func resolveExplicitPath(path string) (string, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false, fmt.Errorf("resolve path: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", false, fmt.Errorf("config file not found: %s", absPath)
		}
		return "", false, fmt.Errorf("check config file: %w", err)
	}

	return absPath, true, nil
}

// findConfigFileAuto 自动查找配置文件
// 从当前目录向上查找 config.*（最多3层）
// 返回：配置文件路径、是否找到、错误
func findConfigFileAuto() (string, bool, error) {
	// 获取当前工作目录
	startDir, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("get working directory: %w", err)
	}

	// 最多向上查找3层
	maxDepth := 3
	currentDir := startDir

	for i := 0; i <= maxDepth; i++ {
		// 在当前目录查找 config.*
		for _, ext := range supportedExts {
			configPath := filepath.Join(currentDir, "config"+ext)
			if _, err := os.Stat(configPath); err == nil {
				// 找到配置文件
				absPath, err := filepath.Abs(configPath)
				if err != nil {
					return "", false, fmt.Errorf("resolve config path: %w", err)
				}
				return absPath, true, nil
			}
		}

		// 向上一级目录
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// 到达根目录，停止
			break
		}
		currentDir = parent
	}

	// 未找到配置文件
	return "", false, nil
}
