package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

const (
	accessLogEvent  = "access_log"
	payloadLogEvent = "payload_log"
)

// LoggerFunc 抽象访问日志输出，避免 middleware 依赖具体日志实现。
type LoggerFunc func(ctx context.Context, level string, event string, fields map[string]any)

// AccessLogConfig 描述访问日志行为。
type AccessLogConfig struct {
	Logger                 LoggerFunc
	CapturePayload         bool
	PayloadLogLevel        string
	MaxBodyBytes           int64
	AllowedContentTypes    []string
	AllowedRequestHeaders  []string
	AllowedResponseHeaders []string
	ShouldCapturePayload   func(*gin.Context, int) bool
	Multipart              MultipartConfig
	Redaction              RedactionConfig

	// SkipPaths 精确匹配路径跳过日志记录。
	// 示例: []string{"/health", "/readyz", "/livez"}
	SkipPaths []string

	// ShouldLog 完全自定义的日志过滤。
	// 返回 false 时跳过全部日志。
	ShouldLog func(*gin.Context) bool
}

// MultipartCaptureMode 控制 multipart payload 的捕获模式。
type MultipartCaptureMode string

const (
	MultipartDisabled       MultipartCaptureMode = "disabled"
	MultipartMetadataOnly   MultipartCaptureMode = "metadata_only"
	MultipartFormFieldsOnly MultipartCaptureMode = "form_fields_only"
	MultipartSelectedParts  MultipartCaptureMode = "selected_parts"
)

// MultipartConfig 描述 multipart 载荷日志行为。
type MultipartConfig struct {
	Mode              MultipartCaptureMode
	PartAllowlist     []string
	MaxPartValueBytes int64
	RedactFilenames   bool
}

// RedactionStrategy 描述脱敏策略。
type RedactionStrategy string

const (
	RedactionRedact RedactionStrategy = "redact"
	RedactionMask   RedactionStrategy = "mask"
	RedactionHash   RedactionStrategy = "hash"
	RedactionDrop   RedactionStrategy = "drop"
)

// RedactionRule 描述一个 scope + key 的脱敏规则。
type RedactionRule struct {
	Scope    string
	Key      string
	Strategy RedactionStrategy
}

// RedactionConfig 描述 payload 脱敏行为。
type RedactionConfig struct {
	Rules         []RedactionRule
	MaxValueBytes int
	HashSalt      string
}

// AccessLog 记录结构化访问日志。
func AccessLog(configs ...AccessLogConfig) gin.HandlerFunc {
	config := defaultAccessLogConfig()
	if len(configs) > 0 {
		config = normalizeAccessLogConfig(configs[0])
	}

	skipSet := make(map[string]struct{}, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipSet[path] = struct{}{}
	}

	return func(c *gin.Context) {
		if _, skip := skipSet[c.Request.URL.Path]; skip {
			c.Next()
			return
		}
		if config.ShouldLog != nil && !config.ShouldLog(c) {
			c.Next()
			return
		}

		startedAt := time.Now()

		requestContentType := mediaType(c.Request.Header.Get("Content-Type"))
		requestBodyCapture := prepareRequestBodyCapture(c, requestContentType, config)
		c.Request.Body = requestBodyCapture

		ctx := WithStreamLogConfig(c.Request.Context(), StreamLogConfig{
			Logger:                config.Logger,
			AllowedRequestHeaders: config.AllowedRequestHeaders,
		})
		c.Request = c.Request.WithContext(ctx)

		responseBodyCapture := newCaptureResponseWriter(c.Writer, config.MaxBodyBytes)
		c.Writer = responseBodyCapture

		c.Next()

		if c.GetString(utils.StreamingKey) != "" {
			return
		}

		status := statusCode(c.Writer.Status())
		fields := map[string]any{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"route":      fullRoute(c),
			"status":     status,
			"latency_ms": time.Since(startedAt).Milliseconds(),
			"client_ip":  clientIPFromContext(c),
			"host":       c.Request.Host,
			"user_agent": c.Request.UserAgent(),
			"referer":    c.Request.Referer(),
			"request_id": requestIDFromContext(c),
			"trace_id":   traceIDFromContext(c),
			"bytes_in":   requestBytesIn(c.Request),
			"bytes_out":  responseBytesOut(c.Writer),
		}

		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
		}

		config.Logger(c.Request.Context(), accessLogLevel(status), accessLogEvent, fields)

		payloadFields, ok := buildPayloadFields(c, config, requestBodyCapture, responseBodyCapture, status)
		if ok {
			config.Logger(c.Request.Context(), config.PayloadLogLevel, payloadLogEvent, payloadFields)
		}
	}
}

