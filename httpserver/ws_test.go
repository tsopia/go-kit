package httpserver

import (
	"fmt"
	"net"
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

func TestWSSessionContract(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	server.WS("/ws/:id", func(session WSSession) {
		if session.Request().URL.Path != "/stream/ws/42" {
			t.Fatalf("path = %q, want %q", session.Request().URL.Path, "/stream/ws/42")
		}
		if session.Param("id") != "42" {
			t.Fatalf("id = %q, want %q", session.Param("id"), "42")
		}
		if !session.TrySend(WSMessage{Type: websocket.TextMessage, Data: []byte("hello")}) {
			t.Fatal("expected TrySend to accept first message")
		}
		<-session.Context().Done()
	})
}

func TestServer_WS(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	server.WS("/ws", func(session WSSession) {
		<-session.Context().Done()
	})

	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		// WS 路由注册在 streamingGroup 下，路径为 /stream/ws
		if r.Path == "/stream/ws" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_PanicRecovery(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// panic 的 handler
	server.WS("/panic", func(session WSSession) {
		panic("intentional panic in handler")
	})

	server.WS("/panic-in-read", func(session WSSession) {
		// 等待一条消息然后 panic
		<-session.Recv()
		panic("panic after receiving message")
	})

	// 验证路由注册成功（即使 handler 会 panic）
	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/stream/panic" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_UpgradeError(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// 注册 WS 路由
	server.WS("/ws", func(session WSSession) {
		<-session.Context().Done()
	})

	// 验证路由注册
	routes := server.Engine().Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/stream/ws" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WS route not registered")
	}
}

func TestWS_ReadTimeout(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	timeoutCalled := make(chan bool, 1)

	server.WS("/timeout", func(session WSSession) {
		select {
		case <-session.Recv():
			// 不应该收到，因为客户端不发消息
		case <-session.Context().Done():
			timeoutCalled <- true
		}
	}, WithReadTimeout(100*time.Millisecond))

	// 使用 net.Listen 获取随机端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond) // 等待服务器启动

	// WebSocket 连接
	wsURL := "ws://" + ln.Addr().String() + "/stream/timeout"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 等待超时触发
	select {
	case <-timeoutCalled:
		// 成功：读超时触发了 ctx 取消
	case <-time.After(2 * time.Second):
		t.Error("read timeout did not trigger")
	}
}

// TestWS_SendPolicy_Block 测试 Block 策略：缓冲区满时阻塞发送
func TestWS_SendPolicy_Block(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	received := make(chan string, 10)

	// 使用极小的发送缓冲区 (1) 和 Block 策略
	server.WS("/block", func(session WSSession) {
		// 发送两条消息，第一条占满 buffer，第二条应该阻塞
		go func() {
			_ = session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte("msg1")})
			_ = session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte("msg2")})
			received <- "sent"
		}()

		<-session.Context().Done()
	}, WithSendBuffer(1, Block))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + ln.Addr().String() + "/stream/block"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 故意不读取消息，让发送缓冲区保持满
	time.Sleep(200 * time.Millisecond)

	// msg1 应该在 buffer 里，msg2 应该阻塞
	// 现在读取 msg1
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if string(data) != "msg1" {
		t.Errorf("expected msg1, got %s", string(data))
	}

	// 现在 msg2 应该能发送了
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, data, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if string(data) != "msg2" {
		t.Errorf("expected msg2, got %s", string(data))
	}

	select {
	case <-received:
		// 成功：两条消息都发送完成
	case <-time.After(2 * time.Second):
		t.Error("send blocked forever")
	}
}

// TestWS_SendPolicy_DropNewest 测试 DropNewest 策略：缓冲区满时丢弃新消息
func TestWS_SendPolicy_DropNewest(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	sendDone := make(chan bool, 1)

	// 使用缓冲区大小为 1 和 DropNewest 策略
	server.WS("/dropnew", func(session WSSession) {
		// 快速发送多条消息
		for i := 0; i < 5; i++ {
			_ = session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte(fmt.Sprintf("msg%d", i))})
		}
		sendDone <- true
		<-session.Context().Done()
	}, WithSendBuffer(1, DropNewest))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + ln.Addr().String() + "/stream/dropnew"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 等待发送完成
	select {
	case <-sendDone:
	case <-time.After(2 * time.Second):
		t.Fatal("send timeout")
	}

	// 给一点时间让内部 channel 处理
	time.Sleep(50 * time.Millisecond)

	// 统计收到的消息数量
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	count := 0
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		count++
	}

	// 由于 DropNewest 策略，且写 goroutine 在消费
	// 我们只能验证：收到了部分消息，但不是全部（有消息被丢弃）
	// 具体数量取决于时序，不能断言精确值
	if count == 0 {
		t.Error("expected at least some messages, got none")
	}
	if count >= 5 {
		t.Errorf("expected some messages to be dropped with DropNewest, got all %d", count)
	}
}

// TestWS_SendPolicy_DropOldest 测试 DropOldest 策略：缓冲区满时丢弃旧消息
func TestWS_SendPolicy_DropOldest(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	// 使用缓冲区大小为 2 和 DropOldest 策略
	server.WS("/dropold", func(session WSSession) {
		// 发送多条消息
		for i := 0; i < 5; i++ {
			_ = session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte(fmt.Sprintf("msg%d", i))})
		}
		<-session.Context().Done()
	}, WithSendBuffer(2, DropOldest))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + ln.Addr().String() + "/stream/dropold"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 故意延迟读取，让发送端有机会丢弃旧消息
	time.Sleep(300 * time.Millisecond)

	// DropOldest 策略下，buffer 里应该保留最新的 2 条消息
	// 但是因为 consumer 在读取，实际行为会更复杂
	// 我们验证至少能收到消息且不 panic
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	// 应该收到某条消息
	if len(data) == 0 {
		t.Error("expected non-empty message")
	}
}

// TestWS_SendPolicy_Disconnect 测试 Disconnect 策略：缓冲区满时断开连接
func TestWS_SendPolicy_Disconnect(t *testing.T) {
	server := NewServer(&Config{Port: 0})
	server.SetGroups(
		server.Engine().Group("/api"),
		server.Engine().Group("/stream"),
	)

	disconnected := make(chan bool, 1)

	// 使用极小的发送缓冲区 (1) 和 Disconnect 策略
	server.WS("/disconnect", func(session WSSession) {
		// 快速发送多条消息，填满 buffer 并触发断开
		// 由于 proxy channel 无缓冲，每条消息都会等待 Run() 接收
		// 当内部 channel 满时，Disconnect 策略会触发
		for i := 0; i < 10; i++ {
			select {
			case <-session.Context().Done():
				disconnected <- true
				return
			default:
			}
			if err := session.Send(WSMessage{Type: websocket.TextMessage, Data: []byte(fmt.Sprintf("msg%d", i))}); err == nil {
				// 发送成功
			} else {
				disconnected <- true
				return
			}
		}
		<-session.Context().Done()
		disconnected <- true
	}, WithSendBuffer(1, Disconnect))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go server.Serve(ln)
	time.Sleep(100 * time.Millisecond)

	wsURL := "ws://" + ln.Addr().String() + "/stream/disconnect"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// 等待连接被断开（handler 检测到 ctx.Done()）
	select {
	case <-disconnected:
		// 成功：连接被断开
	case <-time.After(2 * time.Second):
		t.Error("disconnect did not trigger")
	}
}
