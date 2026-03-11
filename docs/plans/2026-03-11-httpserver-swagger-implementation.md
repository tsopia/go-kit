# httpserver Swagger Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `httpserver/swagger` integration that keeps Swagger UI public by default, works cleanly with protected route groups, and gives AI coding a stable `swaggo`-based template without forcing a new response DSL.

**Architecture:** Keep Swagger support out of `httpserver.Server` core and implement it as a small `httpserver/swagger` subpackage. Use `gin-swagger` and `swaggerFiles` for runtime UI mounting, keep document generation in `swaggo`, and update repository docs/capability metadata so both humans and AI follow the same route-group and annotation conventions.

**Tech Stack:** Go 1.24, Gin 1.11, `github.com/swaggo/gin-swagger`, `github.com/swaggo/files`, table-driven tests, `go test`, `go run ./cmd/gokit list`

---

**References:**
- Read [docs/plans/2026-03-11-httpserver-swagger-design.md](/Users/kj/projects/go-kit/docs/plans/2026-03-11-httpserver-swagger-design.md) before touching code.
- Follow repository workflow in [AGENTS.md](/Users/kj/projects/go-kit/AGENTS.md): update capability metadata before building the new package.
- Keep package docs aligned across [httpserver/doc.go](/Users/kj/projects/go-kit/httpserver/doc.go), [httpserver/README.md](/Users/kj/projects/go-kit/httpserver/README.md), and the new `httpserver/swagger` package docs.
- Use `@superpowers:test-driven-development` for Tasks 2 and 3.
- Use `@superpowers:verification-before-completion` before claiming the feature is done.

### Task 1: Register Swagger as a repository capability first

**Files:**
- Modify: `.ai/capabilities.yaml`
- Modify: `cmd/gokit/pkg/gokit/capabilities.yaml`
- Modify: `cmd/gokit/pkg/gokit/capability_test.go`
- Modify: `AGENTS.md`

**Step 1: Update capability metadata before code**

This repository requires capability metadata to be updated before adding a new package. Add a new capability entry for the new subpackage and keep the embedded fallback in sync:

```yaml
- name: swagger
  description: Swagger UI 路由挂载与文档接入（配合 httpserver 使用）
  import: github.com/tsopia/go-kit/httpserver/swagger
  scenarios:
    - name: 挂载 Swagger UI
      snippet: |
        httpswagger.Register(public, httpswagger.Config{
            Path: "/swagger/*any",
        })
  dependencies: [httpserver]
```

Also update the quick-reference table in `AGENTS.md` with a new row for `httpserver/swagger`.

**Step 2: Add a capability loading test**

Add a focused test to [cmd/gokit/pkg/gokit/capability_test.go](/Users/kj/projects/go-kit/cmd/gokit/pkg/gokit/capability_test.go):

```go
func TestGetCapabilitySwagger(t *testing.T) {
	t.Parallel()

	capability, err := GetCapability("swagger")
	if err != nil {
		t.Fatalf("GetCapability(swagger): %v", err)
	}
	if capability.Import != "github.com/tsopia/go-kit/httpserver/swagger" {
		t.Fatalf("unexpected import: %s", capability.Import)
	}
	if len(capability.Scenarios) == 0 {
		t.Fatal("expected swagger scenarios")
	}
}
```

**Step 3: Run metadata verification**

Run: `GOCACHE=/tmp/go-build go test ./cmd/gokit/pkg/gokit -run TestGetCapabilitySwagger -v`

Expected: PASS

Run: `GOCACHE=/tmp/go-build go run ./cmd/gokit list`

Expected: output contains a `swagger` row with import path `github.com/tsopia/go-kit/httpserver/swagger`

**Step 4: Commit**

```bash
git add .ai/capabilities.yaml cmd/gokit/pkg/gokit/capabilities.yaml cmd/gokit/pkg/gokit/capability_test.go AGENTS.md
git commit -m "docs(ai): add swagger capability metadata"
```

