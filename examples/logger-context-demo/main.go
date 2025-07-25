package main

import (
	"context"
	"time"

	"github.com/tsopia/go-kit/pkg/logger"
)

func main() {
	// 1. 基本配置：只输出到stdout
	basicLogger := logger.NewWithOptions(logger.Options{
		Level:      logger.InfoLevel,
		Format:     logger.FormatConsole,
		TimeFormat: "2006-01-02 15:04:05",
		Caller:     true,
		Stacktrace: true,
	})

	basicLogger.Info("基本配置日志 - 只输出到stdout")

	// 2. 配置默认日志文件路径
	logger.SetDefaultLogDir("demo_logs")
	logger.SetDefaultLogFile("demo.log")

	logger.Infof("默认日志路径设置为: %s", logger.GetDefaultLogPath())

	// 3. 开发环境配置：只输出到stdout，debug级别
	logger.SetupDevelopment()
	logger.Info("开发环境配置")
	logger.Debug("调试信息")

	// 4. 生产环境配置：同时输出到stdout和文件（使用默认路径）
	logger.SetupProduction()
	logger.Info("生产环境配置 - 同时输出到stdout和文件")

	// 5. 自定义配置：设置不同的日志级别和格式
	customLogger := logger.NewWithOptions(logger.Options{
		Level:            logger.WarnLevel,
		Format:           logger.FormatJSON,
		TimeFormat:       time.RFC3339,
		Caller:           true,
		Stacktrace:       false,
		EnableFileOutput: true, // 启用文件输出，使用默认路径
		Rotate: &logger.RotateConfig{
			Filename:   "demo_logs/custom.log", // 自定义文件路径
			MaxSize:    50,                     // MB
			MaxBackups: 5,
			MaxAge:     7, // 天
			Compress:   true,
			LocalTime:  true,
		},
	})

	customLogger.Info("这条信息不会显示，因为级别是WARN")
	customLogger.Warn("自定义配置警告日志")
	customLogger.Error("自定义配置错误日志")

	// 6. 带上下文的日志
	ctx := context.Background()
	ctx = context.WithValue(ctx, logger.ContextKey("trace_id"), "trace-12345")
	ctx = context.WithValue(ctx, logger.ContextKey("user_id"), 123)

	ctxLogger := customLogger.WithContext(ctx)
	ctxLogger.Warn("带上下文的日志")

	// 7. 格式化日志
	customLogger.Warnf("用户 %d 在 %s 执行了操作", 123, time.Now().Format("15:04:05"))

	// 8. 全局日志配置
	logger.Init(logger.Options{
		Level:            logger.DebugLevel,
		Format:           logger.FormatConsole,
		TimeFormat:       "15:04:05",
		Caller:           true,
		EnableFileOutput: false, // 只输出到stdout
	})

	logger.Info("全局配置日志")
	logger.Debugf("调试信息：%s", "测试数据")

	// 9. 确保日志目录存在的示例
	if err := logger.EnsureLogDir(); err != nil {
		logger.Error("创建日志目录失败", "error", err)
	}

	// 10. 带文件输出的完整配置
	fileLogger := logger.NewWithOptions(logger.Options{
		Level:            logger.InfoLevel,
		Format:           logger.FormatJSON,
		TimeFormat:       time.RFC3339,
		Caller:           true,
		Stacktrace:       true,
		EnableFileOutput: true, // 使用默认路径
		Fields: map[string]interface{}{
			"service": "demo",
			"version": "1.0.0",
		},
	})

	fileLogger.Info("完整配置日志 - 同时输出到stdout和文件")
	fileLogger.WithFields(map[string]interface{}{
		"action": "demo",
		"module": "main",
	}).Info("带额外字段的日志")

	// 11. 演示清理功能（实际使用中通常在测试中使用）
	logger.Info("演示完成，可以使用 logger.CleanupLogFiles() 清理日志文件")

	// 同步日志缓冲区
	fileLogger.Sync()
	customLogger.Sync()
	logger.Sync()

	logger.Info("日志配置演示完成")
}
