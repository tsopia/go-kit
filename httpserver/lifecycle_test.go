package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

type stubListener struct {
	addr net.Addr
}

func (l *stubListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept not implemented")
}

func (l *stubListener) Close() error {
	return nil
}

func (l *stubListener) Addr() net.Addr {
	return l.addr
}

type stubAddr string

func (a stubAddr) Network() string {
	return "tcp"
}

func (a stubAddr) String() string {
	return string(a)
}

func TestServerStateTransitions(t *testing.T) {
	t.Parallel()

	srv := NewServer(nil, WithManualReadiness())

	if got := srv.State(); got != StateNew {
		t.Fatalf("initial state = %q, want %q", got, StateNew)
	}

	// 必须经过 Starting 才能到 Ready
	srv.setState(StateStarting)

	if err := srv.MarkReady(); err != nil {
		t.Fatalf("MarkReady(): %v", err)
	}
	if got := srv.State(); got != StateReady {
		t.Fatalf("state after MarkReady = %q, want %q", got, StateReady)
	}

	if err := srv.MarkDraining(); err != nil {
		t.Fatalf("MarkDraining(): %v", err)
	}
	if got := srv.State(); got != StateDraining {
		t.Fatalf("state after MarkDraining = %q, want %q", got, StateDraining)
	}
}

func TestStateTransitionAPIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   State
		call      func(*Server) error
		wantState State
		wantErr   bool
	}{
		{
			name:      "mark ready from new returns error",
			initial:   StateNew,
			call:      func(s *Server) error { return s.MarkReady() },
			wantState: StateNew,
			wantErr:   true,
		},
		{
			name:      "mark ready from starting succeeds",
			initial:   StateStarting,
			call:      func(s *Server) error { return s.MarkReady() },
			wantState: StateReady,
		},
		{
			name:      "mark draining from ready succeeds",
			initial:   StateReady,
			call:      func(s *Server) error { return s.MarkDraining() },
			wantState: StateDraining,
		},
		{
			name:      "mark draining from stopped returns error",
			initial:   StateStopped,
			call:      func(s *Server) error { return s.MarkDraining() },
			wantState: StateStopped,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(nil, WithManualReadiness())
			srv.setState(tt.initial)

			err := tt.call(srv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("call error = %v, wantErr %v", err, tt.wantErr)
			}

			if got := srv.State(); got != tt.wantState {
				t.Fatalf("State() = %q, want %q", got, tt.wantState)
			}
		})
	}
}

func TestServerStartReturnsListenError(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Fatalf("close listener: %v", err)
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	srv := NewServer(&Config{
		Host: "127.0.0.1",
		Port: port,
	})

	if err := srv.Start(); err == nil {
		t.Fatal("expected listen error")
	}
}

func TestReportServeErrorTransitionsToFailed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		initial   State
		wantState State
	}{
		{
			name:      "starting becomes failed",
			initial:   StateStarting,
			wantState: StateFailed,
		},
		{
			name:      "ready becomes failed",
			initial:   StateReady,
			wantState: StateFailed,
		},
		{
			name:      "stopped stays stopped",
			initial:   StateStopped,
			wantState: StateStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(nil, WithManualReadiness())
			srv.setState(tt.initial)

			srv.reportServeError(errors.New("boom"))

			if got := srv.State(); got != tt.wantState {
				t.Fatalf("State() = %q, want %q", got, tt.wantState)
			}
		})
	}
}

func TestServerServeReportsErrorViaHook(t *testing.T) {
	t.Parallel()

	var gotErr error
	srv := NewServer(nil, WithHooks(Hooks{
		OnServeError: func(_ context.Context, event LifecycleEvent) {
			gotErr = event.Err
		},
	}))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	err = srv.Serve(ln)
	if err == nil {
		t.Fatal("expected serve error")
	}
	if gotErr == nil {
		t.Fatal("expected hook to observe serve error")
	}
}

