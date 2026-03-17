package httpserver

import (
	"context"
	"net"
	"testing"
)

func TestServerStateTransitions(t *testing.T) {
	t.Parallel()

	srv := NewServer(nil, WithManualReadiness())

	if got := srv.State(); got != StateNew {
		t.Fatalf("initial state = %q, want %q", got, StateNew)
	}

	srv.MarkReady()
	if got := srv.State(); got != StateReady {
		t.Fatalf("state after MarkReady = %q, want %q", got, StateReady)
	}

	srv.MarkDraining()
	if got := srv.State(); got != StateDraining {
		t.Fatalf("state after MarkDraining = %q, want %q", got, StateDraining)
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
