package kit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// webhookManager Webhook 管理器
type webhookManager struct {
	webhooks []*webhook
	client   *http.Client
}

// webhook 内部 webhook 结构
type webhook struct {
	config WebhookConfig
}

// newWebhookManager 创建 webhook 管理器
func newWebhookManager(configs []*WebhookConfig) *webhookManager {
	if len(configs) == 0 {
		return nil
	}

	wm := &webhookManager{
		webhooks: make([]*webhook, 0, len(configs)),
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	for _, cfg := range configs {
		// 设置默认值
		if cfg.Method == "" {
			cfg.Method = "POST"
		}
		if cfg.Timeout == 0 {
			cfg.Timeout = 5 * time.Second
		}
		if cfg.Filter == nil {
			// 默认只发送 Error 及以上级别
			cfg.Filter = defaultWebhookFilter
		}

		wm.webhooks = append(wm.webhooks, &webhook{config: *cfg})
	}

	return wm
}

// defaultWebhookFilter 默认 webhook 过滤器（Error 及以上级别）
func defaultWebhookFilter(_ context.Context, record LogRecord) bool {
	return record.Level >= ErrorLevel
}

// send 发送 webhook 通知
func (wm *webhookManager) send(ctx context.Context, record LogRecord) {
	if wm == nil {
		return
	}

	for _, wh := range wm.webhooks {
		// 检查过滤条件
		if wh.config.Filter != nil && !wh.config.Filter(ctx, record) {
			continue
		}

		// 异步发送，不阻塞日志记录
		go wm.sendWebhook(ctx, wh, record)
	}
}

// sendWebhook 发送单个 webhook
func (wm *webhookManager) sendWebhook(ctx context.Context, wh *webhook, record LogRecord) {
	// 构建 payload
	var payload map[string]interface{}
	if wh.config.BuildPayload != nil {
		payload = wh.config.BuildPayload(ctx, record)
	} else {
		payload = wm.buildDefaultPayload(record)
	}

	// 序列化
	body, err := json.Marshal(payload)
	if err != nil {
		// webhook 发送失败不应影响主流程，仅打印到 stderr
		fmt.Fprintf(placeholderWriter{}, "webhook marshal error: %v\n", err)
		return
	}

	// 创建请求
	reqCtx, cancel := context.WithTimeout(ctx, wh.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, wh.config.Method, wh.config.URL, bytes.NewReader(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wh.config.Headers {
		req.Header.Set(k, v)
	}

	// 发送请求
	resp, err := wm.client.Do(req)
	if err != nil {
		// 静默失败，不阻塞日志记录
		return
	}
	defer resp.Body.Close()
}

// buildDefaultPayload 构建默认 payload
func (wm *webhookManager) buildDefaultPayload(record LogRecord) map[string]interface{} {
	payload := map[string]interface{}{
		"level":     record.Level.String(),
		"message":   record.Message,
		"time":      record.Time.Format(time.RFC3339),
		"caller":    record.Caller,
		"trace_id":  record.TraceID,
		"request_id": record.RequestID,
	}

	if record.UserID != "" {
		payload["user_id"] = record.UserID
	}

	if record.StackTrace != "" {
		payload["stack_trace"] = record.StackTrace
	}

	if len(record.Fields) > 0 {
		for k, v := range record.Fields {
			payload[k] = v
		}
	}

	return payload
}

// placeholderWriter 占位 writer，用于避免循环依赖
type placeholderWriter struct{}

func (placeholderWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
