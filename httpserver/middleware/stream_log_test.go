package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tsopia/go-kit/utils"
)

type recordingObserver struct {
	connects    []string
	disconnects []string
}

func (o *recordingObserver) OnConnect(transport string)    { o.connects = append(o.connects, transport) }
func (o *recordingObserver) OnDisconnect(transport string) { o.disconnects = append(o.disconnects, transport) }

func TestStreamObserverContextRoundTrip(t *testing.T) {
	obs := &recordingObserver{}
	ctx := WithStreamObserver(context.Background(), obs)

	got, ok := StreamObserverFromContext(ctx)
	if !ok {
		t.Fatal("StreamObserverFromContext returned ok=false")
	}
	got.OnConnect("sse")
	if len(obs.connects) != 1 || obs.connects[0] != "sse" {
		t.Errorf("connects = %v, want [sse]", obs.connects)
	}
}

func TestStreamObserverFromContextMissing(t *testing.T) {
	if _, ok := StreamObserverFromContext(context.Background()); ok {
		t.Error("expected ok=false for empty context")
	}
}

func TestMarkStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	MarkStreaming("ws")(c)

	if got := c.GetString(utils.StreamingKey); got != "ws" {
		t.Errorf("StreamingKey = %q, want %q", got, "ws")
	}
}
