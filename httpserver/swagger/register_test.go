package swagger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if cfg.Path != "/swagger/*any" {
		t.Fatalf("unexpected path: %s", cfg.Path)
	}

	if cfg.DocURL != "doc.json" {
		t.Fatalf("unexpected doc url: %s", cfg.DocURL)
	}
}

func TestRegisterRouteModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		wantPath string
	}{
		{
			name:     "zero value uses defaults",
			cfg:      Config{},
			wantPath: "/swagger/*any",
		},
		{
			name:     "custom path",
			cfg:      Config{Path: "/docs/*any"},
			wantPath: "/docs/*any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			Register(engine.Group(""), tt.cfg)

			found := false
			for _, route := range engine.Routes() {
				if route.Method == http.MethodGet && route.Path == tt.wantPath {
					found = true
				}
			}

			if !found {
				t.Fatalf("expected route %s to be registered", tt.wantPath)
			}
		})
	}
}

func TestSwaggerRouteStaysPublic(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	public := engine.Group("")
	protected := engine.Group("/api")
	protected.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})

	Register(public, Config{})
	protected.GET("/users", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	swaggerResp := httptest.NewRecorder()
	engine.ServeHTTP(swaggerResp, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if swaggerResp.Code == http.StatusUnauthorized {
		t.Fatal("swagger route should not inherit protected middleware")
	}

	apiResp := httptest.NewRecorder()
	engine.ServeHTTP(apiResp, httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if apiResp.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected api status: %d", apiResp.Code)
	}
}

func TestRegisterUsesCustomDocURL(t *testing.T) {
	t.Parallel()

	engine := gin.New()
	Register(engine.Group(""), Config{
		Path:   "/docs/*any",
		DocURL: "/openapi/doc.json",
	})

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/docs/swagger-initializer.js", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}

	if !strings.Contains(resp.Body.String(), "/openapi/doc.json") {
		t.Fatal("expected custom doc url in swagger initializer")
	}
}
