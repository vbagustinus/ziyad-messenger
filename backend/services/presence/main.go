package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type UserStatus int

const (
	StatusOffline UserStatus = iota
	StatusOnline
	StatusBusy
	StatusAway
)

type ValidationRequest struct {
	UserID string     `json:"user_id"`
	Status UserStatus `json:"status"`
}

type PresenceService struct {
	db *sql.DB
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
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func withRequestTrace(name string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := getOrCreateRequestID(r)
		w.Header().Set(requestIDHeader, rid)
		r.Header.Set(requestIDHeader, rid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)

		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "presence",
			"handler":    name,
			"request_id": rid,
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     rec.status,
			"latency_ms": float64(time.Since(start).Microseconds()) / 1000.0,
		}
		if b, err := json.Marshal(entry); err == nil {
			log.Println(string(b))
		}
	}
}

func NewPresenceService(dbPath string) (*PresenceService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	svc := &PresenceService{db: db}
	if err := svc.initDB(); err != nil {
		return nil, err
	}
	return svc, nil
}

func (s *PresenceService) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS user_presence (
		user_id TEXT PRIMARY KEY,
		status INTEGER NOT NULL,
		last_seen INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_user_presence_status ON user_presence(status);
	CREATE INDEX IF NOT EXISTS idx_user_presence_last_seen ON user_presence(last_seen);
	`
	_, err := s.db.Exec(query)
	return err
}

// HeartbeatHandler updates a user's status and last seen time.
func (s *PresenceService) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if req.UserID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO user_presence (user_id, status, last_seen, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			status = excluded.status,
			last_seen = excluded.last_seen,
			updated_at = excluded.updated_at`,
		req.UserID, req.Status, now, now,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// StatusHandler returns the status of a user.
func (s *PresenceService) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	var status int
	var lastSeenUnix int64
	err := s.db.QueryRow(
		`SELECT status, last_seen FROM user_presence WHERE user_id = ?`,
		userID,
	).Scan(&status, &lastSeenUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			status = int(StatusOffline)
			lastSeenUnix = 0
		} else {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
	}

	// Auto-offline logic if heartbeat missing for > 1 minute
	lastSeen := time.Unix(lastSeenUnix, 0)
	if time.Since(lastSeen) > 1*time.Minute {
		status = int(StatusOffline)
	}

	resp := map[string]interface{}{
		"user_id":   userID,
		"status":    status,
		"last_seen": lastSeen,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *PresenceService) CleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		cutoff := time.Now().Add(-1 * time.Minute).Unix()
		_, _ = s.db.Exec(
			`UPDATE user_presence SET status = ? WHERE last_seen < ?`,
			int(StatusOffline), cutoff,
		)
	}
}

func main() {
	dbPath := os.Getenv("PRESENCE_DB_PATH")
	if dbPath == "" {
		dbPath = "data/presence.db"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v", err)
	}

	svc, err := NewPresenceService(dbPath)
	if err != nil {
		log.Fatalf("Failed to init presence service: %v", err)
	}

	go svc.CleanupLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/heartbeat", withRequestTrace("heartbeat", svc.HeartbeatHandler))
	mux.HandleFunc("/status", withRequestTrace("status", svc.StatusHandler))
	mux.HandleFunc("/health", withRequestTrace("health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Presence Service is running")
	}))

	log.Println("Presence Service started on :8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}
