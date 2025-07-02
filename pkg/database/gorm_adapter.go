package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm/logger"
)

// SimpleLogger 定义基础日志接口
type SimpleLogger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// ContextualLogger 定义支持Context的日志接口
type ContextualLogger interface {
	SimpleLogger
	InfoWithContext(ctx context.Context, msg string, fields ...interface{})
	WarnWithContext(ctx context.Context, msg string, fields ...interface{})
	ErrorWithContext(ctx context.Context, msg string, fields ...interface{})
}

// GORMLogger 把任意 SimpleLogger 转成 gorm.logger.Interface
type GORMLogger struct {
	l            SimpleLogger
	level        logger.LogLevel
	traceEnabled bool
}

// NewGormLogger 构造函数
//   - l: 你的 zap / logrus / zerolog ... 实例
//   - level: GORM 日志级别，不想打印 SQL 就传 logger.Silent
func NewGormLogger(l SimpleLogger, logLevel string) logger.Interface {
	level := getLogLevel(logLevel)
	return &GORMLogger{
		l:            l,
		level:        level,
		traceEnabled: level == logger.Info || level == logger.Warn,
	}
}

// LogMode 实现接口
func (g *GORMLogger) LogMode(l logger.LogLevel) logger.Interface {
	return &GORMLogger{
		l:            g.l,
		level:        l,
		traceEnabled: l == logger.Info || l == logger.Warn,
	}
}

// Info 实现logger.Interface
func (g *GORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.level < logger.Info {
		return
	}

	message := fmt.Sprintf(msg, data...)

	// 如果日志器支持Context，优先使用Context方法
	if cl, ok := g.l.(ContextualLogger); ok {
		cl.InfoWithContext(ctx, message)
	} else {
		g.l.Info(message)
	}
}

// Warn 实现logger.Interface
func (g *GORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.level < logger.Warn {
		return
	}

	message := fmt.Sprintf(msg, data...)

	// 如果日志器支持Context，优先使用Context方法
	if cl, ok := g.l.(ContextualLogger); ok {
		cl.WarnWithContext(ctx, message)
	} else {
		g.l.Warn(message)
	}
}

// Error 实现logger.Interface
func (g *GORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.level < logger.Error {
		return
	}

	message := fmt.Sprintf(msg, data...)

	// 如果日志器支持Context，优先使用Context方法
	if cl, ok := g.l.(ContextualLogger); ok {
		cl.ErrorWithContext(ctx, message)
	} else {
		g.l.Error(message)
	}
}

// Trace 打印 SQL
func (g *GORMLogger) Trace(ctx context.Context, begin time.Time,
	fc func() (sql string, rowsAffected int64), err error) {

	if !g.traceEnabled {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 构建结构化的日志字段
	fields := []interface{}{
		"sql", sql,
		"elapsed", elapsed,
		"rows", rows,
	}

	// 处理错误情况
	if err != nil {
		fields = append(fields, "error", err)

		// 根据错误类型选择不同的日志级别
		switch err {
		case logger.ErrRecordNotFound:
			// 记录未找到通常不是错误，使用Info级别
			if cl, ok := g.l.(ContextualLogger); ok {
				cl.InfoWithContext(ctx, "GORM SQL - Record not found", fields...)
			} else {
				g.l.Info("GORM SQL - Record not found", fields...)
			}
		default:
			// 其他错误使用Error级别
			if cl, ok := g.l.(ContextualLogger); ok {
				cl.ErrorWithContext(ctx, "GORM SQL ERROR", fields...)
			} else {
				g.l.Error("GORM SQL ERROR", fields...)
			}
		}
	} else {
		// 成功情况使用Info级别
		if cl, ok := g.l.(ContextualLogger); ok {
			cl.InfoWithContext(ctx, "GORM SQL", fields...)
		} else {
			g.l.Info("GORM SQL", fields...)
		}
	}
}

// getLogLevel 获取日志级别
func getLogLevel(level string) logger.LogLevel {
	switch level {
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	case "silent":
		return logger.Silent
	default:
		return logger.Silent
	}
}

// IsValidLogLevel 验证日志级别是否有效
func IsValidLogLevel(level string) bool {
	switch level {
	case "error", "warn", "info", "silent":
		return true
	default:
		return false
	}
}
