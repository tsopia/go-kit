package httpserver

import (
	"testing"
	"time"
)

type testWSSender struct {
	ch    chan WSMessage
	allow bool
}

func (s *testWSSender) TrySend(msg WSMessage) bool {
	if !s.allow {
		return false
	}
	select {
	case s.ch <- msg:
		return true
	default:
		return false
	}
}

func TestWSHub_JoinLeave(t *testing.T) {
	hub := NewWSHub()
	send := &testWSSender{ch: make(chan WSMessage, 10), allow: true}

	hub.Join("room1", send)
	if hub.RoomCount("room1") != 1 {
		t.Error("Join failed")
	}

	hub.Leave("room1", send)
	if hub.RoomCount("room1") != 0 {
		t.Error("Leave failed")
	}
}

func TestWSHub_UsesTrySend(t *testing.T) {
	hub := NewWSHub()
	sender := &testWSSender{ch: make(chan WSMessage, 1), allow: true}

	hub.Join("room1", sender)
	hub.Broadcast("room1", WSMessage{Type: 1, Data: []byte("hello")})

	select {
	case msg := <-sender.ch:
		if string(msg.Data) != "hello" {
			t.Fatalf("message = %q, want %q", string(msg.Data), "hello")
		}
	default:
		t.Fatal("expected one broadcast message")
	}
}

func TestWSHub_Broadcast(t *testing.T) {
	hub := NewWSHub()
	send1 := &testWSSender{ch: make(chan WSMessage, 10), allow: true}
	send2 := &testWSSender{ch: make(chan WSMessage, 10), allow: true}

	hub.Join("room1", send1)
	hub.Join("room1", send2)

	msg := WSMessage{Type: 1, Data: []byte("hello")}
	hub.Broadcast("room1", msg)

	select {
	case m := <-send1.ch:
		if string(m.Data) != "hello" {
			t.Error("Broadcast to send1 failed")
		}
	case <-time.After(time.Second):
		t.Error("Broadcast timeout to send1")
	}

	select {
	case m := <-send2.ch:
		if string(m.Data) != "hello" {
			t.Error("Broadcast to send2 failed")
		}
	case <-time.After(time.Second):
		t.Error("Broadcast timeout to send2")
	}
}

func TestWSHub_MultiRoom(t *testing.T) {
	hub := NewWSHub()
	send1 := &testWSSender{ch: make(chan WSMessage, 10), allow: true}
	send2 := &testWSSender{ch: make(chan WSMessage, 10), allow: true}

	hub.Join("room1", send1)
	hub.Join("room2", send2)

	hub.Broadcast("room1", WSMessage{Data: []byte("room1 msg")})

	select {
	case m := <-send1.ch:
		if string(m.Data) != "room1 msg" {
			t.Error("Wrong message in room1")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for room1 message")
	}

	select {
	case <-send2.ch:
		t.Error("room2 should not receive room1 message")
	case <-time.After(100 * time.Millisecond):
		// 正确：没有消息
	}
}

func TestWSHub_Broadcast_Concurrent(t *testing.T) {
	hub := NewWSHub()
	send := &testWSSender{ch: make(chan WSMessage, 10), allow: true}

	hub.Join("room1", send)

	// 并发执行 Broadcast 和 Leave
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			hub.Broadcast("room1", WSMessage{Data: []byte("msg")})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			hub.Leave("room1", send)
			hub.Join("room1", send)
		}
		done <- true
	}()

	// 等待完成或超时
	select {
	case <-done:
		select {
		case <-done:
			// 两个都完成
		case <-time.After(5 * time.Second):
			t.Error("timeout waiting for second goroutine")
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for first goroutine")
	}
}
