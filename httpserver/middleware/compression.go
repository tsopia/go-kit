package middleware

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CompressionConfig 描述响应压缩行为。
type CompressionConfig struct {
	MinSizeBytes         int
	Level                int
	AllowedContentTypes  []string
	ExcludedContentTypes []string
	ShouldCompress       func(*gin.Context, int) bool
}

// Compression 在客户端声明支持 gzip 时压缩响应。
func Compression(configs ...CompressionConfig) gin.HandlerFunc {
	config := defaultCompressionConfig()
	if len(configs) > 0 {
		config = normalizeCompressionConfig(configs[0])
	}

	return func(c *gin.Context) {
		appendVaryHeader(c.Writer.Header(), "Accept-Encoding")

		if !acceptsGzip(c.Request) {
			c.Next()
			return
		}

		writer := newCompressionResponseWriter(c.Writer)
		c.Writer = writer

		c.Next()

		if writer.bypass {
			return
		}

		status := writer.Status()
		if status == 0 {
			status = http.StatusOK
		}
		if shouldSkipCompression(c, writer.Header(), status, config) {
			writer.flushOriginal()
			return
		}
		if config.ShouldCompress != nil && !config.ShouldCompress(c, status) {
			writer.flushOriginal()
			return
		}
		if writer.body.Len() < config.MinSizeBytes {
			writer.flushOriginal()
			return
		}

		contentType := writer.Header().Get("Content-Type")
		if !isCompressionContentTypeAllowed(contentType, config) {
			writer.flushOriginal()
			return
		}

		compressed, err := gzipBytes(writer.body.Bytes(), config.Level)
		if err != nil {
			writer.flushOriginal()
			return
		}

		headers := writer.Header()
		headers.Set("Content-Encoding", "gzip")
		appendVaryHeader(headers, "Accept-Encoding")
		headers.Del("Content-Length")

		writer.flushCompressed(compressed)
	}
}

type compressionResponseWriter struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
	bypass bool
}

func newCompressionResponseWriter(writer gin.ResponseWriter) *compressionResponseWriter {
	return &compressionResponseWriter{
		ResponseWriter: writer,
	}
}