func defaultAccessLogConfig() AccessLogConfig {
	return normalizeAccessLogConfig(AccessLogConfig{})
}

func normalizeAccessLogConfig(config AccessLogConfig) AccessLogConfig {
	if config.Logger == nil {
		config.Logger = defaultAccessLogger
	}
	if config.PayloadLogLevel == "" {
		config.PayloadLogLevel = "debug"
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = 8 << 10
	}
	if len(config.AllowedContentTypes) == 0 {
		config.AllowedContentTypes = []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"text/plain",
			"multipart/form-data",
		}
	}
	if config.Multipart.Mode == "" {
		config.Multipart.Mode = MultipartMetadataOnly
	}
	if config.Multipart.MaxPartValueBytes <= 0 {
		config.Multipart.MaxPartValueBytes = config.MaxBodyBytes
	}
	if config.Redaction.MaxValueBytes <= 0 {
		config.Redaction.MaxValueBytes = 512
	}
	if len(config.Redaction.Rules) == 0 {
		config.Redaction.Rules = defaultRedactionRules()
	}
	return config
}

func defaultAccessLogger(ctx context.Context, level string, event string, fields map[string]any) {
	args := make([]any, 0, len(fields)*2)
	for key, value := range fields {
		args = append(args, key, value)
	}
	slog.Default().Log(ctx, slogLevel(level), event, args...)
}

func slogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func fullRoute(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return c.Request.URL.Path
}

func statusCode(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

func accessLogLevel(status int) string {
	if status >= http.StatusInternalServerError {
		return "error"
	}
	return "info"
}

func requestBytesIn(req *http.Request) int64 {
	if req == nil || req.ContentLength < 0 {
		return 0
	}
	return req.ContentLength
}

func responseBytesOut(writer gin.ResponseWriter) int {
	if writer == nil {
		return 0
	}
	if size := writer.Size(); size > 0 {
		return size
	}
	return 0
}

func traceIDFromContext(c *gin.Context) string {
	if traceID, exists := c.Get(utils.TraceIDKey); exists {
		if value, ok := traceID.(string); ok {
			return value
		}
	}
	return c.Writer.Header().Get(utils.TraceIDHeader)
}

func requestIDFromContext(c *gin.Context) string {
	if requestID, exists := c.Get(utils.RequestIDKey); exists {
		if value, ok := requestID.(string); ok {
			return value
		}
	}
	return c.Writer.Header().Get(utils.RequestIDHeader)
}

type bodyCaptureReader struct {
	io.ReadCloser
	maxBytes  int64
	buffer    bytes.Buffer
	truncated bool
	preloaded bool
}

func newBodyCaptureReader(body io.ReadCloser, maxBytes int64) *bodyCaptureReader {
	if body == nil {
		body = http.NoBody
	}
	return &bodyCaptureReader{
		ReadCloser: body,
		maxBytes:   maxBytes,
	}
}

func newPreloadedBodyCaptureReader(raw []byte, maxBytes int64) *bodyCaptureReader {
	reader := &bodyCaptureReader{
		ReadCloser: io.NopCloser(bytes.NewReader(raw)),
		maxBytes:   maxBytes,
		preloaded:  true,
	}
	reader.capture(raw)
	return reader
}

func (r *bodyCaptureReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && !r.preloaded {
		r.capture(p[:n])
	}
	return n, err
}

func (r *bodyCaptureReader) capture(data []byte) {
	if r.maxBytes <= 0 {
		return
	}

	remaining := r.maxBytes - int64(r.buffer.Len())
	if remaining <= 0 {
		r.truncated = true
		return
	}

	toWrite := data
	if int64(len(toWrite)) > remaining {
		toWrite = toWrite[:remaining]
		r.truncated = true
	}

	_, _ = r.buffer.Write(toWrite)
	if int64(len(data)) > remaining {
		r.truncated = true
	}
}

func (r *bodyCaptureReader) Bytes() []byte {
	return r.buffer.Bytes()
}

func prepareRequestBodyCapture(c *gin.Context, contentType string, config AccessLogConfig) *bodyCaptureReader {
	if !config.CapturePayload || !shouldPreloadRequestBody(contentType) {
		return newBodyCaptureReader(c.Request.Body, config.MaxBodyBytes)
	}

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		captureErr := c.Error(fmt.Errorf("read request body for access log: %w", err))
		if captureErr != nil {
			captureErr.Type = gin.ErrorTypePrivate
		}
	}
	return newPreloadedBodyCaptureReader(raw, config.MaxBodyBytes)
}

