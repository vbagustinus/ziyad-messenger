package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"lan-chat/protocol"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client represents a connected user over WebSocket
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

// MessageRouter handles message routing, storage, and real-time delivery.
type MessageRouter struct {
	db      *sql.DB
	clients map[string][]*Client // UserID -> Multiple connections
	mu      sync.RWMutex
}

func NewMessageRouter(dbPath string) (*MessageRouter, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &MessageRouter{
		db:      db,
		clients: make(map[string][]*Client),
	}, nil
}

func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		channel_id TEXT,
		sender_id TEXT,
		timestamp INTEGER,
		type INTEGER,
		content BLOB,
		nonce BLOB,
		signature BLOB
	);
	CREATE INDEX IF NOT EXISTS idx_channel_timestamp ON messages(channel_id, timestamp);
	`
	_, err := db.Exec(query)
	return err
}

// Register adds a new client connection
func (r *MessageRouter) Register(userID string, conn *websocket.Conn) *Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
	r.clients[userID] = append(r.clients[userID], client)

	log.Printf("Client registered: %s (Total conns: %d)", userID, len(r.clients[userID]))
	return client
}

// Unregister removes a client connection
func (r *MessageRouter) Unregister(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	userID := client.UserID
	conns := r.clients[userID]
	for i, c := range conns {
		if c == client {
			r.clients[userID] = append(conns[:i], conns[i+1:]...)
			close(client.Send)
			break
		}
	}
	if len(r.clients[userID]) == 0 {
		delete(r.clients, userID)
	}
	log.Printf("Client unregistered: %s", userID)
}

// Broadcast sends a message to specific users (who should receive this message)
// For now, it broadcasts to everyone globally for demo, or we can look up channel members.
// Realistically, it should send to members of req.ChannelID.
func (r *MessageRouter) Broadcast(msg *protocol.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, _ := json.Marshal(msg)

	// In a real Slack-like app, we'd only send to users in the same channel.
	// For this LAN implementation, we broadcast to all online users.
	// The client-side will filter by ChannelID.
	for _, conns := range r.clients {
		for _, client := range conns {
			select {
			case client.Send <- data:
			default:
				// If buffer is full, we skip or handle accordingly
			}
		}
	}
}

// HandleWS handles WebSocket upgrade and loop
func (r *MessageRouter) HandleWS(w http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	userID := req.URL.Query().Get("user_id")
	if userID == "" {
		userID = "anonymous-" + uuid.New().String()[:8]
	}

	client := r.Register(userID, conn)

	// Read loop (client sending messages via WS)
	go func() {
		defer func() {
			r.Unregister(client)
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var sendReq protocol.SendMessageRequest
			if err := json.Unmarshal(message, &sendReq); err == nil {
				msg, err := r.SaveMessage(sendReq, userID)
				if err == nil {
					r.Broadcast(msg)
				}
			}
		}
	}()

	// Write loop (sending messages to client)
	for data := range client.Send {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			break
		}
	}
}

func (r *MessageRouter) SaveMessage(req protocol.SendMessageRequest, senderID string) (*protocol.Message, error) {
	msg := &protocol.Message{
		ID:        uuid.New().String(),
		ChannelID: req.ChannelID,
		SenderID:  senderID,
		Timestamp: time.Now().UnixMilli(),
		Type:      req.Type,
		Content:   req.Content,
		Nonce:     req.Nonce,
		Signature: req.Signature,
	}

	_, err := r.db.Exec(`
		INSERT INTO messages (id, channel_id, sender_id, timestamp, type, content, nonce, signature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ChannelID, msg.SenderID, msg.Timestamp, msg.Type, msg.Content, msg.Nonce, msg.Signature,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to persist message: %w", err)
	}

	return msg, nil
}

func (r *MessageRouter) HistoryHandler(w http.ResponseWriter, req *http.Request) {
	channelID := req.URL.Query().Get("channel_id")
	if channelID == "" {
		http.Error(w, "missing channel_id", http.StatusBadRequest)
		return
	}

	rows, err := r.db.Query(`
		SELECT id, channel_id, sender_id, timestamp, type, content, nonce, signature 
		FROM messages WHERE channel_id = ? ORDER BY timestamp ASC LIMIT 100`, channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []protocol.Message
	for rows.Next() {
		var m protocol.Message
		err := rows.Scan(&m.ID, &m.ChannelID, &m.SenderID, &m.Timestamp, &m.Type, &m.Content, &m.Nonce, &m.Signature)
		if err == nil {
			history = append(history, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func main() {
	dbPath := os.Getenv("MESSAGING_DB_PATH")
	if dbPath == "" {
		dbPath = "data/chat.db"
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	router, err := NewMessageRouter(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize router: %v", err)
	}

	http.HandleFunc("/ws", router.HandleWS)
	http.HandleFunc("/history", router.HistoryHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Messaging Service is running")
	})
	http.HandleFunc("/send", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var msgReq protocol.SendMessageRequest
		if err := json.NewDecoder(req.Body).Decode(&msgReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		senderID := req.Header.Get("X-User-ID")
		if senderID == "" {
			senderID = "anonymous"
		}

		msg, err := router.SaveMessage(msgReq, senderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		router.Broadcast(msg)
		json.NewEncoder(w).Encode(protocol.SendMessageResponse{MessageID: msg.ID, Success: true})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Messaging Service (WS/HTTP) started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