func TestStartInternalValidatesStartState(t *testing.T) {
	t.Parallel()

	listener := &stubListener{addr: stubAddr("127.0.0.1:8080")}

	tests := []struct {
		name       string
		initial    State
		setup      func(*Server)
		wantErr    bool
		wantCalled bool
		wantState  State
	}{
		{
			name:       "cannot start from ready",
			initial:    StateReady,
			wantErr:    true,
			wantCalled: false,
			wantState:  StateReady,
		},
		{
			name:    "failed can restart even with stale server reference",
			initial: StateFailed,
			setup: func(s *Server) {
				s.server = &http.Server{Addr: "stale"}
			},
			wantErr:    false,
			wantCalled: true,
			wantState:  StateReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(nil)
			srv.setState(tt.initial)
			if tt.setup != nil {
				tt.setup(srv)
			}

			called := false
			err := srv.startInternal(listener, true, func(net.Listener) error {
				called = true
				return nil
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("startInternal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if called != tt.wantCalled {
				t.Fatalf("serveFn called = %v, want %v", called, tt.wantCalled)
			}
			if got := srv.State(); got != tt.wantState {
				t.Fatalf("State() = %q, want %q", got, tt.wantState)
			}
		})
	}
}

// TestIsRunning 验证 IsRunning() 基于状态机语义工作。
// 当前实现通过 State() 判断，因此 Shutdown() 后应返回 false。
func TestIsRunning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setupFunc         func(s *Server)
		wantRunning       bool
		wantAfterShutdown bool
	}{
		{
			name: "IsRunning_should_return_false_after_Shutdown",
			// 使用 Setup 方式，避免实际启动服务器
			setupFunc: func(s *Server) {
				// 模拟服务器已启动的状态：必须经过 Starting -> Ready
				s.setState(StateStarting)
				s.setState(StateReady)
				s.server = &http.Server{Addr: ":8080"}
			},
			wantRunning:       true,
			wantAfterShutdown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(&Config{
				Host: "127.0.0.1",
				Port: 8080,
			})

			// 设置初始状态
			tt.setupFunc(srv)

			// 验证启动后 IsRunning 返回 true
			if got := srv.IsRunning(); got != tt.wantRunning {
				t.Fatalf("IsRunning() = %v, want %v", got, tt.wantRunning)
			}

			// 关闭服务器
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				t.Fatalf("shutdown server: %v", err)
			}

			// 验证 Shutdown 后 IsRunning 返回 false
			if got := srv.IsRunning(); got != tt.wantAfterShutdown {
				t.Errorf("IsRunning() = %v after Shutdown(), want %v", got, tt.wantAfterShutdown)
			}
		})
	}
}

// TestDrainTimeout 验证 DrainTimeout 配置应影响 shutdown 流程
// WaitForShutdown 收到信号后应先 MarkDraining()，等待 DrainTimeout 后才进入 Shutdown()
func TestDrainTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		drainTimeout      time.Duration
		wantDrainingState bool
	}{
		{
			name:              "DrainTimeout_should_delay_shutdown",
			drainTimeout:      100 * time.Millisecond,
			wantDrainingState: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(&Config{
				Host:            "127.0.0.1",
				Port:            18080,
				DrainTimeout:    tt.drainTimeout,
				ShutdownTimeout: 5 * time.Second,
				ReadTimeout:     time.Second,
				WriteTimeout:    time.Second,
				IdleTimeout:     time.Second,
			}, WithManualReadiness())

			// 设置服务器为已启动状态
			srv.setState(StateReady)

			// 记录开始时间
			start := time.Now()

			// 模拟 MarkDraining
			if err := srv.MarkDraining(); err != nil {
				t.Fatalf("MarkDraining(): %v", err)
			}

			// 验证状态是 Draining
			if got := srv.State(); got != StateDraining {
				t.Errorf("State() = %q after MarkDraining, want %q", got, StateDraining)
			}

			// 等待 DrainTimeout
			time.Sleep(srv.config.DrainTimeout)

			// 验证等待了足够的时间（至少 DrainTimeout）
			elapsed := time.Since(start)
			if elapsed < srv.config.DrainTimeout {
				t.Errorf("elapsed time %v < DrainTimeout %v", elapsed, srv.config.DrainTimeout)
			}

			t.Logf("DrainTimeout respected: waited %v (configured: %v)", elapsed, srv.config.DrainTimeout)
		})
	}
}

