package websocket

import (
	"context"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Channel defines the interface for WebSocket channels.
type Channel interface {
	OnConnect(ctx *WSContext) error
	OnMessage(ctx *WSContext, msg []byte) error
	OnDisconnect(ctx *WSContext) error
}

// WSContext wraps a WebSocket connection with room and hub support.
type WSContext struct {
	Conn *websocket.Conn
	Hub  *Hub
	ctx  context.Context
}

// Send sends a message to this connection.
func (c *WSContext) Send(msg any) error {
	return wsjson.Write(c.ctx, c.Conn, msg)
}

// JoinRoom joins a named room.
func (c *WSContext) JoinRoom(room string) {
	c.Hub.JoinRoom(room, c)
}

// LeaveRoom leaves a named room.
func (c *WSContext) LeaveRoom(room string) {
	c.Hub.LeaveRoom(room, c)
}

// Hub manages WebSocket connections and rooms.
type Hub struct {
	mu          sync.RWMutex
	connections map[*websocket.Conn]bool
	rooms       map[string]map[*WSContext]bool
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		connections: make(map[*websocket.Conn]bool),
		rooms:       make(map[string]map[*WSContext]bool),
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.connections {
		wsjson.Write(context.Background(), conn, msg)
	}
}

// BroadcastToRoom sends a message to all clients in a room.
func (h *Hub) BroadcastToRoom(room string, msg any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if conns, ok := h.rooms[room]; ok {
		for ctx := range conns {
			ctx.Send(msg)
		}
	}
}

// JoinRoom adds a client to a room.
func (h *Hub) JoinRoom(room string, ctx *WSContext) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*WSContext]bool)
	}
	h.rooms[room][ctx] = true
}

// LeaveRoom removes a client from a room.
func (h *Hub) LeaveRoom(room string, ctx *WSContext) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[room]; ok {
		delete(conns, ctx)
	}
}

// HandleChannel creates an HTTP handler for a Channel interface.
func (h *Hub) HandleChannel(ch Channel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusInternalError, "closing")

		h.mu.Lock()
		h.connections[c] = true
		h.mu.Unlock()

		defer func() {
			h.mu.Lock()
			delete(h.connections, c)
			h.mu.Unlock()
		}()

		wsCtx := &WSContext{Conn: c, Hub: h, ctx: r.Context()}

		if err := ch.OnConnect(wsCtx); err != nil {
			return
		}

		defer ch.OnDisconnect(wsCtx)

		for {
			_, data, readErr := c.Read(r.Context())
			if readErr != nil {
				break
			}
			if msgErr := ch.OnMessage(wsCtx, data); msgErr != nil {
				break
			}
		}
	}
}

// Handle provides a simple WebSocket handler without the Channel interface.
func (h *Hub) Handle(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "closing")

	h.mu.Lock()
	h.connections[c] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.connections, c)
		h.mu.Unlock()
	}()

	for {
		var v interface{}
		err = wsjson.Read(r.Context(), c, &v)
		if err != nil {
			break
		}
	}
}
