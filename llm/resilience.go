package llm

import (
	"context"
	"errors"
	"strings"
)

// isTransientModelError 判断模型调用错误是否值得重试。
// 重试：429/502/503、rate limit、too many requests。
// 不重试：context 取消/超时、4xx 客户端错误、nil。
func isTransientModelError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	s := strings.ToLower(err.Error())
	// 明确排除客户端错误（400/401/403/404）
	for _, code := range []string{"400 ", "401 ", "403 ", "404 "} {
		if strings.Contains(s, code) {
			return false
		}
	}
	return strings.Contains(s, "429") ||
		strings.Contains(s, "502") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "too many requests")
}
