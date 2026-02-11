package websocket

import (
	"encoding/json"
	"sync"
)

const (
	EventUserConnected    = "USER_CONNECTED"
	EventUserDisconnected = "USER_DISCONNECTED"
	EventDeviceOnline     = "DEVICE_ONLINE"
	EventDeviceOffline    = "DEVICE_OFFLINE"
	EventMessageFlow     = "MESSAGE_FLOW"
	EventFileTransfer    = "FILE_TRANSFER"
	EventChannelActivity = "CHANNEL_ACTIVITY"
	EventSystemHealth    = "SYSTEM_HEALTH"
	EventClusterStatus   = "CLUSTER_STATUS"
)

type Client struct {
	ID     string
	Send   chan []byte
	UserID string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[*Client]bool),
		broadcast: make(chan []byte, 256),
		register:  make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.BroadcastEvent(EventUserConnected, map[string]string{"user_id": c.UserID, "client_id": c.ID})

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
			h.BroadcastEvent(EventUserDisconnected, map[string]string{"user_id": c.UserID, "client_id": c.ID})

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.Send <- msg:
				default:
					close(c.Send)
					delete(h.clients, c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastEvent(event string, payload interface{}) {
	b, err := json.Marshal(map[string]interface{}{"event": event, "payload": payload})
	if err != nil {
		return
	}
	h.broadcast <- b
}

func (h *Hub) Emit(event string, payload interface{}) {
	h.BroadcastEvent(event, payload)
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	n := len(h.clients)
	h.mu.RUnlock()
	return n
}
