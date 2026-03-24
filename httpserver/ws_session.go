package httpserver

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type wsSession struct {
	ctx     context.Context
	request *http.Request
	params  gin.Params
	recv    <-chan WSMessage
	send    chan<- WSMessage
	closeFn func(int, string) error
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
	if s.closeFn == nil {
		return nil
	}
	return s.closeFn(code, reason)
}
