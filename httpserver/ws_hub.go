package httpserver

import (
	"sync"
)

type wsTrySender interface {
	TrySend(WSMessage) bool
}

// WSHub 管理 WebSocket 连接的分组（如聊天室）
type WSHub struct {
	mu    sync.RWMutex
	rooms map[string]map[wsTrySender]struct{}
}

// NewWSHub 创建一个新的 WSHub
func NewWSHub() *WSHub {
	return &WSHub{
		rooms: make(map[string]map[wsTrySender]struct{}),
	}
}

// Join 将连接加入指定房间
func (h *WSHub) Join(roomID string, send wsTrySender) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[wsTrySender]struct{})
	}
	h.rooms[roomID][send] = struct{}{}
}

// Leave 将连接从房间移除
func (h *WSHub) Leave(roomID string, send wsTrySender) {
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
	members := make([]wsTrySender, 0, len(h.rooms[roomID]))
	for send := range h.rooms[roomID] {
		members = append(members, send)
	}
	h.mu.RUnlock()

	for _, send := range members {
		send.TrySend(msg)
	}
}

// RoomCount 返回房间内连接数
func (h *WSHub) RoomCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}