type captureResponseWriter struct {
	gin.ResponseWriter
	maxBytes  int64
	buffer    bytes.Buffer
	truncated bool
}

func newCaptureResponseWriter(writer gin.ResponseWriter, maxBytes int64) *captureResponseWriter {
	return &captureResponseWriter{
		ResponseWriter: writer,
		maxBytes:       maxBytes,
	}
}

func (w *captureResponseWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *captureResponseWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *captureResponseWriter) capture(data []byte) {
	if w.maxBytes <= 0 {
		return
	}

	remaining := w.maxBytes - int64(w.buffer.Len())
	if remaining <= 0 {
		w.truncated = true
		return
	}

	toWrite := data
	if int64(len(toWrite)) > remaining {
		toWrite = toWrite[:remaining]
		w.truncated = true
	}

	_, _ = w.buffer.Write(toWrite)
	if int64(len(data)) > remaining {
		w.truncated = true
	}
}

func (w *captureResponseWriter) Bytes() []byte {
	return w.buffer.Bytes()
}

func buildPayloadFields(c *gin.Context, config AccessLogConfig, requestCapture *bodyCaptureReader, responseCapture *captureResponseWriter, status int) (map[string]any, bool) {
	if !config.CapturePayload {
		return nil, false
	}
	if config.ShouldCapturePayload != nil && !config.ShouldCapturePayload(c, status) {
		return nil, false
	}

	requestContentType := mediaType(c.Request.Header.Get("Content-Type"))
	responseContentType := mediaType(c.Writer.Header().Get("Content-Type"))

	fields := map[string]any{
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"route":      fullRoute(c),
		"status":     status,
		"request_id": requestIDFromContext(c),
		"trace_id":   traceIDFromContext(c),
	}

	if requestContentType == "multipart/form-data" {
		if body, ok := sanitizeMultipartRequest(c.Request, config); ok {
			fields["request_body"] = body
		}
	} else if body, ok := sanitizePayloadBody("request_body", requestContentType, requestCapture.Bytes(), requestCapture.truncated, config); ok {
		fields["request_body"] = body
	}
	if requestCapture.truncated {
		fields["request_truncated"] = true
	}

	if body, ok := sanitizePayloadBody("response_body", responseContentType, responseCapture.Bytes(), responseCapture.truncated, config); ok {
		fields["response_body"] = body
	}
	if responseCapture.truncated {
		fields["response_truncated"] = true
	}

	if headers := captureHeaders(c.Request.Header, config.AllowedRequestHeaders, "request_header", config); len(headers) > 0 {
		fields["request_headers"] = headers
	}
	if headers := captureHeaders(c.Writer.Header(), config.AllowedResponseHeaders, "response_header", config); len(headers) > 0 {
		fields["response_headers"] = headers
	}

	if len(fields) == 6 {
		return nil, false
	}

	return fields, true
}

func sanitizePayloadBody(scope string, contentType string, raw []byte, truncated bool, config AccessLogConfig) (any, bool) {
	if len(raw) == 0 || !isAllowedContentType(contentType, config.AllowedContentTypes) {
		return nil, false
	}

	switch contentType {
	case "application/json":
		if truncated {
			return string(raw), true
		}

		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return string(raw), true
		}
		return sanitizeStructuredValue(scope, "", decoded, config)
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(raw))
		if err != nil {
			return string(raw), true
		}
		return sanitizeFormValues(scope, values, config), true
	default:
		return string(raw), true
	}
}