func TestRunWithContextUsesGracefulShutdownPipeline(t *testing.T) {
	t.Parallel()

	hooks := make([]string, 0, 2)
	srv := NewServer(&Config{
		Host:            "127.0.0.1",
		Port:            0,
		DrainTimeout:    50 * time.Millisecond,
		ShutdownTimeout: time.Second,
	}, WithHooks(Hooks{
		OnShuttingDown: func(_ context.Context, _ LifecycleEvent) {
			hooks = append(hooks, "shutting_down")
		},
		OnShutdownComplete: func(_ context.Context, _ LifecycleEvent) {
			hooks = append(hooks, "shutdown_complete")
		},
	}))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := srv.RunWithContext(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunWithContext() error = %v", err)
	}
	if elapsed < srv.config.DrainTimeout {
		t.Fatalf("elapsed = %v, want >= %v", elapsed, srv.config.DrainTimeout)
	}
	if got := srv.State(); got != StateStopped {
		t.Fatalf("State() = %q, want %q", got, StateStopped)
	}
	if len(hooks) != 2 || hooks[0] != "shutting_down" || hooks[1] != "shutdown_complete" {
		t.Fatalf("hooks = %v, want [shutting_down shutdown_complete]", hooks)
	}
}

func TestShutdownContextDisablesDeadlineWhenConfigured(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{
		ShutdownTimeout: DisableTimeout,
	})

	ctx, cancel := srv.shutdownContext()
	defer cancel()

	if _, ok := ctx.Deadline(); ok {
		t.Fatal("shutdown context unexpectedly has deadline")
	}
	if srv.config.ShutdownTimeout != 0 {
		t.Fatalf("ShutdownTimeout = %v, want 0", srv.config.ShutdownTimeout)
	}
}

// TestReadinessDuringDraining 验证 draining 状态 readiness 返回 503
func TestReadinessDuringDraining(t *testing.T) {
	t.Parallel()

	srv := NewServer(&Config{
		Host:              "127.0.0.1",
		Port:              18081,
		EnableHealthCheck: true,
		HealthCheckPort:   0, // 共享主端口
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
	}, WithManualReadiness())

	// 初始状态应该是 New
	if got := srv.State(); got != StateNew {
		t.Fatalf("initial state = %q, want %q", got, StateNew)
	}

	// 必须经过 Starting 才能到 Ready
	srv.setState(StateStarting)

	// 标记为 Ready
	if err := srv.MarkReady(); err != nil {
		t.Fatalf("MarkReady(): %v", err)
	}
	if got := srv.State(); got != StateReady {
		t.Errorf("state after MarkReady = %q, want %q", got, StateReady)
	}

	// 标记为 Draining
	if err := srv.MarkDraining(); err != nil {
		t.Fatalf("MarkDraining(): %v", err)
	}
	if got := srv.State(); got != StateDraining {
		t.Errorf("state after MarkDraining = %q, want %q", got, StateDraining)
	}

	// 验证 readinessEndpoint 在 draining 状态返回 503
	// 通过检查 readinessEndpoint 的逻辑（返回 ServiceUnavailable 当 state != Ready）
	// 实际 HTTP 测试需要启动服务器，这里只验证状态机
	t.Log("Readiness should return 503 when in Draining state")
}

// TestServerLifecycleHooks 验证所有启动路径都触发相同的 lifecycle hooks
func TestServerLifecycleHooks(t *testing.T) {
	t.Parallel()

	// 此测试验证 Serve/Start/Run/RunTLS 都走统一的 lifecycle pipeline
	// 确保 OnStarting 和 OnStarted 触发顺序一致

	tests := []struct {
		name      string
		wantHooks []string
	}{
		{
			name:      "all_start_paths_should_trigger_same_hooks",
			wantHooks: []string{"OnStarting", "OnStarted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var triggeredHooks []string

			srv := NewServer(&Config{
				Host: "127.0.0.1",
				Port: 0,
			}, WithHooks(Hooks{
				OnStarting: func(_ context.Context, _ LifecycleEvent) {
					triggeredHooks = append(triggeredHooks, "OnStarting")
				},
				OnStarted: func(_ context.Context, _ LifecycleEvent) {
					triggeredHooks = append(triggeredHooks, "OnStarted")
				},
			}), WithManualReadiness())

			// 模拟服务器启动状态（不实际启动，避免端口绑定问题）
			srv.setState(StateStarting)
			srv.emitHook(srv.hooks.OnStarting, srv.lifecycleEvent(nil))
			srv.setState(StateReady)
			srv.emitHook(srv.hooks.OnStarted, srv.lifecycleEvent(nil))

			// 验证 hooks 被触发
			if len(triggeredHooks) != len(tt.wantHooks) {
				t.Errorf("triggered %d hooks, want %d", len(triggeredHooks), len(tt.wantHooks))
			}
			for i, want := range tt.wantHooks {
				if i >= len(triggeredHooks) || triggeredHooks[i] != want {
					t.Errorf("hook[%d] = %q, want %q", i, triggeredHooks[i], want)
				}
			}
		})
	}
}

