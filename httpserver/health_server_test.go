package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestReadinessAndLivenessEndpoints(t *testing.T) {
	tests := []struct {
		name            string
		opts            []Option
		action          func(*Server)
		wantReadyStatus int
		wantLiveStatus  int
	}{
		{
			name:            "auto ready server returns ready",
			wantReadyStatus: http.StatusOK,
			wantLiveStatus:  http.StatusOK,
		},
		{
			name:            "manual readiness starts unready",
			opts:            []Option{WithManualReadiness()},
			wantReadyStatus: http.StatusServiceUnavailable,
			wantLiveStatus:  http.StatusOK,
		},
		{
			name: "draining server becomes unready",
			action: func(s *Server) {
				if err := s.MarkDraining(); err != nil {
					t.Fatalf("MarkDraining(): %v", err)
				}
			},
			wantReadyStatus: http.StatusServiceUnavailable,
			wantLiveStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appPort := freeTCPPort(t)
			srv := NewServer(&Config{
				Host:              "127.0.0.1",
				Port:              appPort,
				EnableHealthCheck: true,
				HealthCheckPath:   "/healthz",
				ReadinessPath:     "/readyz",
				LivenessPath:      "/livez",
			}, tt.opts...)

			if err := srv.Start(); err != nil {
				t.Fatalf("start: %v", err)
			}
			defer func() {
				if err := srv.Shutdown(context.Background()); err != nil {
					t.Fatalf("shutdown: %v", err)
				}
			}()

			if tt.action != nil {
				tt.action(srv)
			}

			waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/readyz", appPort), tt.wantReadyStatus)
			waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/livez", appPort), tt.wantLiveStatus)
		})
	}
}

func TestHealthCheckServingModes(t *testing.T) {
	tests := []struct {
		name                 string
		healthPort           int
		wantMainHealthStatus int
	}{
		{
			name:                 "shared port",
			healthPort:           0,
			wantMainHealthStatus: http.StatusOK,
		},
		{
			name:                 "dedicated port",
			healthPort:           freeTCPPort(t),
			wantMainHealthStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appPort := freeTCPPort(t)
			srv := NewServer(&Config{
				Host:              "127.0.0.1",
				Port:              appPort,
				EnableHealthCheck: true,
				HealthCheckPath:   "/healthz",
				HealthCheckPort:   tt.healthPort,
			})
			srv.GET("/users/ping", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			if err := srv.Start(); err != nil {
				t.Fatalf("start: %v", err)
			}
			defer func() {
				if err := srv.Shutdown(context.Background()); err != nil {
					t.Fatalf("shutdown: %v", err)
				}
			}()

			waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/users/ping", appPort), http.StatusNoContent)

			healthPort := appPort
			if tt.healthPort != 0 {
				healthPort = tt.healthPort
			}

			waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/healthz", healthPort), http.StatusOK)
			waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/healthz", appPort), tt.wantMainHealthStatus)
		})
	}
}

func TestServeStartsDedicatedHealthServer(t *testing.T) {
	appListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen app: %v", err)
	}

	appPort := appListener.Addr().(*net.TCPAddr).Port
	healthPort := freeTCPPort(t)
	srv := NewServer(&Config{
		Host:              "127.0.0.1",
		Port:              appPort,
		EnableHealthCheck: true,
		HealthCheckPath:   "/healthz",
		HealthCheckPort:   healthPort,
	})
	srv.GET("/users/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(appListener)
	}()

	waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/users/ping", appPort), http.StatusNoContent)
	waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/healthz", healthPort), http.StatusOK)
	waitForHTTPStatus(t, fmt.Sprintf("http://127.0.0.1:%d/healthz", appPort), http.StatusNotFound)

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if err := <-serveErrCh; err != nil {
		t.Fatalf("serve returned error: %v", err)
	}
}

func TestRunWithContextMarksServerUnreadyBeforeShutdown(t *testing.T) {
	appPort := freeTCPPort(t)
	srv := NewServer(&Config{
		Host:              "127.0.0.1",
		Port:              appPort,
		EnableHealthCheck: true,
		HealthCheckPath:   "/healthz",
		ReadinessPath:     "/readyz",
		LivenessPath:      "/livez",
		DrainTimeout:      150 * time.Millisecond,
		ShutdownTimeout:   time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.RunWithContext(ctx)
	}()

	readyURL := fmt.Sprintf("http://127.0.0.1:%d/readyz", appPort)
	waitForHTTPStatus(t, readyURL, http.StatusOK)

	cancel()
	waitForHTTPStatusBeforeShutdown(t, readyURL, done, http.StatusServiceUnavailable)

	if err := <-done; err != nil {
		t.Fatalf("RunWithContext() error = %v", err)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Fatalf("close free port listener: %v", err)
		}
	}()

	return ln.Addr().(*net.TCPAddr).Port
}

func waitForHTTPStatus(t *testing.T, url string, wantStatus int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	client := &http.Client{Timeout: 200 * time.Millisecond}
	var lastErr error
	var lastStatus int

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			lastStatus = resp.StatusCode
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("close response body: %v", err)
			}
			if resp.StatusCode == wantStatus {
				return
			}
		} else {
			lastErr = err
		}

		time.Sleep(20 * time.Millisecond)
	}

	if lastErr != nil && lastStatus == 0 {
		t.Fatalf("request %s: %v", url, lastErr)
	}

	t.Fatalf("request %s: got status %d, want %d", url, lastStatus, wantStatus)
}

func waitForHTTPStatusBeforeShutdown(t *testing.T, url string, done <-chan error, wantStatus int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	client := &http.Client{Timeout: 200 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("server exited before status %d was observed: %v", wantStatus, err)
			}
			t.Fatalf("server exited before status %d was observed", wantStatus)
		default:
		}

		resp, err := client.Get(url)
		if err == nil {
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("close response body: %v", err)
			}
			if resp.StatusCode == wantStatus {
				return
			}
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("request %s: status %d was not observed before shutdown", url, wantStatus)
}
