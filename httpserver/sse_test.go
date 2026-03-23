package httpserver

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSSESender_Event(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Event("update", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "event: update\ndata: {\"key\":\"value\"}\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Data(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Data("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := "data: hello\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestSSESender_Comment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sender := &sseSender{ginCtx: c}
	err := sender.Comment("heartbeat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := w.Body.String()
	want := ": heartbeat\n\n"
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}