func sanitizeMultipartRequest(req *http.Request, config AccessLogConfig) (map[string]any, bool) {
	if config.Multipart.Mode == MultipartDisabled || req == nil {
		return nil, false
	}
	if req.MultipartForm == nil {
		if err := req.ParseMultipartForm(config.Multipart.MaxPartValueBytes); err != nil {
			return nil, false
		}
	}
	if req.MultipartForm == nil {
		return nil, false
	}

	result := make(map[string]any)

	switch config.Multipart.Mode {
	case MultipartMetadataOnly:
		fieldNames := multipartFieldNames(req.MultipartForm.Value)
		if len(fieldNames) > 0 {
			result["field_names"] = fieldNames
		}
	case MultipartFormFieldsOnly:
		fields := sanitizeMultipartFields(req.MultipartForm.Value, nil, config)
		if len(fields) > 0 {
			result["fields"] = fields
		}
	case MultipartSelectedParts:
		fields := sanitizeMultipartFields(req.MultipartForm.Value, config.Multipart.PartAllowlist, config)
		if len(fields) > 0 {
			result["fields"] = fields
		}
	default:
		fieldNames := multipartFieldNames(req.MultipartForm.Value)
		if len(fieldNames) > 0 {
			result["field_names"] = fieldNames
		}
	}

	files := sanitizeMultipartFiles(req.MultipartForm.File, config)
	if len(files) > 0 {
		result["files"] = files
	}
	if len(result) == 0 {
		return nil, false
	}
	return result, true
}

func sanitizeStructuredValue(scope string, key string, value any, config AccessLogConfig) (any, bool) {
	if key != "" {
		switch strategyFor(scope, key, config.Redaction.Rules) {
		case RedactionRedact:
			return "***", true
		case RedactionMask:
			return maskValue(key, value, config.Redaction.MaxValueBytes), true
		case RedactionHash:
			return hashValue(value, config.Redaction.HashSalt), true
		case RedactionDrop:
			return nil, false
		}
	}

	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			sanitized, ok := sanitizeStructuredValue(scope, childKey, childValue, config)
			if ok {
				result[childKey] = sanitized
			}
		}
		return result, true
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitized, ok := sanitizeStructuredValue(scope, "", item, config)
			if ok {
				result = append(result, sanitized)
			}
		}
		return result, true
	default:
		return trimValue(value, config.Redaction.MaxValueBytes), true
	}
}

func sanitizeFormValues(scope string, values url.Values, config AccessLogConfig) map[string]any {
	result := make(map[string]any, len(values))
	for key, items := range values {
		if len(items) == 1 {
			sanitized, ok := sanitizeStructuredValue(scope, key, items[0], config)
			if ok {
				result[key] = sanitized
			}
			continue
		}

		arr := make([]any, 0, len(items))
		for _, item := range items {
			sanitized, ok := sanitizeStructuredValue(scope, key, item, config)
			if ok {
				arr = append(arr, sanitized)
			}
		}
		result[key] = arr
	}
	return result
}

func sanitizeMultipartFields(values map[string][]string, allowlist []string, config AccessLogConfig) map[string]any {
	result := make(map[string]any)
	allowed := make(map[string]struct{}, len(allowlist))
	for _, item := range allowlist {
		allowed[strings.ToLower(item)] = struct{}{}
	}

	for key, items := range values {
		if len(allowlist) > 0 {
			if _, ok := allowed[strings.ToLower(key)]; !ok {
				continue
			}
		}

		if len(items) == 1 {
			sanitized, ok := sanitizeStructuredValue("request_body", key, items[0], config)
			if ok {
				result[key] = sanitized
			}
			continue
		}

		arr := make([]any, 0, len(items))
		for _, item := range items {
			sanitized, ok := sanitizeStructuredValue("request_body", key, item, config)
			if ok {
				arr = append(arr, sanitized)
			}
		}
		result[key] = arr
	}
	return result
}

func sanitizeMultipartFiles(files map[string][]*multipart.FileHeader, config AccessLogConfig) []map[string]any {
	result := make([]map[string]any, 0)
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, file := range files[key] {
			result = append(result, map[string]any{
				"field":        key,
				"filename":     maskedFilename(file.Filename, config),
				"content_type": file.Header.Get("Content-Type"),
				"size":         file.Size,
			})
		}
	}
	return result
}

