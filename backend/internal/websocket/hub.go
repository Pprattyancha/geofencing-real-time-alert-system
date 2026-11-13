package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type Hub struct {
	mu       sync.RWMutex
	writeMu  sync.Mutex
	clients  map[*websocket.Conn]bool
	upgrader websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{clients: map[*websocket.Conn]bool{}, upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}}
}
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.clients, conn); h.mu.Unlock(); _ = conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	for {
		if _, _, err = conn.ReadMessage(); err != nil {
			break
		}
	}
}
func (h *Hub) Broadcast(v any) {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("marshal websocket alert: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			h.mu.Lock()
			delete(h.clients, c)
			h.mu.Unlock()
			_ = c.Close()
		}
	}
}
