package httpserver

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type wsSession struct {
	ctx             context.Context
	request         *http.Request
	params          gin.Params
	recv            <-chan WSMessage
	send            chan WSMessage
	closeFn         func(int, string) error
	gracefulCloseFn func(context.Context, int, string) error
	stateMu         sync.Mutex
	closing         bool
	activeSenders   sync.WaitGroup
	closeOnce       sync.Once
	closeErr        error
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
	if err := s.beginSend(); err != nil {
		return err
	}
	defer s.activeSenders.Done()

	select {
	case <-s.ctx.Done():
		return s.sendErr()
	case s.send <- msg:
		return nil
	}
}

func (s *wsSession) TrySend(msg WSMessage) bool {
	if err := s.beginSend(); err != nil {
		return false
	}
	defer s.activeSenders.Done()

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
		s.markClosing()
		if s.closeFn != nil {
			s.closeErr = s.closeFn(code, reason)
		}
		s.activeSenders.Wait()
	})
	return s.closeErr
}

func (s *wsSession) CloseGracefully(ctx context.Context, code int, reason string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.closeOnce.Do(func() {
		s.markClosing()
		if err := s.waitForActiveSenders(ctx); err != nil {
			s.closeErr = err
			if s.closeFn != nil {
				if closeErr := s.closeFn(code, reason); closeErr != nil {
					s.closeErr = closeErr
				}
			}
			return
		}
		if s.gracefulCloseFn != nil {
			s.closeErr = s.gracefulCloseFn(ctx, code, reason)
			return
		}
		if s.closeFn != nil {
			s.closeErr = s.closeFn(code, reason)
		}
	})
	return s.closeErr
}

func (s *wsSession) beginSend() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.closing {
		return ErrWSSessionClosed
	}
	s.activeSenders.Add(1)
	return nil
}

func (s *wsSession) markClosing() {
	s.stateMu.Lock()
	s.closing = true
	s.stateMu.Unlock()
}

func (s *wsSession) isClosing() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.closing
}

func (s *wsSession) sendErr() error {
	if s.isClosing() {
		return ErrWSSessionClosed
	}
	return s.ctx.Err()
}

func (s *wsSession) waitForActiveSenders(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.activeSenders.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
