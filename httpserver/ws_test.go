package httpserver

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSBufferPolicy_Constants(t *testing.T) {
	if Block != 0 {
		t.Error("Block should be 0")
	}
	if DropNewest != 1 {
		t.Error("DropNewest should be 1")
	}
	if DropOldest != 2 {
		t.Error("DropOldest should be 2")
	}
	if Disconnect != 3 {
		t.Error("Disconnect should be 3")
	}
}

func TestWSConfig_Defaults(t *testing.T) {
	cfg := defaultWSConfig()
	if cfg.RecvBufferSize != 100 {
		t.Errorf("expected RecvBufferSize=100, got %d", cfg.RecvBufferSize)
	}
	if cfg.SendBufferSize != 100 {
		t.Errorf("expected SendBufferSize=100, got %d", cfg.SendBufferSize)
	}
	if cfg.RecvPolicy != DropNewest {
		t.Error("expected RecvPolicy=DropNewest")
	}
	if cfg.SendPolicy != DropOldest {
		t.Error("expected SendPolicy=DropOldest")
	}
	if cfg.PingPeriod != 30*time.Second {
		t.Errorf("expected PingPeriod=30s, got %v", cfg.PingPeriod)
	}
	if cfg.PongTimeout != 60*time.Second {
		t.Errorf("expected PongTimeout=60s, got %v", cfg.PongTimeout)
	}
}

func TestWSMessage(t *testing.T) {
	msg := WSMessage{
		Type: websocket.TextMessage,
		Data: []byte("hello"),
	}
	if msg.Type != websocket.TextMessage {
		t.Error("Type mismatch")
	}
	if string(msg.Data) != "hello" {
		t.Error("Data mismatch")
	}
}

func TestWSOptions(t *testing.T) {
	cfg := defaultWSConfig()

	opt1 := WithRecvBuffer(200, Block)
	opt1.apply(&cfg)
	if cfg.RecvBufferSize != 200 || cfg.RecvPolicy != Block {
		t.Error("WithRecvBuffer failed")
	}

	opt2 := WithWSPingPeriod(10 * time.Second)
	opt2.apply(&cfg)
	if cfg.PingPeriod != 10*time.Second {
		t.Error("WithWSPingPeriod failed")
	}

	opt3 := WithSendBuffer(300, DropOldest)
	opt3.apply(&cfg)
	if cfg.SendBufferSize != 300 || cfg.SendPolicy != DropOldest {
		t.Error("WithSendBuffer failed")
	}

	opt4 := WithWSPongTimeout(20 * time.Second)
	opt4.apply(&cfg)
	if cfg.PongTimeout != 20*time.Second {
		t.Error("WithWSPongTimeout failed")
	}
}
