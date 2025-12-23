package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// slogHandler 自定义 slog Handler，支持采样、轮转、默认字段、上下文字段、Hook、时间格式等。
type slogHandler struct {
	opts         Options
	handler      slog.Handler
	levelVar     *slog.LevelVar
	ctxExtractor ContextExtractor
	hooks        []Hook
	defaultAttrs []slog.Attr
	sampler      *logSampler
	includeStack bool
}

func newSlogHandler(opts Options, extractor ContextExtractor, hooks []Hook) (*slogHandler, *slog.LevelVar) {
	levelVar := &slog.LevelVar{}
	levelVar.Set(convertSlogLevel(opts.Level))

	writer := buildSlogWriter(opts)
	handlerOpts := &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: opts.Caller,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && opts.TimeFormat != "" {
				a.Value = slog.StringValue(a.Value.Time().Format(opts.TimeFormat))
			}

			if opts.Color && opts.Format != FormatJSON && a.Key == slog.LevelKey {
				a.Value = slog.StringValue(colorizeLevel(a.Value.String()))
			}
			return a
		},
	}

	var baseHandler slog.Handler
	if opts.Format == FormatJSON {
		baseHandler = slog.NewJSONHandler(writer, handlerOpts)
	} else {
		baseHandler = slog.NewTextHandler(writer, handlerOpts)
	}

	defaultAttrs := make([]slog.Attr, 0, len(opts.Fields))
	for k, v := range opts.Fields {
		defaultAttrs = append(defaultAttrs, slog.Any(k, v))
	}

	var sampler *logSampler
	if opts.Sampling != nil {
		sampler = newLogSampler(*opts.Sampling)
	}

	return &slogHandler{
		opts:         opts,
		handler:      baseHandler,
		levelVar:     levelVar,
		ctxExtractor: extractor,
		hooks:        hooks,
		defaultAttrs: defaultAttrs,
		sampler:      sampler,
		includeStack: opts.Stacktrace,
	}, levelVar
}

// Enabled 判断是否启用
func (h *slogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录
func (h *slogHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.sampler != nil && !h.sampler.Allow() {
		return nil
	}

	for _, attr := range h.defaultAttrs {
		record.AddAttrs(attr)
	}

	if h.ctxExtractor != nil {
		for k, v := range h.ctxExtractor.Extract(ctx) {
			record.AddAttrs(slog.Any(k, v))
		}
	}

	if h.includeStack && record.Level >= slog.LevelError {
		record.AddAttrs(slog.String("stacktrace", string(debug.Stack())))
	}

	if len(h.hooks) > 0 {
		attrs := make([]slog.Attr, 0, record.NumAttrs())
		record.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a)
			return true
		})

		entry := Entry{
			Level:      record.Level,
			Time:       record.Time,
			Message:    record.Message,
			Attributes: attrs,
		}

		for _, hook := range h.hooks {
			if err := hook(entry); err != nil {
				fmt.Fprintf(os.Stderr, "日志钩子执行失败: %v\n", err)
			}
		}
	}

	return h.handler.Handle(ctx, record)
}

// WithAttrs 返回带附加属性的新 Handler
func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogHandler{
		opts:         h.opts,
		handler:      h.handler.WithAttrs(attrs),
		levelVar:     h.levelVar,
		ctxExtractor: h.ctxExtractor,
		hooks:        h.hooks,
		defaultAttrs: h.defaultAttrs,
		sampler:      h.sampler,
		includeStack: h.includeStack,
	}
}

// WithGroup 返回带分组的新 Handler
func (h *slogHandler) WithGroup(name string) slog.Handler {
	return &slogHandler{
		opts:         h.opts,
		handler:      h.handler.WithGroup(name),
		levelVar:     h.levelVar,
		ctxExtractor: h.ctxExtractor,
		hooks:        h.hooks,
		defaultAttrs: h.defaultAttrs,
		sampler:      h.sampler,
		includeStack: h.includeStack,
	}
}

// buildSlogWriter 构建 slog 输出
func buildSlogWriter(opts Options) io.Writer {
	writers := []io.Writer{os.Stdout}

	if opts.EnableFileOutput {
		if opts.Rotate != nil {
			writers = append(writers, &lumberjack.Logger{
				Filename:   opts.Rotate.Filename,
				MaxSize:    opts.Rotate.MaxSize,
				MaxBackups: opts.Rotate.MaxBackups,
				MaxAge:     opts.Rotate.MaxAge,
				Compress:   opts.Rotate.Compress,
				LocalTime:  opts.Rotate.LocalTime,
			})
		} else {
			logPath := GetDefaultLogPath()
			if err := EnsureLogDirForPath(logPath); err == nil {
				file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
				if err == nil {
					writers = append(writers, file)
				}
			}
		}
	}

	if len(writers) == 1 {
		return writers[0]
	}
	return io.MultiWriter(writers...)
}

type logSampler struct {
	cfg      SamplingConfig
	mu       sync.Mutex
	count    int
	lastTick time.Time
}

func newLogSampler(cfg SamplingConfig) *logSampler {
	return &logSampler{cfg: cfg}
}

func (s *logSampler) Allow() bool {
	s.mu.Lock()
	// lock before possible return
	defer s.mu.Unlock()

	if s.cfg.Tick <= 0 {
		s.cfg.Tick = time.Second
	}
	if s.cfg.Thereafter <= 0 {
		s.cfg.Thereafter = 1
	}

	now := time.Now()
	if s.lastTick.IsZero() || now.Sub(s.lastTick) > s.cfg.Tick {
		s.lastTick = now
		s.count = 0
	}

	if s.count < s.cfg.Initial {
		s.count++
		return true
	}

	s.count++
	return (s.count-s.cfg.Initial)%s.cfg.Thereafter == 0
}

func convertSlogLevel(level Level) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel, FatalLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func colorizeLevel(level string) string {
	switch level {
	case "DEBUG", "debug":
		return "\033[36m" + level + "\033[0m"
	case "INFO", "info":
		return "\033[32m" + level + "\033[0m"
	case "WARN", "warn", "WARNING", "warning":
		return "\033[33m" + level + "\033[0m"
	case "ERROR", "error":
		return "\033[31m" + level + "\033[0m"
	default:
		return level
	}
}
