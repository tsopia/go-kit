package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["email"] != "foo@example.com" {
		t.Fatalf("expected email to round-trip, got %q", resp["email"])
	}
}

func TestHandleJSONValidateError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}
}

func TestHandleJSONDecodeError(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(func(ctx context.Context, req loginRequest) (gin.H, error) {
		return gin.H{"email": req.Email}, nil
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleJSONCustomErrorMapper(t *testing.T) {
	sentinel := errors.New("duplicate user")

	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(
		func(ctx context.Context, req loginRequest) (gin.H, error) {
			return nil, sentinel
		},
		WithErrorMapper(func(err error) (int, any) {
			if errors.Is(err, sentinel) {
				return http.StatusConflict, gin.H{"error": "duplicate"}
			}

			return http.StatusInternalServerError, gin.H{"error": err.Error()}
		}),
	))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestHandleJSONCustomEncoderAndSuccessStatus(t *testing.T) {
	srv := NewServer(nil)
	srv.POST("/login", HandleJSON(
		func(ctx context.Context, req loginRequest) (string, error) {
			return "token-123", nil
		},
		WithSuccessStatus(http.StatusAccepted),
		WithEncoder(func(c *gin.Context, status int, resp any) {
			token, _ := resp.(string)
			c.String(status, "token=%s", token)
		}),
	))

	w := performJSONRequest(t, srv, http.MethodPost, "/login", gin.H{"email": "foo@example.com"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
	if w.Body.String() != "token=token-123" {
		t.Fatalf("expected custom encoder body, got %q", w.Body.String())
	}
}

func performJSONRequest(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)

	return w
}
