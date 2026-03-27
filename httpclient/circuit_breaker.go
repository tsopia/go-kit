package httpclient

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen          = errors.New("circuit breaker is open")
	ErrCircuitHalfOpenLimited = errors.New("circuit breaker half-open request limit reached")
)

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	Execute(func() error) error
	State() string
}

type circuitState string

const (
	stateClosed   circuitState = "closed"
	stateOpen     circuitState = "open"
	stateHalfOpen circuitState = "half-open"
)

// statefulCircuitBreaker 具有基础状态机的熔断器实现
type statefulCircuitBreaker struct {
	config    CircuitBreakerConfig
	state     circuitState
	failures  uint32
	successes uint32
	openedAt  time.Time
	mu        sync.Mutex
}

// newCircuitBreaker 创建新的熔断器
func newCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 1
	}
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &statefulCircuitBreaker{config: config, state: stateClosed}
}

// Execute 执行函数
func (cb *statefulCircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.currentStateLocked()

	if state == stateOpen {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	if state == stateHalfOpen && cb.successes >= cb.config.MaxRequests {
		cb.mu.Unlock()
		return ErrCircuitHalfOpenLimited
	}

	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.updateStateLocked(err == nil)

	return err
}

// State 获取熔断器状态
func (cb *statefulCircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return string(cb.currentStateLocked())
}

func (cb *statefulCircuitBreaker) currentStateLocked() circuitState {
	if cb.state == stateOpen && time.Since(cb.openedAt) >= cb.config.Timeout {
		cb.state = stateHalfOpen
		cb.failures = 0
		cb.successes = 0
	}

	return cb.state
}

func (cb *statefulCircuitBreaker) updateStateLocked(success bool) {
	switch cb.currentStateLocked() {
	case stateClosed:
		if success {
			cb.failures = 0
			return
		}

		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.tripLocked()
		}
	case stateHalfOpen:
		if success {
			cb.successes++
			if cb.successes >= cb.config.SuccessThreshold {
				cb.resetLocked()
			}
			return
		}

		cb.tripLocked()
	}
}

func (cb *statefulCircuitBreaker) tripLocked() {
	cb.state = stateOpen
	cb.openedAt = time.Now()
	cb.failures = 0
	cb.successes = 0
}

func (cb *statefulCircuitBreaker) resetLocked() {
	cb.state = stateClosed
	cb.failures = 0
	cb.successes = 0
}