func (w *compressionResponseWriter) WriteHeader(code int) {
	if w.bypass {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.status = code
}

func (w *compressionResponseWriter) WriteHeaderNow() {
	if w.bypass {
		w.ResponseWriter.WriteHeaderNow()
		return
	}
	w.beginPassthrough()
}

func (w *compressionResponseWriter) Write(data []byte) (int, error) {
	if w.bypass {
		return w.ResponseWriter.Write(data)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *compressionResponseWriter) WriteString(s string) (int, error) {
	if w.bypass {
		return w.ResponseWriter.WriteString(s)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.WriteString(s)
}

func (w *compressionResponseWriter) Status() int {
	if w.status != 0 {
		return w.status
	}
	return w.ResponseWriter.Status()
}

func (w *compressionResponseWriter) Size() int {
	if w.bypass {
		return w.ResponseWriter.Size()
	}
	if w.body.Len() > 0 {
		return w.body.Len()
	}
	return w.ResponseWriter.Size()
}

func (w *compressionResponseWriter) Written() bool {
	if w.bypass {
		return w.ResponseWriter.Written()
	}
	return w.status != 0 || w.body.Len() > 0 || w.ResponseWriter.Written()
}

func (w *compressionResponseWriter) flushOriginal() {
	if w.bypass {
		return
	}
	status := w.Status()
	if status == 0 {
		status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(status)
	if w.body.Len() == 0 {
		return
	}
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}

func (w *compressionResponseWriter) flushCompressed(body []byte) {
	if w.bypass {
		return
	}
	status := w.Status()
	if status == 0 {
		status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(status)
	if len(body) == 0 {
		return
	}
	_, _ = w.ResponseWriter.Write(body)
}

func (w *compressionResponseWriter) Flush() {
	if !w.bypass {
		w.beginPassthrough()
	}
	w.ResponseWriter.Flush()
}

func (w *compressionResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if !w.bypass {
		w.beginPassthrough()
	}
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("compression writer: hijacker not supported")
	}
	return hijacker.Hijack()
}

func (w *compressionResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *compressionResponseWriter) beginPassthrough() {
	if w.bypass {
		return
	}
	status := w.Status()
	if status == 0 {
		status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(status)
	if w.body.Len() > 0 {
		_, _ = w.ResponseWriter.Write(w.body.Bytes())
		w.body.Reset()
	}
	w.bypass = true
}

func defaultCompressionConfig() CompressionConfig {
	return normalizeCompressionConfig(CompressionConfig{})
}

func normalizeCompressionConfig(config CompressionConfig) CompressionConfig {
	if config.MinSizeBytes <= 0 {
		config.MinSizeBytes = 1024
	}
	if config.Level == 0 {
		config.Level = gzip.DefaultCompression
	}
	if len(config.AllowedContentTypes) == 0 {
		config.AllowedContentTypes = []string{
			"application/json",
			"text/plain",
			"text/html",
			"text/css",
			"application/javascript",
			"application/xml",
			"text/xml",
		}
	}
	if len(config.ExcludedContentTypes) == 0 {
		config.ExcludedContentTypes = []string{
			"text/event-stream",
			"image/",
			"video/",
			"audio/",
			"application/zip",
			"application/gzip",
			"application/pdf",
			"application/octet-stream",
		}
	}
	return config
}

func acceptsGzip(req *http.Request) bool {
	if req == nil {
		return false
	}
	return acceptsEncoding(req.Header.Get("Accept-Encoding"), "gzip")
}

func isCompressionContentTypeAllowed(contentType string, config CompressionConfig) bool {
	normalized := strings.ToLower(mediaType(contentType))
	if normalized == "" {
		return false
	}
	for _, excluded := range config.ExcludedContentTypes {
		if strings.HasPrefix(normalized, strings.ToLower(excluded)) {
			return false
		}
	}
	for _, allowed := range config.AllowedContentTypes {
		if strings.HasPrefix(normalized, strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}

func shouldSkipCompression(c *gin.Context, headers http.Header, status int, config CompressionConfig) bool {
	if c.Request.Method == http.MethodHead {
		return true
	}
	if (status >= 100 && status < 200) || status == http.StatusNoContent || status == http.StatusNotModified {
		return true
	}
	if headers.Get("Content-Encoding") != "" {
		return true
	}
	if isAttachmentResponse(headers) {
		return true
	}
	if strings.EqualFold(headers.Get("Connection"), "upgrade") || headers.Get("Upgrade") != "" {
		return true
	}
	return !isCompressionContentTypeAllowed(headers.Get("Content-Type"), config)
}

func acceptsEncoding(header string, want string) bool {
	if header == "" {
		return false
	}

	exactFound := false
	exactAllowed := false
	wildcardAllowed := false

	for _, token := range strings.Split(header, ",") {
		part := strings.TrimSpace(token)
		if part == "" {
			continue
		}

		name := part
		qValue := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			name = strings.TrimSpace(part[:semi])
			params := strings.Split(part[semi+1:], ";")
			for _, param := range params {
				param = strings.TrimSpace(param)
				if !strings.HasPrefix(strings.ToLower(param), "q=") {
					continue
				}
				parsed, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64)
				if err == nil {
					qValue = parsed
				}
				break
			}
		}

		switch {
		case strings.EqualFold(name, want):
			exactFound = true
			exactAllowed = qValue > 0
		case name == "*":
			wildcardAllowed = qValue > 0
		}
	}

	if exactFound {
		return exactAllowed
	}
	return wildcardAllowed
}

func isAttachmentResponse(headers http.Header) bool {
	return strings.Contains(strings.ToLower(headers.Get("Content-Disposition")), "attachment")
}

func appendVaryHeader(headers http.Header, value string) {
	current := headers.Get("Vary")
	if current == "" {
		headers.Set("Vary", value)
		return
	}
	for _, part := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return
		}
	}
	headers.Set("Vary", current+", "+value)
}

func gzipBytes(raw []byte, level int) ([]byte, error) {
	var buffer bytes.Buffer

	writer, err := gzip.NewWriterLevel(&buffer, level)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
