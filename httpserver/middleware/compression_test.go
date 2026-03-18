package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCompressionNegotiation(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
	}{
		{
			name:           "client does not accept gzip",
			acceptEncoding: "",
			wantEncoding:   "",
		},
		{
			name:           "client accepts gzip",
			acceptEncoding: "gzip, deflate",
			wantEncoding:   "gzip",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Compression(CompressionConfig{
				MinSizeBytes: 1,
			}))
			engine.GET("/data", func(c *gin.Context) {
				c.Data(http.StatusOK, "application/json", []byte(`{"message":"hello compression"}`))
			})

			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			if tc.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tc.acceptEncoding)
			}

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if got := w.Header().Get("Content-Encoding"); got != tc.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, tc.wantEncoding)
			}
		})
	}
}

func TestCompressionRoundTrip(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	originalBody := strings.Repeat(`{"status":"active","department":"sales"}`, 16)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/users", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(originalBody))
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}

	decompressedBody := mustReadGzipBody(t, w.Body.Bytes())
	if decompressedBody != originalBody {
		t.Fatalf("decompressed body = %q, want %q", decompressedBody, originalBody)
	}
}

func TestCompressionSkipsSmallBodies(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 64,
	}))
	engine.GET("/small", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(`{"ok":true}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/small", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if got := w.Body.String(); got != `{"ok":true}` {
		t.Fatalf("body = %q, want %q", got, `{"ok":true}`)
	}
}

func TestCompressionSkipsStatuses(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "head request", method: http.MethodHead, path: "/head", status: http.StatusOK},
		{name: "204 response", method: http.MethodGet, path: "/no-content", status: http.StatusNoContent},
		{name: "304 response", method: http.MethodGet, path: "/not-modified", status: http.StatusNotModified},
	}

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/head", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 16)))
	})
	engine.GET("/no-content", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	engine.GET("/not-modified", func(c *gin.Context) {
		c.Status(http.StatusNotModified)
	})

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding = %q, want empty", got)
			}
		})
	}
}

func TestCompressionSkipsContentTypes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		contentType string
		headers     map[string]string
	}{
		{name: "sse", contentType: "text/event-stream"},
		{name: "image", contentType: "image/png"},
		{name: "pdf", contentType: "application/pdf"},
		{
			name:        "pre-encoded",
			contentType: "application/json",
			headers: map[string]string{
				"Content-Encoding": "br",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Compression(CompressionConfig{
				MinSizeBytes: 1,
			}))
			engine.GET("/data", func(c *gin.Context) {
				for key, value := range tc.headers {
					c.Header(key, value)
				}
				c.Data(http.StatusOK, tc.contentType, []byte(strings.Repeat("compress-me-", 32)))
			})

			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			wantEncoding := tc.headers["Content-Encoding"]
			if got := w.Header().Get("Content-Encoding"); got != wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, wantEncoding)
			}
		})
	}
}

func TestCompressionShouldCompress(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
		ShouldCompress: func(*gin.Context, int) bool {
			return false
		},
	}))
	engine.GET("/data", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 32)))
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressionCustomContentTypes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		config      CompressionConfig
		contentType string
		wantGzip    bool
	}{
		{
			name: "custom allowed content type",
			config: CompressionConfig{
				MinSizeBytes:        1,
				AllowedContentTypes: []string{"application/vnd.api+json"},
			},
			contentType: "application/vnd.api+json",
			wantGzip:    true,
		},
		{
			name: "custom excluded content type",
			config: CompressionConfig{
				MinSizeBytes:         1,
				ExcludedContentTypes: []string{"application/json"},
			},
			contentType: "application/json",
			wantGzip:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Compression(tc.config))
			engine.GET("/data", func(c *gin.Context) {
				c.Data(http.StatusOK, tc.contentType, []byte(strings.Repeat("compressible-", 32)))
			})

			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			gotGzip := w.Header().Get("Content-Encoding") == "gzip"
			if gotGzip != tc.wantGzip {
				t.Fatalf("gzip applied = %t, want %t", gotGzip, tc.wantGzip)
			}
		})
	}
}

func TestCompressionRespectsGzipQualityZero(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/data", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 32)))
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "br, gzip;q=0")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressionExplicitGzipZeroOverridesWildcard(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/data", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 32)))
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Accept-Encoding", "*;q=1, gzip;q=0")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressionSkipsAttachmentResponses(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/download", func(c *gin.Context) {
		c.Header("Content-Disposition", `attachment; filename="users.json"`)
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 32)))
	})

	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func TestCompressionFlushBypassesCompression(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		if _, err := c.Writer.WriteString("hello"); err != nil {
			t.Fatalf("write hello: %v", err)
		}
		c.Writer.Flush()
		if _, err := c.Writer.WriteString("world"); err != nil {
			t.Fatalf("write world: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if got := w.Body.String(); got != "helloworld" {
		t.Fatalf("body = %q, want %q", got, "helloworld")
	}
}

func TestCompressionPreservesFirstStatusCode(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name    string
		handler func(*gin.Context) error
	}{
		{
			name: "write body before writeheader keeps 200",
			handler: func(c *gin.Context) error {
				c.Header("Content-Type", "application/json")
				if _, err := c.Writer.Write([]byte(strings.Repeat(`{"x":1}`, 32))); err != nil {
					return err
				}
				c.Writer.WriteHeader(http.StatusInternalServerError)
				return nil
			},
		},
		{
			name: "first writeheader wins",
			handler: func(c *gin.Context) error {
				c.Header("Content-Type", "application/json")
				c.Writer.WriteHeader(http.StatusCreated)
				c.Writer.WriteHeader(http.StatusInternalServerError)
				if _, err := c.Writer.Write([]byte(strings.Repeat(`{"x":1}`, 32))); err != nil {
					return err
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Compression(CompressionConfig{
				MinSizeBytes: 1,
			}))
			engine.GET("/data", func(c *gin.Context) {
				if err := tc.handler(c); err != nil {
					t.Fatalf("handler error: %v", err)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if tc.name == "write body before writeheader keeps 200" && w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
			if tc.name == "first writeheader wins" && w.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
			}
		})
	}
}

func TestCompressionSetsVaryHeaderWhenSkipping(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		acceptEncoding string
		minSizeBytes   int
	}{
		{
			name:           "client does not accept gzip",
			acceptEncoding: "",
			minSizeBytes:   1,
		},
		{
			name:           "small body skipped",
			acceptEncoding: "gzip",
			minSizeBytes:   1024,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := gin.New()
			engine.Use(Compression(CompressionConfig{
				MinSizeBytes: tc.minSizeBytes,
			}))
			engine.GET("/data", func(c *gin.Context) {
				c.Data(http.StatusOK, "application/json", []byte(`{"ok":true}`))
			})

			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			if tc.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tc.acceptEncoding)
			}

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if got := w.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Fatalf("Vary = %q, want %q", got, "Accept-Encoding")
			}
		})
	}
}

func TestCompressionDoesNotTreatInlineDispositionAsAttachment(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/inline", func(c *gin.Context) {
		disposition := mime.FormatMediaType("inline", map[string]string{
			"filename": "attachment-guide.json",
		})
		c.Header("Content-Disposition", disposition)
		c.Data(http.StatusOK, "application/json", []byte(strings.Repeat(`{"x":1}`, 32)))
	})

	req := httptest.NewRequest(http.MethodGet, "/inline", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want %q", got, "gzip")
	}
}

func TestCompressionSkipsUnknownContentType(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(Compression(CompressionConfig{
		MinSizeBytes: 1,
	}))
	engine.GET("/raw", func(c *gin.Context) {
		if _, err := c.Writer.Write([]byte{0x00, 0x01, 0x02, 0x03, 0x04}); err != nil {
			t.Fatalf("write raw body: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/raw", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

func mustReadGzipBody(t *testing.T, raw []byte) string {
	t.Helper()

	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			t.Fatalf("gzip reader close: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}

	return string(body)
}
