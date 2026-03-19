package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// State 描述服务器生命周期状态。
type State string

const (
	StateNew      State = "new"
	StateStarting State = "starting"
	StateReady    State = "ready"
	StateDraining State = "draining"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
	StateFailed   State = "failed"
)

// validTransitions 定义允许的状态转换
var validTransitions = map[State][]State{
	StateNew:      {StateStarting, StateStopped},
	StateStarting: {StateReady, StateDraining, StateStopping, StateStopped, StateFailed},
	StateReady:    {StateDraining, StateStopping, StateStopped, StateFailed},
	StateDraining: {StateStopping, StateStopped, StateFailed},
	StateStopping: {StateStopped, StateFailed},
	StateStopped:  {},
	StateFailed:   {StateStarting, StateStopping, StateStopped}, // 允许从失败重启或进入清理流程
}

// canTransition 检查状态转换是否合法
func canTransition(from, to State) bool {
	validStates, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, valid := range validStates {
		if valid == to {
			return true
		}
	}
	return false
}

// tryTransitionTo 尝试状态转换，失败时返回错误。
func (s *Server) tryTransitionTo(state State) error {
	s.stateMu.Lock()
	from := s.state
	if !canTransition(from, state) {
		s.stateMu.Unlock()
		return fmt.Errorf("invalid state transition: %s -> %s", from, state)
	}
	s.state = state
	s.stateMu.Unlock()

	if s.hooks.OnStateChange != nil {
		s.hooks.OnStateChange(context.Background(), from, state)
	}
	return nil
}

// setState 直接设置状态（用于初始化或测试）
func (s *Server) setState(state State) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.state = state
}

// State 返回当前服务器状态。
func (s *Server) State() State {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	return s.state
}

// MarkReady 将服务器标记为可接流量。
func (s *Server) MarkReady() error {
	return s.tryTransitionTo(StateReady)
}

// MarkDraining 将服务器标记为排空中。
func (s *Server) MarkDraining() error {
	return s.tryTransitionTo(StateDraining)
}

func (s *Server) registerProbeRoutes() {
	if !s.config.EnableHealthCheck {
		return
	}
	s.registerReadinessRoute()
	s.registerLivenessRoute()
}

func (s *Server) registerReadinessRoute() {
	if s.readinessRouteRegistered {
		return
	}

	s.engine.GET(s.config.ReadinessPath, s.readinessEndpoint())
	s.readinessRouteRegistered = true
}

func (s *Server) registerLivenessRoute() {
	if s.livenessRouteRegistered {
		return
	}

	s.engine.GET(s.config.LivenessPath, s.livenessEndpoint())
	s.livenessRouteRegistered = true
}

func (s *Server) readinessEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.State() == StateReady {
			c.Status(http.StatusOK)
			return
		}

		c.Status(http.StatusServiceUnavailable)
	}
}

func (s *Server) livenessEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch s.State() {
		case StateStopped, StateFailed:
			c.Status(http.StatusServiceUnavailable)
		default:
			c.Status(http.StatusOK)
		}
	}
}
