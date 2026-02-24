package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lan-chat/protocol"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const requestIDHeader = "X-Request-ID"

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func getOrCreateRequestID(req *http.Request) string {
	if rid := strings.TrimSpace(req.Header.Get(requestIDHeader)); rid != "" {
		return rid
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return uuid.NewString()
}

func withRequestTrace(name string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rid := getOrCreateRequestID(req)
		w.Header().Set(requestIDHeader, rid)
		req.Header.Set(requestIDHeader, rid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, req)

		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "messaging",
			"handler":    name,
			"request_id": rid,
			"method":     req.Method,
			"path":       req.URL.Path,
			"status":     rec.status,
			"latency_ms": float64(time.Since(start).Microseconds()) / 1000.0,
		}
		if b, err := json.Marshal(entry); err == nil {
			log.Println(string(b))
		}
	}
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

var (
	errUnauthorized   = errors.New("unauthorized")
	errChannelMissing = errors.New("channel not found")
	errForbidden      = errors.New("forbidden")
)

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type ChannelView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type ChannelMemberView struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

type CreateDMRequest struct {
	TargetUserID string `json:"target_user_id"`
}

func jwtSecret() []byte {
	secret := os.Getenv("MESSAGING_JWT_SECRET")
	if secret == "" {
		secret = "my_secret_key"
	}
	return []byte(secret)
}

func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errUnauthorized
	}
	return claims, nil
}

func bearerToken(req *http.Request) string {
	raw := req.Header.Get("Authorization")
	if raw != "" {
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	return req.URL.Query().Get("token")
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
	chType, err := r.getChannelType(channelID)
	if err != nil {
		return nil, "", err
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

func (r *MessageRouter) getChannelType(channelID string) (string, error) {
	var chType string
	err := r.db.QueryRow("SELECT type FROM channels WHERE id = ?", channelID).Scan(&chType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errChannelMissing
		}
		return "", err
	}
	return chType, nil
}

func (r *MessageRouter) userExists(userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *MessageRouter) findUserIDByUsername(username string) (string, error) {
	var userID string
	err := r.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (r *MessageRouter) isChannelMember(channelID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM channel_members WHERE channel_id = ? AND user_id = ?)",
		channelID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *MessageRouter) authorizeChannelAccess(userID, channelID string) error {
	chType, err := r.getChannelType(channelID)
	if err != nil {
		return err
	}
	if chType == "public" {
		return nil
	}
	member, err := r.isChannelMember(channelID, userID)
	if err != nil {
		return err
	}
	if !member {
		return errForbidden
	}
	return nil
}

func (r *MessageRouter) listAccessibleChannels(userID string) ([]ChannelView, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT c.id, c.name, c.type
		FROM channels c
		LEFT JOIN channel_members m ON c.id = m.channel_id
		WHERE c.type = 'public' OR m.user_id = ?
		ORDER BY c.name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ChannelView, 0)
	for rows.Next() {
		var ch ChannelView
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type); err == nil {
			out = append(out, ch)
		}
	}
	return out, nil
}

