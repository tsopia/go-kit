package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket服务器
type WebSocketServer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mutex    sync.RWMutex
	server   *http.Server
}

// 创建WebSocket服务器
func NewWebSocketServer(port int) *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境需要更严格的检查
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

// 启动WebSocket服务器
func (ws *WebSocketServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", 8081), // WebSocket端口
		Handler: mux,
	}

	fmt.Printf("🚀 WebSocket服务器启动: ws://localhost:%d/ws\n", 8081)

	// 在goroutine中启动WebSocket服务器
	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket服务器启动失败: %v", err)
		}
	}()

	return nil
}

// 处理WebSocket连接
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 添加客户端
	ws.mutex.Lock()
	ws.clients[conn] = true
	ws.mutex.Unlock()

	fmt.Printf("✅ WebSocket客户端连接: %s\n", conn.RemoteAddr())

	// 发送欢迎消息
	conn.WriteMessage(websocket.TextMessage, []byte("欢迎连接到WebSocket服务器！"))

	// 处理消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("❌ WebSocket客户端断开: %s, 错误: %v\n", conn.RemoteAddr(), err)
			break
		}

		// 回显消息
		response := fmt.Sprintf("收到消息: %s", string(message))
		conn.WriteMessage(messageType, []byte(response))
	}

	// 移除客户端
	ws.mutex.Lock()
	delete(ws.clients, conn)
	ws.mutex.Unlock()

	conn.Close()
}

// 广播消息给所有客户端
func (ws *WebSocketServer) Broadcast(message string) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for client := range ws.clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			fmt.Printf("广播消息失败: %v\n", err)
		}
	}
}

// 获取客户端数量
func (ws *WebSocketServer) GetClientCount() int {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	return len(ws.clients)
}

// 优雅关闭WebSocket服务器
func (ws *WebSocketServer) Shutdown(ctx context.Context) error {
	if ws.server == nil {
		return nil
	}

	fmt.Println("🔄 开始关闭WebSocket服务器...")

	// 关闭所有WebSocket连接
	ws.mutex.Lock()
	for client := range ws.clients {
		client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "服务器关闭"))
		client.Close()
	}
	ws.clients = make(map[*websocket.Conn]bool)
	ws.mutex.Unlock()

	// 关闭HTTP服务器
	if err := ws.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("WebSocket服务器关闭失败: %w", err)
	}

	fmt.Println("✅ WebSocket服务器已关闭")
	return nil
}

// HTTP服务器
type HTTPServer struct {
	engine *gin.Engine
	server *http.Server
}

// 创建HTTP服务器
func NewHTTPServer(port int) *HTTPServer {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	return &HTTPServer{
		engine: engine,
	}
}

// 启动HTTP服务器
func (hs *HTTPServer) Start() error {
	hs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", 8080),
		Handler: hs.engine,
	}

	fmt.Printf("🚀 HTTP服务器启动: http://localhost:%d\n", 8080)

	// 在goroutine中启动HTTP服务器
	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP服务器启动失败: %v", err)
		}
	}()

	return nil
}

// 优雅关闭HTTP服务器
func (hs *HTTPServer) Shutdown(ctx context.Context) error {
	if hs.server == nil {
		return nil
	}

	fmt.Println("🔄 开始关闭HTTP服务器...")

	if err := hs.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP服务器关闭失败: %w", err)
	}

	fmt.Println("✅ HTTP服务器已关闭")
	return nil
}

// 注册路由
func (hs *HTTPServer) RegisterRoutes(routes func(r *gin.Engine)) {
	routes(hs.engine)
}

// 复合服务器：包含HTTP和WebSocket
type CompositeServer struct {
	httpServer *HTTPServer
	wsServer   *WebSocketServer
}

// 创建复合服务器
func NewCompositeServer() *CompositeServer {
	return &CompositeServer{
		httpServer: NewHTTPServer(8080),
		wsServer:   NewWebSocketServer(8081),
	}
}

// 启动复合服务器
func (cs *CompositeServer) Start() error {
	// 启动WebSocket服务器
	if err := cs.wsServer.Start(); err != nil {
		return fmt.Errorf("WebSocket服务器启动失败: %w", err)
	}

	// 启动HTTP服务器
	if err := cs.httpServer.Start(); err != nil {
		return fmt.Errorf("HTTP服务器启动失败: %w", err)
	}

	return nil
}

// 优雅关闭复合服务器
func (cs *CompositeServer) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var httpErr, wsErr error

	// 并行关闭两个服务器
	wg.Add(2)

	go func() {
		defer wg.Done()
		httpErr = cs.httpServer.Shutdown(ctx)
	}()

	go func() {
		defer wg.Done()
		wsErr = cs.wsServer.Shutdown(ctx)
	}()

	wg.Wait()

	// 返回第一个错误
	if httpErr != nil {
		return httpErr
	}
	return wsErr
}

// 等待关闭信号
func (cs *CompositeServer) WaitForShutdown() error {
	quit := make(chan struct{})
	go func() {
		// 监听系统信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("收到关闭信号，开始优雅关闭所有服务器...")
		close(quit)
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return cs.Shutdown(ctx)
}

func main() {
	fmt.Println("=== HTTP + WebSocket 复合服务器优雅关闭演示 ===")
	fmt.Println()

	// 创建复合服务器
	server := NewCompositeServer()

	// 注册HTTP路由
	server.httpServer.RegisterRoutes(func(r *gin.Engine) {
		r.GET("/", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "HTTP + WebSocket 复合服务器",
				"endpoints": gin.H{
					"http":      "http://localhost:8080",
					"websocket": "ws://localhost:8081/ws",
				},
				"ws_clients": server.wsServer.GetClientCount(),
				"time":       time.Now().Format("15:04:05"),
			})
		})

		r.GET("/ws-status", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"ws_clients": server.wsServer.GetClientCount(),
				"status":     "running",
			})
		})

		r.POST("/broadcast", func(c *gin.Context) {
			var req struct {
				Message string `json:"message" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "参数错误"})
				return
			}

			server.wsServer.Broadcast(req.Message)
			c.JSON(200, gin.H{
				"message": "广播成功",
				"clients": server.wsServer.GetClientCount(),
			})
		})

		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "healthy",
				"services": gin.H{
					"http":      "running",
					"websocket": "running",
				},
				"ws_clients": server.wsServer.GetClientCount(),
			})
		})
	})

	// 启动服务器
	fmt.Println("🚀 启动复合服务器...")
	if err := server.Start(); err != nil {
		log.Fatal("服务器启动失败:", err)
	}

	fmt.Println("✅ 所有服务器已启动")
	fmt.Println("📡 服务地址:")
	fmt.Println("   HTTP服务器: http://localhost:8080")
	fmt.Println("   WebSocket服务器: ws://localhost:8081/ws")
	fmt.Println("   WebSocket状态: http://localhost:8080/ws-status")
	fmt.Println("   广播消息: POST http://localhost:8080/broadcast")
	fmt.Println("   健康检查: http://localhost:8080/health")
	fmt.Println()
	fmt.Println("💡 按 Ctrl+C 优雅关闭所有服务器")

	// 等待关闭信号
	if err := server.WaitForShutdown(); err != nil {
		log.Fatal("服务器关闭失败:", err)
	}

	fmt.Println("✅ 所有服务器已优雅关闭")
}
