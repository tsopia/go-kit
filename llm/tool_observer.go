package llm

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
)

type ctxKeyToolLogSession struct{}

type toolLogSession struct {
	mu       sync.Mutex
	attempts map[string]int
}

func withToolLogSession(ctx context.Context) context.Context {
	if toolLogSessionFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyToolLogSession{}, &toolLogSession{
		attempts: make(map[string]int),
	})
}

func toolLogSessionFromContext(ctx context.Context) *toolLogSession {
	session, _ := ctx.Value(ctxKeyToolLogSession{}).(*toolLogSession)
	return session
}

func (s *toolLogSession) nextAttempt(name string) int {
	if s == nil {
		return 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts[name]++
	return s.attempts[name]
}

type ctxKeyToolLogState struct{}

type toolLogState struct {
	mu        sync.Mutex
	hasError  bool
	errText   string
	retryable bool
	terminal  bool
}

func toolLogStateFromContext(ctx context.Context) *toolLogState {
	state, _ := ctx.Value(ctxKeyToolLogState{}).(*toolLogState)
	return state
}

func (s *toolLogState) markError(err error, retryable, terminal bool) {
	if s == nil || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hasError = true
	s.errText = err.Error()
	s.retryable = retryable
	s.terminal = terminal
}

func (s *toolLogState) snapshot() (string, bool, bool, bool) {
	if s == nil {
		return "", false, false, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.errText, s.retryable, s.terminal, s.hasError
}

func newToolObserverMiddleware(logs *structuredLogger, directReturnTools map[string]struct{}) compose.ToolMiddleware {
	if logs == nil || !logs.enabled() {
		return compose.ToolMiddleware{}
	}

	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				attempt := 1
				if session := toolLogSessionFromContext(ctx); session != nil {
					attempt = session.nextAttempt(input.Name)
				}

				startAttrs := []any{
					"tool_name", input.Name,
					"tool_call_id", input.CallID,
					"attempt", attempt,
				}
				if logs.cfg.LogToolArguments {
					startAttrs = append(startAttrs, "arguments", truncateField(input.Arguments, logs.cfg.MaxFieldLength))
				}
				logs.logInfo("tool.start", startAttrs...)

				state := &toolLogState{}
				ctx = context.WithValue(ctx, ctxKeyToolLogState{}, state)

				started := time.Now()
				output, err := next(ctx, input)
				latencyMS := time.Since(started).Milliseconds()

				if errText, retryable, terminal, ok := state.snapshot(); ok {
					logs.logError("tool.error",
						"tool_name", input.Name,
						"tool_call_id", input.CallID,
						"attempt", attempt,
						"latency_ms", latencyMS,
						"retryable", retryable,
						"terminal", terminal,
						"error", errText,
					)
					return output, err
				}

				if err != nil {
					logs.logError("tool.error",
						"tool_name", input.Name,
						"tool_call_id", input.CallID,
						"attempt", attempt,
						"latency_ms", latencyMS,
						"retryable", false,
						"terminal", true,
						"error", err.Error(),
					)
					return nil, err
				}

				successAttrs := []any{
					"tool_name", input.Name,
					"tool_call_id", input.CallID,
					"attempt", attempt,
					"latency_ms", latencyMS,
				}
				if _, ok := directReturnTools[input.Name]; ok {
					successAttrs = append(successAttrs, "direct_return", true)
				}
				if logs.cfg.LogToolResults && output != nil {
					successAttrs = append(successAttrs, "result", truncateField(output.Result, logs.cfg.MaxFieldLength))
				}
				logs.logInfo("tool.success", successAttrs...)
				return output, nil
			}
		},
	}
}
