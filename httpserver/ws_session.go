package httpserver

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type wsSession struct {
	ctx       context.Context
	request   *http.Request
	params    gin.Params
	recv      <-chan WSMessage
	send      chan WSMessage
	closeFn   func(int, string) error
	closeOnce sync.Once
	closeErr  error
}

func (s *wsSession) Context() context.Context {
	return s.ctx
}

func (s *wsSession) Request() *http.Request {
	return s.request
}

func (s *wsSession) Param(name string) string {
	return s.params.ByName(name)
}

func (s *wsSession) Recv() <-chan WSMessage {
	return s.recv
}

func (s *wsSession) Send(msg WSMessage) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case s.send <- msg:
		return nil
	}
}

func (s *wsSession) TrySend(msg WSMessage) bool {
	select {
	case <-s.ctx.Done():
		return false
	case s.send <- msg:
		return true
	default:
		return false
	}
}

func (s *wsSession) Close(code int, reason string) error {
	s.closeOnce.Do(func() {
		if s.closeFn != nil {
			s.closeErr = s.closeFn(code, reason)
		}
	})
	return s.closeErr
}
