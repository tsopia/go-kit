package httpserver

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	httpmiddleware "github.com/tsopia/go-kit/httpserver/middleware"
	"github.com/tsopia/go-kit/utils"
)

func logStreamEvent(c *gin.Context, level string, event string, transport string, startedAt time.Time, err error) {
	if c == nil || c.Request == nil {
		return
	}

	config, _ := httpmiddleware.StreamLogConfigFromContext(c.Request.Context())
	logger := config.Logger
	if logger == nil {
		logger = httpmiddleware.DefaultLogger
	}

	fields := map[string]any{
		"transport":   transport,
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"route":       streamRoute(c),
		"client_ip":   streamClientIP(c.Request),
		"host":        c.Request.Host,
		"user_agent":  c.Request.UserAgent(),
		"request_id":  utils.RequestIDFromContext(c.Request.Context()),
		"trace_id":    utils.TraceIDFromContext(c.Request.Context()),
		"remote_addr": c.Request.RemoteAddr,
	}

	if !startedAt.IsZero() {
		fields["duration_ms"] = time.Since(startedAt).Milliseconds()
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	if headers := httpmiddleware.CaptureAllowedRequestHeaders(c.Request.Header, config.AllowedRequestHeaders); len(headers) > 0 {
		fields["request_headers"] = headers
	}

	logger(c.Request.Context(), level, event, fields)
}

func streamRoute(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	if c.Request != nil && c.Request.URL != nil {
		return c.Request.URL.Path
	}
	return ""
}

func streamClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if clientIP := utils.ClientIPFromContext(r.Context()); clientIP != "" {
		return clientIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
