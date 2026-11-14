package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	gorilla "github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 50 * time.Second
	maxMessageSize = 512
)

type Client struct {
	hub  *Hub
	conn *gorilla.Conn
	send chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	allowedOrigins map[string]bool
}

func NewHub(origins ...string) *Hub {
	allowed := make(map[string]bool)

	for _, origin := range origins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin != "" {
			allowed[origin] = true
		}
	}

	return &Hub{
		clients:        make(map[*Client]bool),
		register:       make(chan *Client, 32),
		unregister:     make(chan *Client, 32),
		broadcast:      make(chan []byte, 256),
		allowedOrigins: allowed,
	}
}

// Run manages all WebSocket clients.
// Start this once when the application starts.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Slow client: disconnect it.
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

// CheckOrigin validates the browser's WebSocket origin.
func (h *Hub) checkOrigin(r *http.Request) bool {
	origin := strings.TrimRight(
		strings.TrimSpace(r.Header.Get("Origin")),
		"/",
	)

	if origin == "" {
		log.Println("WebSocket rejected: missing Origin header")
		return false
	}

	allowedOrigins := map[string]bool{
		"https://geofencing-real-time-alert-system-frontend-dz4m-54nwpgh2k.vercel.app": true,
		"https://geofencing-real-time-git-1e38d4-prattyanchas-projects-4bd1d7b3.vercel.app": true,
		"http://localhost:5173": true,
	}

	// Check configured origins.
	if h.allowedOrigins[origin] || allowedOrigins[origin] {
		log.Printf("WebSocket origin accepted: %s", origin)
		return true
	}

	log.Printf(
		"WebSocket rejected: origin %q not allowed. Configured origins: %v",
		origin,
		h.allowedOrigins,
	)

	return false
}

// WebSocket upgrader.
func (h *Hub) upgrader() *gorilla.Upgrader {
	return &gorilla.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
}

// ServeHTTP handles incoming WebSocket connections.

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf(
		"WebSocket request: method=%s path=%s origin=%s",
		r.Method,
		r.URL.Path,
		r.Header.Get("Origin"),
	)

	conn, err := h.upgrader().Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("WebSocket connected: %s", r.RemoteAddr)

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Broadcast sends an alert to all connected clients.
func (h *Hub) Broadcast(message any) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket JSON error: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Println("WebSocket broadcast queue full; message dropped")
	}
}

// readPump handles incoming messages and connection health.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump sends messages and periodic ping frames.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				_ = c.conn.WriteMessage(
					gorilla.CloseMessage,
					[]byte{},
				)
				return
			}

			if err := c.conn.WriteMessage(
				gorilla.TextMessage,
				message,
			); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(
				gorilla.PingMessage,
				nil,
			); err != nil {
				return
			}
		}
	}
}
