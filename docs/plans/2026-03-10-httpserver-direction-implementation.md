# httpserver Capability Enhancement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Stabilize `httpserver` core lifecycle and health-check behavior, then add an optional generic handler contract that improves project integration and AI coding consistency without turning `httpserver` into a framework.

**Architecture:** Keep `httpserver` as a Gin-based transport core. Preserve current direct Gin APIs, add minimal `Option`/hook/module primitives, then layer optional generic handler adapters for `decode -> validate -> invoke -> map error -> encode`. Keep logging, DI, and business conventions outside the core package.

**Tech Stack:** Go 1.24, Gin 1.11, standard library `net/http`, table-driven tests, `go test`

---

**References:**
- Read [docs/plans/2026-03-10-httpserver-direction-design.md](docs/plans/2026-03-10-httpserver-direction-design.md) before touching code.
- Check current implementation in `httpserver/server.go` and `httpserver/server_test.go`.
- Keep package examples aligned in `docs/httpserver.md`, `.ai/capabilities.yaml`, and `AGENTS.md`.
- Use `@superpowers:test-driven-development` for Tasks 1-5.
- Use `@superpowers:verification-before-completion` before claiming the package is done.

### Task 1: Make lifecycle behavior safe and observable

**Files:**
- Modify: `httpserver/server.go`
- Create: `httpserver/options.go`
- Create: `httpserver/lifecycle_test.go`

**Step 1: Write the failing tests**

Add table-driven tests that lock down the new lifecycle behavior:

```go
func TestServerStartReturnsListenError(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	srv := NewServer(&Config{Host: "127.0.0.1", Port: port})

	err = srv.Start()
	if err == nil {
		t.Fatal("expected listen error")
	}
}

func TestServerServeReportsAsyncErrorViaHook(t *testing.T) {
	t.Parallel()

	var gotErr error
	srv := NewServer(nil, WithHooks(Hooks{
		OnServeError: func(_ context.Context, event LifecycleEvent) {
			gotErr = event.Err
		},
	}))

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	_ = ln.Close()

	err := srv.Serve(ln)
	if err == nil {
		t.Fatal("expected serve error")
	}
	if gotErr == nil {
		t.Fatal("expected hook to observe serve error")
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestServer(StartReturnsListenError|ServeReportsAsyncErrorViaHook)' -v`

Expected: FAIL because `WithHooks`, `Hooks`, `LifecycleEvent`, and `Serve` do not exist yet, and `Start()` still panics on asynchronous serve failure.

**Step 3: Write the minimal implementation**

Implement the smallest possible lifecycle surface:

```go
type LifecycleEvent struct {
	Addr       string
	HealthAddr string
	Err        error
}

type Hooks struct {
	OnStarting         func(context.Context, LifecycleEvent)
	OnStarted          func(context.Context, LifecycleEvent)
	OnServeError       func(context.Context, LifecycleEvent)
	OnShuttingDown     func(context.Context, LifecycleEvent)
	OnShutdownComplete func(context.Context, LifecycleEvent)
}

type Option func(*Server)

func WithHooks(h Hooks) Option {
	return func(s *Server) { s.hooks = h }
}
```

Update `Server` and lifecycle methods:

```go
type Server struct {
	config     *Config
	engine     *gin.Engine
	server     *http.Server
	serveErrCh chan error
	hooks      Hooks
}

func (s *Server) Serve(ln net.Listener) error { ... }
func (s *Server) Errors() <-chan error { return s.serveErrCh }
func (s *Server) Start() error { ... } // use net.Listen synchronously, never panic
```

