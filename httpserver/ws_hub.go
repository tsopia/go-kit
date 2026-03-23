package httpserver

import (
	"sync"
)

// WSHub 管理 WebSocket 连接的分组（如聊天室）
type WSHub struct {
	mu    sync.RWMutex
	rooms map[string]map[chan<- WSMessage]struct{}
}

// NewWSHub 创建一个新的 WSHub
func NewWSHub() *WSHub {
	return &WSHub{
		rooms: make(map[string]map[chan<- WSMessage]struct{}),
	}
}

// Join 将连接加入指定房间
func (h *WSHub) Join(roomID string, send chan<- WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[chan<- WSMessage]struct{})
	}
	h.rooms[roomID][send] = struct{}{}
}

// Leave 将连接从房间移除
func (h *WSHub) Leave(roomID string, send chan<- WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, ok := h.rooms[roomID]; ok {
		delete(members, send)
		if len(members) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

// Broadcast 向房间所有连接广播消息
func (h *WSHub) Broadcast(roomID string, msg WSMessage) {
	h.mu.RLock()
	members := h.rooms[roomID]
	h.mu.RUnlock()

	for send := range members {
		select {
		case send <- msg:
		default:
			// 非阻塞发送，失败则丢弃
		}
	}
}

// RoomCount 返回房间内连接数
func (h *WSHub) RoomCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}