func (r *MessageRouter) listChannelMembers(channelID string) ([]ChannelMemberView, error) {
	rows, err := r.db.Query(`
		SELECT u.id, u.username, COALESCE(u.full_name, '')
		FROM channel_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.channel_id = ?
		ORDER BY u.username ASC`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ChannelMemberView, 0)
	for rows.Next() {
		var m ChannelMemberView
		if err := rows.Scan(&m.ID, &m.Username, &m.FullName); err == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (r *MessageRouter) resolveRequestedChannel(senderID, requestedChannelID string) (string, error) {
	if requestedChannelID == "" {
		return "", errChannelMissing
	}

	if _, err := r.getChannelType(requestedChannelID); err == nil {
		return requestedChannelID, nil
	} else if !errors.Is(err, errChannelMissing) {
		return "", err
	}

	exists, err := r.userExists(requestedChannelID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errChannelMissing
	}
	if requestedChannelID == senderID {
		return "", errForbidden
	}
	return r.findOrCreateDMChannel(senderID, requestedChannelID)
}

func (r *MessageRouter) authenticate(req *http.Request) (string, error) {
	token := bearerToken(req)
	if token == "" {
		return "", errUnauthorized
	}
	claims, err := validateToken(token)
	if err != nil {
		return "", errUnauthorized
	}
	userID, err := r.findUserIDByUsername(claims.Username)
	if err != nil {
		return "", errUnauthorized
	}
	return userID, nil
}

// Broadcast sends a message to specific users (who should receive this message)
func (r *MessageRouter) Broadcast(msg *protocol.Message) error {
	members, chType, err := r.getChannelMembers(msg.ChannelID)
	if err != nil {
		return err
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
	return nil
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
	userID, err := r.authenticate(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
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
				finalChannelID, err := r.resolveRequestedChannel(userID, sendReq.ChannelID)
				if err != nil {
					continue
				}
				if err := r.authorizeChannelAccess(userID, finalChannelID); err != nil {
					continue
				}

				msg, err := r.SaveMessage(sendReq, userID, finalChannelID)
				if err == nil {
					_ = r.Broadcast(msg)
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
	userID, err := r.authenticate(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	requestedChannelID := req.URL.Query().Get("channel_id")
	if requestedChannelID == "" {
		http.Error(w, "missing channel_id", http.StatusBadRequest)
		return
	}

	channelID, err := r.resolveRequestedChannel(userID, requestedChannelID)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}
	if err := r.authorizeChannelAccess(userID, channelID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
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

func (r *MessageRouter) ChannelsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, err := r.authenticate(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	channels, err := r.listAccessibleChannels(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"channels": channels})
}

func (r *MessageRouter) ChannelMembersHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, err := r.authenticate(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID := req.URL.Query().Get("channel_id")
	if channelID == "" {
		http.Error(w, "missing channel_id", http.StatusBadRequest)
		return
	}
	if err := r.authorizeChannelAccess(userID, channelID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	members, err := r.listChannelMembers(channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"members": members})
}

func (r *MessageRouter) CreateDMHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, err := r.authenticate(req)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body CreateDMRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if body.TargetUserID == "" {
		http.Error(w, "missing target_user_id", http.StatusBadRequest)
		return
	}
	if body.TargetUserID == userID {
		http.Error(w, "invalid target", http.StatusBadRequest)
		return
	}
	exists, err := r.userExists(body.TargetUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "target user not found", http.StatusNotFound)
		return
	}

	channelID, err := r.findOrCreateDMChannel(userID, body.TargetUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"channel_id": channelID})
}

func main() {
	dbPath := os.Getenv("MESSAGING_DB_PATH")
	if dbPath == "" {
		dbPath = "data/chat.db"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	router, err := NewMessageRouter(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize router: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", withRequestTrace("ws", router.HandleWS))
	mux.HandleFunc("/history", withRequestTrace("history", router.HistoryHandler))
	mux.HandleFunc("/channels", withRequestTrace("channels", router.ChannelsHandler))
	mux.HandleFunc("/channel-members", withRequestTrace("channel-members", router.ChannelMembersHandler))
	mux.HandleFunc("/dm", withRequestTrace("dm", router.CreateDMHandler))
	mux.HandleFunc("/health", withRequestTrace("health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Messaging Service is running")
	}))
	mux.HandleFunc("/send", withRequestTrace("send", func(w http.ResponseWriter, req *http.Request) {
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
		authUserID, err := router.authenticate(req)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		senderID = authUserID

		channelID, err := router.resolveRequestedChannel(senderID, msgReq.ChannelID)
		if err != nil {
			http.Error(w, "channel not found", http.StatusNotFound)
			return
		}
		if err := router.authorizeChannelAccess(senderID, channelID); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		msg, err := router.SaveMessage(msgReq, senderID, channelID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := router.Broadcast(msg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(protocol.SendMessageResponse{MessageID: msg.ID, Success: true})
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Messaging Service (WS/HTTP) started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
