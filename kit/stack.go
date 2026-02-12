package kit

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// stackTracer 堆栈跟踪器
type stackTracer struct {
	config StackTraceConfig
}

// newStackTracer 创建堆栈跟踪器
func newStackTracer(config StackTraceConfig) *stackTracer {
	return &stackTracer{config: config}
}

// getCaller 获取调用者信息（文件:行号）
// skip 为跳过的帧数，0 表示 getCaller 本身
func (s *stackTracer) getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	// 简化路径，只保留最后两级
	simplified := simplifyPath(file)
	return fmt.Sprintf("%s:%d", simplified, line)
}

// getStack 获取堆栈信息
// 返回堆栈帧列表（项目内）
func (s *stackTracer) getStack() []string {
	if !s.config.Enabled {
		return nil
	}

	// 获取堆栈
	buf := make([]byte, 1024*s.config.Depth)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])

	// 解析堆栈帧
	frames := s.parseStack(stack)
	return frames
}

// parseStack 解析堆栈字符串
func (s *stackTracer) parseStack(stack string) []string {
	lines := strings.Split(stack, "\n")
	var frames []string

	// runtime.Stack 输出格式：
	// goroutine 1 [running]:
	// main.main()
	//     /path/to/main.go:10 +0x...
	// main.foo()
	//     /path/to/foo.go:20 +0x...

	for i := 1; i < len(lines); i += 2 {
		if i+1 >= len(lines) {
			break
		}

		// 函数名行
		funcLine := strings.TrimSpace(lines[i])
		// 文件行
		fileLine := strings.TrimSpace(lines[i+1])

		// 跳过 kit 包内部调用
		if strings.Contains(funcLine, "github.com/tsopia/go-kit/kit") {
			continue
		}

		// 跳过 runtime 帧
		if s.config.SkipRuntime && strings.HasPrefix(funcLine, "runtime.") {
			continue
		}

		// 解析文件路径和行号
		if loc := extractLocation(fileLine); loc != "" {
			frames = append(frames, loc)
		}

		// 限制深度
		if len(frames) >= s.config.Depth {
			break
		}
	}

	return frames
}

// extractLocation 从文件行提取位置信息
// 输入: "/path/to/file.go:123 +0x..."
// 输出: "file.go:123"
func extractLocation(fileLine string) string {
	// 找到第一个空格之前的部分
	if idx := strings.Index(fileLine, " "); idx > 0 {
		fileLine = fileLine[:idx]
	}

	// 简化路径
	simplified := simplifyPath(fileLine)
	return simplified
}

// simplifyPath 简化文件路径
// /Users/kj/projects/go-kit/service/user.go:42 -> service/user.go:42
func simplifyPath(path string) string {
	// 分离行号
	parts := strings.Split(path, ":")
	if len(parts) < 2 {
		return path
	}

	file := parts[0]
	line := parts[1]

	// 获取最后两级目录
	dir := filepath.Dir(file)
	base := filepath.Base(file)
	parent := filepath.Base(dir)

	return parent + "/" + base + ":" + line
}

// shouldCaptureStack 判断是否应该捕获堆栈
func (s *stackTracer) shouldCaptureStack(level Level) bool {
	if !s.config.Enabled {
		return false
	}
	return level >= s.config.Level
}
