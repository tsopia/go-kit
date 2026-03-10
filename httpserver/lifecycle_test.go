package httpserver

import (
	"context"
	"net"
	"testing"
)

func TestServerStartReturnsListenError(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

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
