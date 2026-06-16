package llm

import "sync"

// extractionState 记录 Extraction 模式下的工具调用状态。
// 被 react-based Agent 和 adk-based ADKAgent 共享。
type extractionState struct {
	mu            sync.Mutex
	successCount  int
	failureCount  int
	maxRetries    int
	lastToolName  string
	lastToolTotal string // 完整的工具输出（JSON）
}

func newExtractionState(maxRetries int) *extractionState {
	return &extractionState{maxRetries: maxRetries}
}

func (s *extractionState) recordSuccess(name, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.successCount++
	s.lastToolName = name
	s.lastToolTotal = result
}

func (s *extractionState) recordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failureCount++
}

// shouldForceToolCall 判断是否应该继续强制模型调用工具。
// 逻辑：尚无成功的工具调用 且 失败次数未超限 → 强制。
func (s *extractionState) shouldForceToolCall() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.successCount == 0 && s.failureCount < s.maxRetries
}

// retriesExhausted 判断重试次数是否已耗尽。
func (s *extractionState) retriesExhausted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failureCount >= s.maxRetries && s.successCount == 0
}

// lastSuccessfulTool 返回最后一次成功的工具调用结果。
func (s *extractionState) lastSuccessfulTool() (name, result string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.successCount > 0 {
		return s.lastToolName, s.lastToolTotal, true
	}
	return "", "", false
}