### Task 2: Create the `httpserver/swagger` package surface

**Files:**
- Create: `httpserver/swagger/config.go`
- Create: `httpserver/swagger/register.go`
- Create: `httpserver/swagger/register_test.go`
- Create: `httpserver/swagger/doc.go`
- Create: `httpserver/swagger/README.md`
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Write the failing tests**

Add table-driven tests that define the package contract:

```go
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Fatal("expected swagger to be enabled by default")
	}
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
		name       string
		cfg        Config
		wantMethod string
		wantPath   string
		wantRoute  bool
	}{
		{
			name:       "zero value uses defaults",
			cfg:        Config{},
			wantMethod: http.MethodGet,
			wantPath:   "/swagger/*any",
			wantRoute:  true,
		},
		{
			name:       "disabled skips registration",
			cfg:        Config{Enabled: false},
			wantMethod: http.MethodGet,
			wantPath:   "/swagger/*any",
			wantRoute:  false,
		},
		{
			name:       "custom path",
			cfg:        Config{Enabled: true, Path: "/docs/*any"},
			wantMethod: http.MethodGet,
			wantPath:   "/docs/*any",
			wantRoute:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			Register(engine.Group(""), tt.cfg)

			found := false
			for _, route := range engine.Routes() {
				if route.Method == tt.wantMethod && route.Path == tt.wantPath {
					found = true
				}
			}

			if found != tt.wantRoute {
				t.Fatalf("route found=%v want=%v", found, tt.wantRoute)
			}
		})
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/swagger -run 'Test(DefaultConfig|RegisterRouteModes)' -v`

Expected: FAIL because the `httpserver/swagger` package and its config/register functions do not exist yet.

**Step 3: Write the minimal implementation**

Add the package surface:

```go
type Config struct {
	Enabled bool
	Path    string
	DocURL  string
}

func DefaultConfig() Config {
	return Config{
		Enabled: true,
		Path:    "/swagger/*any",
		DocURL:  "doc.json",
	}
}

func Register(r gin.IRoutes, cfg Config) {
	cfg = defaultConfig(cfg)
	if !cfg.Enabled {
		return
	}

	r.GET(
		cfg.Path,
		ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.URL(cfg.DocURL),
		),
	)
}
```

Implementation rules:
- Keep the package runtime-only; it should not run `swag init`.
- Add `gin-swagger` and `swaggerFiles` to `go.mod`, then run `go mod tidy`.
- `Config{}` should mean “enabled with defaults”, not “disabled”.
- Document in `doc.go` and `README.md` that callers must import generated docs separately.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/swagger -run 'Test(DefaultConfig|RegisterRouteModes)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add go.mod go.sum httpserver/swagger/config.go httpserver/swagger/register.go httpserver/swagger/register_test.go httpserver/swagger/doc.go httpserver/swagger/README.md
git commit -m "feat(httpserver): add swagger registration package"
```

### Task 3: Lock down public Swagger and protected API routing

**Files:**
- Modify: `httpserver/swagger/register_test.go`
- Modify: `httpserver/swagger/register.go`

**Step 1: Write the failing tests**

Extend the test suite with a public-vs-protected routing regression test and a custom DocURL test:

```go
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
		Enabled: true,
		Path:    "/docs/*any",
		DocURL:  "/openapi/doc.json",
	})

	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/docs/index.html", nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "/openapi/doc.json") {
		t.Fatal("expected custom doc url in swagger html")
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/swagger -run 'Test(SwaggerRouteStaysPublic|RegisterUsesCustomDocURL)' -v`

Expected: FAIL until `Register` correctly preserves group placement and forwards the configured `DocURL`.

**Step 3: Write the minimal implementation**

Keep the implementation small:

```go
func Register(r gin.IRoutes, cfg Config) {
	cfg = defaultConfig(cfg)
	if !cfg.Enabled {
		return
	}

	handler := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL(cfg.DocURL),
	)

	r.GET(cfg.Path, handler)
}
```

Implementation rules:
- Do not add auth bypass logic inside the package.
- Preserve the caller’s `gin.IRoutes` exactly; route visibility must come from which group the caller passes in.
- Keep custom `DocURL` support explicit so projects can mount Swagger UI away from `doc.json` if needed.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/swagger -run 'Test(SwaggerRouteStaysPublic|RegisterUsesCustomDocURL)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/swagger/register.go httpserver/swagger/register_test.go
git commit -m "test(httpserver): lock swagger public route behavior"
```