func multipartFieldNames(values map[string][]string) []string {
	fieldNames := make([]string, 0, len(values))
	for key := range values {
		fieldNames = append(fieldNames, key)
	}
	sort.Strings(fieldNames)
	return fieldNames
}

func captureHeaders(headers http.Header, allowlist []string, scope string, config AccessLogConfig) map[string]any {
	if len(allowlist) == 0 {
		return nil
	}

	result := make(map[string]any, len(allowlist))
	for _, headerName := range allowlist {
		values := headers.Values(headerName)
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			sanitized, ok := sanitizeStructuredValue(scope, headerName, values[0], config)
			if ok {
				result[headerName] = sanitized
			}
			continue
		}

		arr := make([]any, 0, len(values))
		for _, value := range values {
			sanitized, ok := sanitizeStructuredValue(scope, headerName, value, config)
			if ok {
				arr = append(arr, sanitized)
			}
		}
		result[headerName] = arr
	}
	return result
}

func mediaType(contentType string) string {
	if contentType == "" {
		return ""
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return contentType
	}
	return mediaType
}

func isAllowedContentType(contentType string, allowlist []string) bool {
	if contentType == "" {
		return false
	}

	for _, allowed := range allowlist {
		if allowed == contentType {
			return true
		}
	}
	return false
}

func shouldPreloadRequestBody(contentType string) bool {
	switch contentType {
	case "application/json", "application/x-www-form-urlencoded", "text/plain":
		return true
	default:
		return false
	}
}

func defaultRedactionRules() []RedactionRule {
	return []RedactionRule{
		{Key: "authorization", Strategy: RedactionRedact},
		{Key: "cookie", Strategy: RedactionRedact},
		{Key: "set-cookie", Strategy: RedactionRedact},
		{Key: "password", Strategy: RedactionRedact},
		{Key: "token", Strategy: RedactionRedact},
		{Key: "access_token", Strategy: RedactionRedact},
		{Key: "refresh_token", Strategy: RedactionRedact},
		{Key: "secret", Strategy: RedactionRedact},
		{Key: "client_secret", Strategy: RedactionRedact},
		{Key: "private_key", Strategy: RedactionRedact},
		{Key: "phone", Strategy: RedactionMask},
		{Key: "mobile", Strategy: RedactionMask},
		{Key: "email", Strategy: RedactionMask},
		{Key: "id_card", Strategy: RedactionMask},
		{Key: "bank_card", Strategy: RedactionMask},
		{Key: "filename", Strategy: RedactionMask},
	}
}

func strategyFor(scope string, key string, rules []RedactionRule) RedactionStrategy {
	lowerScope := strings.ToLower(scope)
	lowerKey := strings.ToLower(key)

	for _, rule := range rules {
		if rule.Scope != "" && strings.ToLower(rule.Scope) != lowerScope {
			continue
		}
		if strings.EqualFold(rule.Key, lowerKey) {
			return rule.Strategy
		}
	}
	return ""
}

func maskValue(key string, value any, maxValueBytes int) any {
	text := stringifyValue(value)
	lowerKey := strings.ToLower(key)

	switch lowerKey {
	case "email":
		return maskEmail(text)
	case "phone", "mobile":
		return maskPhone(text)
	default:
		return trimText(text, maxValueBytes)
	}
}

func hashValue(value any, salt string) string {
	sum := sha256.Sum256([]byte(salt + stringifyValue(value)))
	return hex.EncodeToString(sum[:8])
}

func trimValue(value any, maxValueBytes int) any {
	if maxValueBytes <= 0 {
		return value
	}
	if text, ok := value.(string); ok {
		return trimText(text, maxValueBytes)
	}
	return value
}

func trimText(value string, maxValueBytes int) string {
	if maxValueBytes <= 0 || len(value) <= maxValueBytes {
		return value
	}
	return value[:maxValueBytes]
}

func maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return "***"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func maskedFilename(filename string, config AccessLogConfig) string {
	if !config.Multipart.RedactFilenames {
		return filename
	}

	sanitized, ok := sanitizeStructuredValue("request_body", "filename", filename, config)
	if !ok {
		return ""
	}
	if value, ok := sanitized.(string); ok {
		return value
	}
	return stringifyValue(sanitized)
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}
