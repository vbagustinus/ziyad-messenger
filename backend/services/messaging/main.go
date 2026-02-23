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
	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT DEFAULT 'public'
	);
	CREATE TABLE IF NOT EXISTS channel_members (
		channel_id TEXT,
		user_id TEXT,
		PRIMARY KEY (channel_id, user_id)
	);
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

// getChannelMembers returns a list of user IDs who are members of the channel.
// If the channel is 'public', it returns an empty list (meaning broadcast to all).
func (r *MessageRouter) getChannelMembers(channelID string) ([]string, string, error) {
	var chType string
	err := r.db.QueryRow("SELECT type FROM channels WHERE id = ?", channelID).Scan(&chType)
	if err != nil {
		// Fallback for demo or if channel not in main DB (messaging might have its own table if synced)
		// Assuming for now it's in the same DB or we have access to it.
		return nil, "public", nil
	}

	if chType == "public" {
		return nil, "public", nil
	}

	rows, err := r.db.Query("SELECT user_id FROM channel_members WHERE channel_id = ?", channelID)
	if err != nil {
		return nil, chType, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			members = append(members, uid)
		}
	}
	return members, chType, nil
}

// Broadcast sends a message to specific users (who should receive this message)
func (r *MessageRouter) Broadcast(msg *protocol.Message) {
	members, chType, err := r.getChannelMembers(msg.ChannelID)
	if err != nil {
		log.Printf("Error getting members: %v", err)
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	data, _ := json.Marshal(msg)

	if chType == "public" {
		// Broadcast to everyone online
		for _, conns := range r.clients {
			for _, client := range conns {
				select {
				case client.Send <- data:
				default:
				}
			}
		}
	} else {
		// Only send to members
		for _, userID := range members {
			if conns, ok := r.clients[userID]; ok {
				for _, client := range conns {
					select {
					case client.Send <- data:
					default:
					}
				}
			}
		}
	}
}

// findOrCreateDMChannel ensures a DM channel exists between two users.
func (r *MessageRouter) findOrCreateDMChannel(u1, u2 string) (string, error) {
	// Standardized name for DM: "dm:<uid1>:<uid2>" where uid1 < uid2
	p1, p2 := u1, u2
	if p1 > p2 {
		p1, p2 = u2, u1
	}
	dmID := fmt.Sprintf("dm:%s:%s", p1, p2)

	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM channels WHERE id = ?)", dmID).Scan(&exists)
	if err != nil {
		return "", err
	}

	if !exists {
		tx, err := r.db.Begin()
		if err != nil {
			return "", err
		}
		defer tx.Rollback()

		_, err = tx.Exec("INSERT INTO channels (id, name, type) VALUES (?, ?, ?)", dmID, "Direct Message", "dm")
		if err != nil {
			return "", err
		}
		_, err = tx.Exec("INSERT INTO channel_members (channel_id, user_id) VALUES (?, ?)", dmID, u1)
		if err != nil {
			return "", err
		}
		_, err = tx.Exec("INSERT INTO channel_members (channel_id, user_id) VALUES (?, ?)", dmID, u2)
		if err != nil {
			return "", err
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
	}
	return dmID, nil
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
				// DM detection: if channel_id matches a user pattern or we decide by prefix
				// For LAN Slack, if a client sends a message to another UserID instead of a ChannelID,
				// we treat it as a DM request.
				finalChannelID := sendReq.ChannelID
				if len(sendReq.ChannelID) > 0 && sendReq.ChannelID[0] != 'c' && sendReq.ChannelID != "general" {
					// Heuristic: if it's not a known channel prefix, try DM
					// In a real app, we'd check if ChannelID exists as a user if not as a channel.
					if dmID, err := r.findOrCreateDMChannel(userID, sendReq.ChannelID); err == nil {
						finalChannelID = dmID
					}
				}

				msg, err := r.SaveMessage(sendReq, userID, finalChannelID)
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

func (r *MessageRouter) SaveMessage(req protocol.SendMessageRequest, senderID string, channelID string) (*protocol.Message, error) {
	msg := &protocol.Message{
		ID:        uuid.New().String(),
		ChannelID: channelID,
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

		msg, err := router.SaveMessage(msgReq, senderID, msgReq.ChannelID)
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
