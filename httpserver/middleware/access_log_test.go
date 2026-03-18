package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

type capturedLog struct {
	level  string
	event  string
	fields map[string]any
}

type capturedLogger struct {
	mu     sync.Mutex
	events []capturedLog
}

func (l *capturedLogger) log(_ context.Context, level string, event string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cloned := make(map[string]any, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}

	l.events = append(l.events, capturedLog{
		level:  level,
		event:  event,
		fields: cloned,
	})
}

func (l *capturedLogger) snapshot() []capturedLog {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]capturedLog, len(l.events))
	copy(out, l.events)
	return out
}

func findLogByEvent(logs []capturedLog, event string) (capturedLog, bool) {
	for _, logEntry := range logs {
		if logEntry.event == event {
			return logEntry, true
		}
	}
	return capturedLog{}, false
}

func newMultipartRequest(t *testing.T, path string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	parts := []struct {
		field string
		value string
	}{
		{field: "title", value: "hello"},
		{field: "token", value: "secret-token"},
		{field: "metadata", value: `{"env":"staging"}`},
	}

	for _, part := range parts {
		if err := writer.WriteField(part.field, part.value); err != nil {
			t.Fatalf("write field %s: %v", part.field, err)
		}
	}

	fileWriter, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(fileWriter, strings.NewReader("PNGDATA-SECRET")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestAccessLogSummary(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger: logger.log,
	}))
	engine.Use(RequestID(), TraceID())
	engine.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.Header.Set(utils.TraceIDHeader, "trace-from-header")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want 1", len(logs))
	}

	logEntry := logs[0]
	if logEntry.event != "access_log" {
		t.Fatalf("event = %q, want %q", logEntry.event, "access_log")
	}
	if logEntry.level != "info" {
		t.Fatalf("level = %q, want %q", logEntry.level, "info")
	}

	testCases := []struct {
		key  string
		want any
	}{
		{key: "method", want: http.MethodGet},
		{key: "path", want: "/users/42"},
		{key: "route", want: "/users/:id"},
		{key: "status", want: http.StatusOK},
		{key: "trace_id", want: "trace-from-header"},
	}

	for _, tc := range testCases {
		if got := logEntry.fields[tc.key]; got != tc.want {
			t.Fatalf("%s = %#v, want %#v", tc.key, got, tc.want)
		}
	}

	requestID, _ := logEntry.fields["request_id"].(string)
	if requestID == "" {
		t.Fatalf("request_id is empty")
	}

	if _, ok := logEntry.fields["latency_ms"]; !ok {
		t.Fatalf("latency_ms missing from log fields")
	}
	if _, ok := logEntry.fields["bytes_out"]; !ok {
		t.Fatalf("bytes_out missing from log fields")
	}
}

func TestAccessLogErrorLevel(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{name: "success", status: http.StatusCreated, wantLevel: "info"},
		{name: "client_error", status: http.StatusBadRequest, wantLevel: "info"},
		{name: "server_error", status: http.StatusInternalServerError, wantLevel: "error"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := &capturedLogger{}

			engine := gin.New()
			engine.Use(AccessLog(AccessLogConfig{
				Logger: logger.log,
			}))
			engine.GET("/status", func(c *gin.Context) {
				c.JSON(tc.status, gin.H{"status": tc.status})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/status", nil)
			engine.ServeHTTP(w, req)

			logs := logger.snapshot()
			if len(logs) != 1 {
				t.Fatalf("log count = %d, want 1", len(logs))
			}

			if got := logs[0].level; got != tc.wantLevel {
				t.Fatalf("level = %q, want %q", got, tc.wantLevel)
			}
		})
	}
}