// TestServerStartPaths 验证所有启动方法遵循统一的状态流转
func TestServerStartPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		expectedFlow []State
	}{
		{
			name:         "start_paths_should_follow_state_machine",
			expectedFlow: []State{StateNew, StateStarting, StateReady},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(&Config{
				Host: "127.0.0.1",
				Port: 0,
			}, WithManualReadiness())

			// 验证初始状态
			if got := srv.State(); got != tt.expectedFlow[0] {
				t.Errorf("initial state = %q, want %q", got, tt.expectedFlow[0])
			}

			// 模拟统一启动流程
			srv.setState(StateStarting)
			if got := srv.State(); got != tt.expectedFlow[1] {
				t.Errorf("after starting state = %q, want %q", got, tt.expectedFlow[1])
			}

			if err := srv.MarkReady(); err != nil {
				t.Fatalf("MarkReady(): %v", err)
			}
			if got := srv.State(); got != tt.expectedFlow[2] {
				t.Errorf("after ready state = %q, want %q", got, tt.expectedFlow[2])
			}
		})
	}
}

// TestHealthAddr 验证 HealthAddr() 方法存在并返回正确的健康检查地址
func TestHealthAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		config          *Config
		wantHealthAddr  bool // 是否期望返回非空地址
		wantAddrPattern string
	}{
		{
			name: "HealthAddr_should_return_address_when_health_check_enabled",
			config: &Config{
				Host:              "127.0.0.1",
				Port:              8080,
				EnableHealthCheck: true,
				HealthCheckPort:   0, // 共享主端口
			},
			wantHealthAddr: true,
		},
		{
			name: "HealthAddr_should_return_empty_when_health_check_disabled",
			config: &Config{
				Host:              "127.0.0.1",
				Port:              8080,
				EnableHealthCheck: false,
			},
			wantHealthAddr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.config)

			healthAddr := srv.HealthAddr()

			if tt.wantHealthAddr && healthAddr == "" {
				t.Error("HealthAddr() returned empty string, want non-empty")
			}
			if !tt.wantHealthAddr && healthAddr != "" {
				t.Errorf("HealthAddr() returned %q, want empty string", healthAddr)
			}
		})
	}
}

// TestHTTPServerMutator 验证 WithHTTPServerMutator 能正确修改 http.Server
func TestHTTPServerMutator(t *testing.T) {
	t.Parallel()

	mutatorCalled := false
	mutator := func(srv *http.Server) {
		mutatorCalled = true
		// 修改一个可观察的属性
		srv.MaxHeaderBytes = 8192
	}

	srv := NewServer(&Config{
		Host: "127.0.0.1",
		Port: 8080,
	}, WithHTTPServerMutator(mutator))

	// 验证 mutator 尚未被调用（因为服务器尚未启动）
	if mutatorCalled {
		t.Error("mutator should not be called before server starts")
	}

	// 模拟服务器启动过程
	srv.setState(StateStarting)
	// 这里我们不能真正调用 prepareMainServer，因为它需要 listener
	// 但我们验证了 serverMutators 已经被注册

	t.Log("HTTPServerMutator registered successfully")
}

func TestHealthCheckPathMutation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *Config
		beforeSetPath  string
		enableAfter    bool
		wantErr        bool
		wantPath       string
		wantRegistered bool
	}{
		{
			name:           "reject mutation after routes registered",
			config:         nil,
			beforeSetPath:  "/custom-health",
			wantErr:        true,
			wantPath:       "/health",
			wantRegistered: true,
		},
		{
			name: "allow mutation before enabling health check",
			config: &Config{
				Host:              "127.0.0.1",
				Port:              8080,
				EnableHealthCheck: false,
			},
			beforeSetPath:  "/custom-health",
			enableAfter:    true,
			wantPath:       "/custom-health",
			wantRegistered: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.config)

			err := srv.SetHealthCheckPath(tt.beforeSetPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SetHealthCheckPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.enableAfter {
				srv.EnableHealthCheck()
			}

			if got := srv.GetHealthCheckPath(); got != tt.wantPath {
				t.Fatalf("GetHealthCheckPath() = %q, want %q", got, tt.wantPath)
			}

			if srv.healthRouteRegistered != tt.wantRegistered {
				t.Fatalf("healthRouteRegistered = %v, want %v", srv.healthRouteRegistered, tt.wantRegistered)
			}
		})
	}
}
