package httpserver

import (
	"net"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWS_GetFromGinKeys(t *testing.T) {
	srv := NewServer(&Config{Port: 0})
	// 模拟鉴权中间件：c.Set("user", ...)
	srv.Use(func(c *gin.Context) {
		c.Set("user", "alice")
		c.Next()
	})
	srv.SetGroups(srv.Engine().Group("/api"), srv.Engine().Group("/stream"))

	got := make(chan string, 1)
	srv.WS("/ws", func(session WSSession) {
		if v, ok := session.GetString("user"); ok {
			got <- v
		} else {
			got <- ""
		}
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() { _ = srv.Serve(ln) }()
	time.Sleep(100 * time.Millisecond)

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/stream/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case v := <-got:
		if v != "alice" {
			t.Errorf("GetString(user) = %q, want %q", v, "alice")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not run")
	}
}
