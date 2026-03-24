package httpserver

import (
	"errors"
)

// ErrSendBufferFull 发送缓冲区已满错误
var ErrSendBufferFull = errors.New("send buffer full")

// wsSender 带策略的 WebSocket 发送器
type wsSender struct {
	ch     chan WSMessage  // 实际发送 channel（给写 goroutine 消费）
	proxy  chan WSMessage  // 代理 channel（暴露给用户）
	policy WSBufferPolicy
}

// newWSSender 创建带策略的发送器
// 返回发送器实例，用户通过代理 channel 发送消息
func newWSSender(size int, policy WSBufferPolicy) *wsSender {
	return &wsSender{
		ch:     make(chan WSMessage, size),
		proxy:  make(chan WSMessage),
		policy: policy,
	}
}

// Proxy 返回代理 channel（暴露给用户）
func (s *wsSender) Proxy() chan<- WSMessage {
	return s.proxy
}

// Channel 返回实际发送 channel（供写 goroutine 消费）
func (s *wsSender) Channel() <-chan WSMessage {
	return s.ch
}

// Run 启动发送器，处理从代理到实际 channel 的转发
// 在单独的 goroutine 中运行，应用 SendPolicy
func (s *wsSender) Run(cancel func()) {
	defer close(s.ch)

	for msg := range s.proxy {
		switch s.policy {
		case Block:
			// 阻塞直到成功
			s.ch <- msg

		case DropNewest:
			// 尝试非阻塞发送，满了就丢弃新消息
			select {
			case s.ch <- msg:
				// 成功
			default:
				// 缓冲区满，丢弃新消息（什么都不做）
			}

		case DropOldest:
			// 尝试发送，满了就丢弃最旧的重试
			select {
			case s.ch <- msg:
				// 成功
			default:
				// 丢弃最旧，然后重试
				select {
				case <-s.ch:
				default:
				}
				select {
				case s.ch <- msg:
				default:
				}
			}

		case Disconnect:
			// 尝试非阻塞发送，满了就断开连接
			select {
			case s.ch <- msg:
				// 成功
			default:
				// 缓冲区满，断开连接
				cancel()
				return
			}
		}
	}
}
