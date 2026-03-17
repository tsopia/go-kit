package httpserver

import (
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
func (s *Server) MarkReady() {
	s.setState(StateReady)
}

// MarkDraining 将服务器标记为排空中。
func (s *Server) MarkDraining() {
	s.setState(StateDraining)
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