### Task 4: Update package docs and AI-facing templates

**Files:**
- Modify: `httpserver/doc.go`
- Modify: `httpserver/README.md`
- Modify: `httpserver/swagger/doc.go`
- Modify: `httpserver/swagger/README.md`
- Optionally create: `httpserver/swagger/.ai-snippet.md`

**Step 1: Update `httpserver` package docs**

Add a short Swagger section to [httpserver/doc.go](/Users/kj/projects/go-kit/httpserver/doc.go) and [httpserver/README.md](/Users/kj/projects/go-kit/httpserver/README.md) covering:

- `httpserver/swagger` subpackage location
- `public` vs `protected` route-group pattern
- recommended `swag init -g cmd/server/main.go -o internal/docs`
- `swaggo` comment placement on typed transport handlers

**Step 2: Finish `httpserver/swagger` package docs**

Document the exact integration shape in [httpserver/swagger/README.md](/Users/kj/projects/go-kit/httpserver/swagger/README.md):

```go
import (
	_ "your/module/internal/docs"

	"github.com/tsopia/go-kit/httpserver"
	httpswagger "github.com/tsopia/go-kit/httpserver/swagger"
)

srv := httpserver.NewServer(nil)
public := srv.Group("")
protected := srv.Group("/api/v1")
protected.Use(AuthMiddleware())

httpswagger.Register(public, httpswagger.Config{
	Path: "/swagger/*any",
})
```

Required documentation points:
- Swagger 默认公开
- 推荐不要把鉴权挂到全局 `Engine`
- 历史项目如已全局鉴权，需在中间件中白名单 `/swagger/`
- AI 注释模板要展示 JSON、Query、鉴权接口三种场景

**Step 3: Verify docs and templates**

Run: `rg -n "swagger.Register|swag init|public := srv.Group|@Security BearerAuth" httpserver/doc.go httpserver/README.md httpserver/swagger/doc.go httpserver/swagger/README.md AGENTS.md`

Expected: all required snippets and conventions appear in the updated docs.

**Step 4: Commit**

```bash
git add httpserver/doc.go httpserver/README.md httpserver/swagger/doc.go httpserver/swagger/README.md AGENTS.md
git commit -m "docs(httpserver): add swagger usage guidelines"
```

### Task 5: Run full verification and close the loop

**Files:**
- Verify: `httpserver/swagger/*`
- Verify: `cmd/gokit/pkg/gokit/*`
- Verify: `.ai/capabilities.yaml`

**Step 1: Run focused package tests**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/swagger ./cmd/gokit/pkg/gokit -v`

Expected: PASS

**Step 2: Run broader repository verification**

Run: `GOCACHE=/tmp/go-build go test ./httpserver/... -v`

Expected: PASS

Run: `GOCACHE=/tmp/go-build go test ./...`

Expected: PASS

**Step 3: Verify the capability list and docs**

Run: `GOCACHE=/tmp/go-build go run ./cmd/gokit list`

Expected: output includes the new `swagger` capability.

Run: `git diff --stat HEAD~4..HEAD`

Expected: only Swagger-related package, tests, and documentation changes are present.

**Step 4: Final commit if needed**

If Task 5 required any final touch-ups:

```bash
git add -A
git commit -m "chore(httpserver): finalize swagger integration docs and verification"
```
