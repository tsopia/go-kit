package pgmq

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

const (
	DefaultSchema        = "pgmq"
	ExtensionName        = "pgmq"
	DefaultDLQSuffix     = "_dlq"
	DefaultVisibility    = 30 * time.Second
	DefaultReadQuantity  = 1
	DefaultMaxRetries    = 5
	DefaultRetryDelay    = 2 * time.Second
	DefaultRetryMaxDelay = 5 * time.Minute
	DefaultBackoffFactor = 2.0
	DefaultPollInterval  = 200 * time.Millisecond
	DefaultConcurrency   = 4
)

var queuePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// QueueConfig 队列配置
type QueueConfig struct {
	Schema         string
	CheckExtension bool
	EnsureQueue    bool
	DLQSuffix      string
	Visibility     time.Duration
	ReadQuantity   int
	Retry          RetryConfig
	Consumer       ConsumerConfig
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	VisibilityTimeout time.Duration
	PollInterval      time.Duration
	MaxConcurrency    int
}

// ReadOptions 读取配置
type ReadOptions struct {
	VisibilityTimeout time.Duration
	Quantity          int
}

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

// SetDefaults 设置默认值
func (c *QueueConfig) SetDefaults() {
	if c.Schema == "" {
		c.Schema = DefaultSchema
	}
	if c.DLQSuffix == "" {
		c.DLQSuffix = DefaultDLQSuffix
	}
	if c.Visibility == 0 {
		c.Visibility = DefaultVisibility
	}
	if c.ReadQuantity == 0 {
		c.ReadQuantity = DefaultReadQuantity
	}
	if c.Retry.MaxRetries == 0 {
		c.Retry.MaxRetries = DefaultMaxRetries
	}
	if c.Retry.InitialDelay == 0 {
		c.Retry.InitialDelay = DefaultRetryDelay
	}
	if c.Retry.MaxDelay == 0 {
		c.Retry.MaxDelay = DefaultRetryMaxDelay
	}
	if c.Retry.BackoffFactor == 0 {
		c.Retry.BackoffFactor = DefaultBackoffFactor
	}
	if c.Consumer.MaxConcurrency == 0 {
		c.Consumer.MaxConcurrency = DefaultConcurrency
	}
	if c.Consumer.VisibilityTimeout == 0 {
		c.Consumer.VisibilityTimeout = DefaultVisibility
	}
	if c.Consumer.PollInterval == 0 {
		c.Consumer.PollInterval = DefaultPollInterval
	}
}

// Validate 校验配置
func (c *QueueConfig) Validate() error {
	if c.Schema == "" {
		return ErrInvalidConfig
	}
	if c.DLQSuffix == "" {
		return ErrInvalidConfig
	}
	if c.Visibility <= 0 {
		return ErrInvalidConfig
	}
	if c.ReadQuantity <= 0 {
		return ErrInvalidConfig
	}
	if c.Retry.MaxRetries < 0 {
		return ErrInvalidConfig
	}
	if c.Retry.InitialDelay < 0 || c.Retry.MaxDelay < 0 {
		return ErrInvalidConfig
	}
	if c.Retry.BackoffFactor < 1 {
		return ErrInvalidConfig
	}
	if c.Consumer.MaxConcurrency <= 0 {
		return ErrInvalidConfig
	}
	if c.Consumer.VisibilityTimeout <= 0 {
		return ErrInvalidConfig
	}
	if c.Consumer.PollInterval <= 0 {
		return ErrInvalidConfig
	}
	return nil
}

func normalizeReadOptions(cfg QueueConfig, opts ReadOptions) (ReadOptions, error) {
	if opts.VisibilityTimeout == 0 {
		opts.VisibilityTimeout = cfg.Visibility
	}
	if opts.Quantity == 0 {
		opts.Quantity = cfg.ReadQuantity
	}
	if opts.VisibilityTimeout <= 0 {
		return ReadOptions{}, ErrInvalidConfig
	}
	if opts.Quantity <= 0 {
		return ReadOptions{}, ErrInvalidQuantity
	}
	return opts, nil
}

func normalizeConsumerConfig(cfg ConsumerConfig) ConsumerConfig {
	if cfg.VisibilityTimeout <= 0 {
		cfg.VisibilityTimeout = DefaultVisibility
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = DefaultConcurrency
	}
	return cfg
}

func validateQueueName(name string) error {
	if name == "" {
		return ErrMissingQueue
	}
	if !queuePattern.MatchString(name) {
		return fmt.Errorf("%w: %s", ErrInvalidQueue, name)
	}
	return nil
}

func computeBackoffDelay(cfg RetryConfig, attempt int32) time.Duration {
	if attempt <= 0 {
		return cfg.InitialDelay
	}
	power := math.Pow(cfg.BackoffFactor, float64(attempt-1))
	seconds := float64(cfg.InitialDelay) * power
	delay := time.Duration(seconds)
	if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return delay
}
