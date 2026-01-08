package pgmq

import "time"

// Option 配置 Queue
type Option func(*QueueConfig, *QueueOptions)

// QueueOptions 额外注入项
type QueueOptions struct {
	logger  SimpleLogger
	metrics Metrics
}

// WithSchema 设置 schema
func WithSchema(schema string) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.Schema = schema
	}
}

// WithCheckExtension 是否检查扩展
func WithCheckExtension(enabled bool) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.CheckExtension = enabled
	}
}

// WithEnsureQueue 是否自动创建队列
func WithEnsureQueue(enabled bool) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.EnsureQueue = enabled
	}
}

// WithDLQSuffix 设置死信队列后缀
func WithDLQSuffix(suffix string) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.DLQSuffix = suffix
	}
}

// WithVisibilityTimeout 设置默认可见性超时
func WithVisibilityTimeout(timeout time.Duration) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.Visibility = timeout
	}
}

// WithReadQuantity 设置默认读取数量
func WithReadQuantity(quantity int) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.ReadQuantity = quantity
	}
}

// WithRetryConfig 设置重试配置
func WithRetryConfig(retry RetryConfig) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.Retry = retry
	}
}

// WithConsumerConfig 设置消费者配置
func WithConsumerConfig(consumer ConsumerConfig) Option {
	return func(cfg *QueueConfig, _ *QueueOptions) {
		cfg.Consumer = consumer
	}
}

// WithLogger 注入日志器
func WithLogger(logger SimpleLogger) Option {
	return func(_ *QueueConfig, opt *QueueOptions) {
		opt.logger = logger
	}
}

// WithMetrics 注入指标实现
func WithMetrics(metrics Metrics) Option {
	return func(_ *QueueConfig, opt *QueueOptions) {
		opt.metrics = metrics
	}
}