Key implementation rules:
- Keep `NewServer(config *Config, opts ...Option) *Server` backward compatible for existing call sites.
- `Start()` must return bind/listen errors directly.
- `Serve(listener)` must surface runtime serve errors through hooks and `Errors()`.
- Remove all direct `panic` and `fmt.Println` from lifecycle paths.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestServer(StartReturnsListenError|ServeReportsAsyncErrorViaHook)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/options.go httpserver/lifecycle_test.go
git commit -m "refactor(httpserver): make lifecycle errors observable"
```

### Task 2: Close the `HealthCheckPort` contract

**Files:**
- Modify: `httpserver/server.go`
- Modify: `httpserver/options.go`
- Create: `httpserver/health_server_test.go`

**Step 1: Write the failing tests**

Add table-driven tests for the two supported modes:

```go
func TestHealthCheckServingModes(t *testing.T) {
	tests := []struct {
		name            string
		appPort         int
		healthPort      int
		expectDedicated bool
	}{
		{name: "shared port", appPort: freePort(t), healthPort: 0, expectDedicated: false},
		{name: "dedicated port", appPort: freePort(t), healthPort: freePort(t), expectDedicated: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(&Config{
				Host:              "127.0.0.1",
				Port:              tt.appPort,
				EnableHealthCheck: true,
				HealthCheckPath:   "/healthz",
				HealthCheckPort:   tt.healthPort,
			})
			srv.GET("/users/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			if err := srv.Start(); err != nil {
				t.Fatalf("start: %v", err)
			}
			defer srv.Shutdown(context.Background())

			assertStatus(t, "http://127.0.0.1:%d/healthz", expectedHealthPort(tt), http.StatusOK)
			assertStatus(t, "http://127.0.0.1:%d/users/ping", tt.appPort, http.StatusNoContent)
		})
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run TestHealthCheckServingModes -v`

Expected: FAIL because the current implementation ignores `HealthCheckPort` and only registers the health route on the main Gin engine.

**Step 3: Write the minimal implementation**

Add dedicated health-server support without changing the shared-port behavior:

```go
type Server struct {
	// existing fields...
	healthServer  *http.Server
	healthHandler gin.HandlerFunc
}

func (s *Server) EnableHealthCheckWithManager(manager *HealthCheckManager) {
	s.healthHandler = HealthHandlerWithManager(manager)
	if s.config.HealthCheckPort == 0 {
		s.engine.GET(s.config.HealthCheckPath, s.healthHandler)
	}
}
```

Implementation rules:
- Default `healthHandler` to `DefaultHealthHandler()`.
- When `HealthCheckPort == 0`, keep serving health checks from the main engine.
- When `HealthCheckPort != 0`, create a separate `gin.Engine` that serves only the health path.
- `Shutdown(ctx)` must stop both the main server and the dedicated health server.
- Reuse the same manager-backed handler regardless of port mode.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run TestHealthCheckServingModes -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/options.go httpserver/health_server_test.go
git commit -m "feat(httpserver): support dedicated health check port"
```

### Task 3: Add route modules and minimal option plumbing

**Files:**
- Modify: `httpserver/server.go`
- Modify: `httpserver/options.go`
- Create: `httpserver/module.go`
- Create: `httpserver/module_test.go`

**Step 1: Write the failing tests**

Add tests for explicit and constructor-based module registration:

```go
type testModule struct{}

func (testModule) RegisterRoutes(r gin.IRoutes) {
	r.GET("/module/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func TestRegisterModules(t *testing.T) {
	srv := NewServer(nil)
	srv.RegisterModules(testModule{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/module/ping", nil)
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNewServerWithModulesOption(t *testing.T) {
	srv := NewServer(nil, WithModules(testModule{}))
	// hit /module/ping and assert 200
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(RegisterModules|NewServerWithModulesOption)' -v`

Expected: FAIL because `RouteModule`, `RegisterModules`, and `WithModules` do not exist yet.

**Step 3: Write the minimal implementation**

Introduce the route-module primitives:

```go
type RouteModule interface {
	RegisterRoutes(r gin.IRoutes)
}

func (s *Server) RegisterModules(modules ...RouteModule) {
	for _, module := range modules {
		module.RegisterRoutes(s.engine)
	}
}

func WithModules(modules ...RouteModule) Option {
	return func(s *Server) {
		s.RegisterModules(modules...)
	}
}
```

Implementation rules:
- Keep `RegisterRoutes(func(*gin.Engine))` unchanged for existing callers.
- `RouteModule` is a convenience contract, not a required pattern.
- `WithModules` must run after the server has been constructed and health routes have been configured.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(RegisterModules|NewServerWithModulesOption)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/server.go httpserver/options.go httpserver/module.go httpserver/module_test.go
git commit -m "feat(httpserver): add route module registration"
```

### Task 4: Add the generic JSON handler contract

**Files:**
- Create: `httpserver/handler.go`
- Create: `httpserver/handler_test.go`

**Step 1: Write the failing tests**

Start with the smallest useful contract: JSON decode, default validation, default error mapping, and JSON response encoding.

```go
type loginRequest struct {
	Email string `json:"email"`
}

func (r loginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	return nil
}

func TestHandleJSONSuccess(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	// POST valid JSON, expect 200 and echoed email
}

func TestHandleJSONValidateError(t *testing.T) {
	// POST {"email":""}, expect 422
}

func TestHandleJSONDecodeError(t *testing.T) {
	// POST malformed JSON, expect 400
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSON(Success|ValidateError|DecodeError)' -v`

Expected: FAIL because `HandleJSON` and the generic handler contracts do not exist yet.

**Step 3: Write the minimal implementation**

Implement the generic contract in `httpserver/handler.go`:

```go
type HandlerFunc[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)
type Decoder[Req any] func(*gin.Context) (Req, error)
type Validator[Req any] func(context.Context, Req) error
type Encoder[Resp any] func(*gin.Context, int, Resp)

func HandleJSON[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption[Req, Resp]) gin.HandlerFunc {
	// decode JSON -> validate -> invoke -> map error -> encode JSON
}
```

Support these minimum extension points:
- default JSON body decoder
- automatic `Validate() error` / `Validate(context.Context) error`
- default error mapper (`400` / `422` / `500`)
- `WithSuccessStatus(code int)`
- `WithErrorMapper(...)`
- `WithEncoder(...)`

Keep the default response body raw JSON. Do not introduce a mandatory envelope format.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'TestHandleJSON(Success|ValidateError|DecodeError)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go
git commit -m "feat(httpserver): add generic json handler contract"
```

### Task 5: Extend the handler contract with query/URI decoding and coexistence tests

**Files:**
- Modify: `httpserver/handler.go`
- Modify: `httpserver/handler_test.go`

**Step 1: Write the failing tests**

Extend coverage to the remaining recommended decoder modes and mixed routing:

```go
func TestHandleQuerySuccess(t *testing.T) {
	type listRequest struct {
		Page int `form:"page"`
	}

	srv := NewServer(nil)
	srv.GET("/users", Handle(func(ctx context.Context, req listRequest) (gin.H, error) {
		return gin.H{"page": req.Page}, nil
	}, WithDecoder(DecodeQuery[listRequest]())))

	// GET /users?page=2, expect 200 and {"page":2}
}

func TestHandleURISuccess(t *testing.T) {
	type getUserRequest struct {
		ID string `uri:"id"`
	}

	srv := NewServer(nil)
	srv.GET("/users/:id", Handle(func(ctx context.Context, req getUserRequest) (gin.H, error) {
		return gin.H{"id": req.ID}, nil
	}, WithDecoder(DecodeURI[getUserRequest]())))

	// GET /users/123, expect 200 and {"id":"123"}
}

func TestTypedAndNativeHandlersCoexist(t *testing.T) {
	// register one typed route and one native gin route, assert both work
}
```

**Step 2: Run the tests to verify they fail**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(HandleQuerySuccess|HandleURISuccess|TypedAndNativeHandlersCoexist)' -v`

Expected: FAIL because `Handle`, `DecodeQuery`, `DecodeURI`, and generic decoder override options do not exist yet.

**Step 3: Write the minimal implementation**

Expand the handler package:

```go
func Handle[Req any, Resp any](fn HandlerFunc[Req, Resp], opts ...HandlerOption[Req, Resp]) gin.HandlerFunc { ... }
func DecodeQuery[Req any]() Decoder[Req] { ... }
func DecodeURI[Req any]() Decoder[Req] { ... }
func ComposeDecoder[Req any](decoders ...Decoder[Req]) Decoder[Req] { ... }
func WithDecoder[Req any, Resp any](decoder Decoder[Req]) HandlerOption[Req, Resp] { ... }
```

Implementation rules:
- Keep `HandleJSON` as a convenience wrapper over `Handle`.
- Allow typed and native Gin routes to coexist on the same server with no hidden middleware or routing assumptions.
- `ComposeDecoder` should run decoders in order and stop on the first error.

**Step 4: Run the tests to verify they pass**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -run 'Test(HandleQuerySuccess|HandleURISuccess|TypedAndNativeHandlersCoexist)' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add httpserver/handler.go httpserver/handler_test.go
git commit -m "feat(httpserver): support query and uri typed handlers"
```

### Task 6: Refresh docs and AI-facing guidance

**Files:**
- Modify: `docs/httpserver.md`
- Modify: `.ai/capabilities.yaml`
- Modify: `AGENTS.md`

**Step 1: Capture the stale examples before editing**

Run:

```bash
rg -n 'httpserver\.New\(|HandleJSON|RouteModule|HealthCheckPort' docs/httpserver.md .ai/capabilities.yaml AGENTS.md
```

Expected:
- Existing examples still reference `httpserver.New(...)`, which does not match the current package API.
- No examples yet demonstrate `HandleJSON`, `RouteModule`, or dedicated health-port behavior.

**Step 2: Update the documentation**

Make the docs match the implemented API:

- Replace stale constructor examples with `NewServer(...)`
- Add one “Gin 直写模式” example
- Add one “typed handler 推荐模式” example for `user.login`
- Document dedicated `HealthCheckPort`
- Document `RouteModule` and constructor-based dependency injection
- Update `.ai/capabilities.yaml` snippets to reflect the recommended API and remove any incorrect dependency assumptions
- Fix the quick-reference line in `AGENTS.md`

Use snippets like:

```go
type UserModule struct {
	auth *AuthService
}

func (m *UserModule) RegisterRoutes(r gin.IRoutes) {
	r.POST("/login", httpserver.HandleJSON(m.Login))
}

func (m *UserModule) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return m.auth.Login(ctx, req)
}
```

**Step 3: Re-run the grep to confirm the docs are aligned**

Run:

```bash
rg -n 'httpserver\.New\(|HandleJSON|RouteModule|HealthCheckPort' docs/httpserver.md .ai/capabilities.yaml AGENTS.md
```

Expected:
- No stale `httpserver.New(...)` examples remain unless a compatibility alias was intentionally added.
- Typed handler and module examples now exist in docs and AI snippets.

**Step 4: Run package verification**

Run: `GOCACHE=/tmp/go-build go test ./httpserver -v`

Expected: PASS

**Step 5: Commit**

```bash
git add docs/httpserver.md .ai/capabilities.yaml AGENTS.md
git commit -m "docs(httpserver): align docs with typed handler workflow"
```

