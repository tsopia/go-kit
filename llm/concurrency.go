package llm

import (
	"context"
	"errors"
	"io"

	"github.com/cloudwego/eino/schema"
)

// concurrencyGuard 基于 buffered channel 的信号量，控制 Agent 调用并发。
// MaxConcurrency=0 时 guard 为 nil，acquire/release 为零开销空操作。
// 每个 Agent 实例独立持有 guard，多实例互不影响。
type concurrencyGuard struct {
	ch chan struct{}
}

func newConcurrencyGuard(maxConcurrency int) *concurrencyGuard {
	if maxConcurrency <= 0 {
		return nil
	}
	return &concurrencyGuard{ch: make(chan struct{}, maxConcurrency)}
}

// acquire 占用一个并发名额，可被 ctx 取消打断。
func (g *concurrencyGuard) acquire(ctx context.Context) error {
	if g == nil {
		return nil
	}
	select {
	case g.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release 释放一个并发名额。
func (g *concurrencyGuard) release() {
	if g == nil {
		return
	}
	<-g.ch
}

// wrapStreamWithGuard 包装 StreamReader，在流消费结束（EOF/error）
// 或读端提前 Close 时自动 release 名额并执行 onEnd 回调。
// 用于 Stream 模式下延迟释放并发名额——acquire 在 Stream 入口，
// release 延迟到调用方消费完流。
func wrapStreamWithGuard(sr *schema.StreamReader[*schema.Message], guard *concurrencyGuard, onEnd func()) *schema.StreamReader[*schema.Message] {
	if guard == nil && onEnd == nil {
		return sr
	}
	newSr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer func() {
			sr.Close() // 关闭原始 stream，避免底层资源泄漏
			sw.Close()
			if guard != nil {
				guard.release()
			}
			if onEnd != nil {
				onEnd()
			}
		}()
		for {
			msg, err := sr.Recv()
			if err != nil {
				// 非 EOF 错误传播给读端；EOF 直接结束
				if !errors.Is(err, io.EOF) {
					sw.Send(msg, err)
				}
				return
			}
			// 读端提前 Close 时 Send 返回 closed=true，退出并 release
			if sw.Send(msg, nil) {
				return
			}
		}
	}()
	return newSr
}