func TestAccessLogPayloadJSON(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   1024,
	}))
	engine.POST("/login", func(c *gin.Context) {
		var body map[string]any
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":         body["email"],
			"access_token": "server-token",
		})
	})

	reqBody := []byte(`{"email":"alice@example.com","password":"super-secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	if payloadLog.level != "debug" {
		t.Fatalf("payload level = %q, want %q", payloadLog.level, "debug")
	}

	requestBody, ok := payloadLog.fields["request_body"].(map[string]any)
	if !ok {
		t.Fatalf("request_body type = %T, want map[string]any", payloadLog.fields["request_body"])
	}
	if got := requestBody["password"]; got != "***" {
		t.Fatalf("password = %#v, want %#v", got, "***")
	}
	if got := requestBody["email"]; got == "alice@example.com" {
		t.Fatalf("email was not masked: %#v", got)
	}

	responseBody, ok := payloadLog.fields["response_body"].(map[string]any)
	if !ok {
		t.Fatalf("response_body type = %T, want map[string]any", payloadLog.fields["response_body"])
	}
	if got := responseBody["access_token"]; got != "***" {
		t.Fatalf("access_token = %#v, want %#v", got, "***")
	}
}

func TestAccessLogPayloadForm(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   1024,
	}))
	engine.POST("/submit", func(c *gin.Context) {
		if err := c.Request.ParseForm(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewBufferString("email=alice%40example.com&password=s3cret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	requestBody, ok := payloadLog.fields["request_body"].(map[string]any)
	if !ok {
		t.Fatalf("request_body type = %T, want map[string]any", payloadLog.fields["request_body"])
	}
	if got := requestBody["password"]; got != "***" {
		t.Fatalf("password = %#v, want %#v", got, "***")
	}
	if got := requestBody["email"]; got == "alice@example.com" {
		t.Fatalf("email was not masked: %#v", got)
	}
}

func TestAccessLogPayloadTruncated(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   8,
	}))
	engine.POST("/echo", func(c *gin.Context) {
		payload := gin.H{"value": "0123456789abcdef"}
		c.JSON(http.StatusOK, payload)
	})

	reqBody := bytes.NewBufferString(`{"value":"0123456789abcdef"}`)
	req := httptest.NewRequest(http.MethodPost, "/echo", reqBody)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	requestTruncated, ok := payloadLog.fields["request_truncated"].(bool)
	if !ok || !requestTruncated {
		t.Fatalf("request_truncated = %#v, want true", payloadLog.fields["request_truncated"])
	}
	responseTruncated, ok := payloadLog.fields["response_truncated"].(bool)
	if !ok || !responseTruncated {
		t.Fatalf("response_truncated = %#v, want true", payloadLog.fields["response_truncated"])
	}

	responseBodyJSON, ok := payloadLog.fields["response_body"].(string)
	if !ok {
		raw, _ := json.Marshal(payloadLog.fields["response_body"])
		t.Fatalf("response_body type = %T, want string (raw=%s)", payloadLog.fields["response_body"], raw)
	}
	if len(responseBodyJSON) != 8 {
		t.Fatalf("response_body length = %d, want 8", len(responseBodyJSON))
	}
}

func TestAccessLogMultipartMetadataOnly(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   1024,
		Multipart: MultipartConfig{
			Mode: MultipartMetadataOnly,
		},
	}))
	engine.POST("/upload", func(c *gin.Context) {
		if _, _, err := c.Request.FormFile("file"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := newMultipartRequest(t, "/upload")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	requestBody, ok := payloadLog.fields["request_body"].(map[string]any)
	if !ok {
		t.Fatalf("request_body type = %T, want map[string]any", payloadLog.fields["request_body"])
	}

	fieldNames, ok := requestBody["field_names"].([]string)
	if !ok {
		t.Fatalf("field_names type = %T, want []string", requestBody["field_names"])
	}
	if len(fieldNames) != 3 {
		t.Fatalf("field_names length = %d, want 3", len(fieldNames))
	}

	files, ok := requestBody["files"].([]map[string]any)
	if !ok {
		t.Fatalf("files type = %T, want []map[string]any", requestBody["files"])
	}
	if len(files) != 1 {
		t.Fatalf("files length = %d, want 1", len(files))
	}
	if files[0]["field"] != "file" {
		t.Fatalf("file field = %#v, want %#v", files[0]["field"], "file")
	}

	if strings.Contains(payloadLog.fields["request_body"].(map[string]any)["files"].([]map[string]any)[0]["filename"].(string), "PNGDATA-SECRET") {
		t.Fatalf("file content leaked into metadata")
	}
}

func TestAccessLogMultipartFormFieldsOnly(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   1024,
		Multipart: MultipartConfig{
			Mode: MultipartFormFieldsOnly,
		},
	}))
	engine.POST("/upload", func(c *gin.Context) {
		if _, _, err := c.Request.FormFile("file"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := newMultipartRequest(t, "/upload")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	requestBody, ok := payloadLog.fields["request_body"].(map[string]any)
	if !ok {
		t.Fatalf("request_body type = %T, want map[string]any", payloadLog.fields["request_body"])
	}
	fields, ok := requestBody["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields type = %T, want map[string]any", requestBody["fields"])
	}
	if got := fields["title"]; got != "hello" {
		t.Fatalf("title = %#v, want %#v", got, "hello")
	}
	if got := fields["token"]; got != "***" {
		t.Fatalf("token = %#v, want %#v", got, "***")
	}
}

func TestAccessLogMultipartSelectedParts(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	logger := &capturedLogger{}

	engine := gin.New()
	engine.Use(AccessLog(AccessLogConfig{
		Logger:         logger.log,
		CapturePayload: true,
		MaxBodyBytes:   1024,
		Multipart: MultipartConfig{
			Mode:          MultipartSelectedParts,
			PartAllowlist: []string{"metadata"},
		},
	}))
	engine.POST("/upload", func(c *gin.Context) {
		if _, _, err := c.Request.FormFile("file"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := newMultipartRequest(t, "/upload")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	logs := logger.snapshot()
	payloadLog, ok := findLogByEvent(logs, "payload_log")
	if !ok {
		t.Fatalf("payload_log not found")
	}

	requestBody, ok := payloadLog.fields["request_body"].(map[string]any)
	if !ok {
		t.Fatalf("request_body type = %T, want map[string]any", payloadLog.fields["request_body"])
	}
	fields, ok := requestBody["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields type = %T, want map[string]any", requestBody["fields"])
	}
	if _, exists := fields["title"]; exists {
		t.Fatalf("title should not be captured in selected parts mode")
	}
	if _, exists := fields["token"]; exists {
		t.Fatalf("token should not be captured in selected parts mode")
	}
	if got := fields["metadata"]; got != `{"env":"staging"}` {
		t.Fatalf("metadata = %#v, want %#v", got, `{"env":"staging"}`)
	}
}
