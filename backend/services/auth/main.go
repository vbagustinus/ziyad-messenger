package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// User represents a user account.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

// AuthService handles authentication and user management.
type AuthService struct {
	db *sql.DB
	mu sync.RWMutex
}

const requestIDHeader = "X-Request-ID"
const defaultAuthBodyLimit = 1 << 20 // 1 MiB

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,32}$`)
var authLimiter = newIPRateLimiter(60, time.Minute)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

type ipRateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	list := l.hits[ip]
	i := 0
	for i < len(list) && list[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		list = list[i:]
	}
	if len(list) >= l.limit {
		l.hits[ip] = list
		return false
	}
	list = append(list, now)
	l.hits[ip] = list
	return true
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
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

func withRequestTrace(name string, maxBodyBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := getOrCreateRequestID(r)
		w.Header().Set(requestIDHeader, rid)
		r.Header.Set(requestIDHeader, rid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		if !authLimiter.Allow(clientIP(r)) {
			http.Error(rec, "Too many requests", http.StatusTooManyRequests)
		} else {
			if maxBodyBytes > 0 {
				r.Body = http.MaxBytesReader(rec, r.Body, maxBodyBytes)
			}
			next(rec, r)
		}

		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "auth",
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

func NewAuthService(dbPath string) (*AuthService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initDB(db); err != nil {
		return nil, err
	}

	return &AuthService{
		db: db,
	}, nil
}

func initDB(db *sql.DB) error {
	// Let admin-service handle the full schema, but ensure users table exists for auth
	query := `
	CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		permissions TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE,
		full_name TEXT,
		password_hash TEXT,
		role_id TEXT,
		department_id TEXT,
		created_at INTEGER,
		updated_at INTEGER,
		FOREIGN KEY (role_id) REFERENCES roles(id)
	);
	`
	_, err := db.Exec(query)
	return err
}

// RegisterRequest is the payload for creating a user.
type RegisterRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// LoginRequest is the payload for auth.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new user.
func (s *AuthService) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Role = strings.TrimSpace(req.Role)
	if !usernamePattern.MatchString(req.Username) {
		http.Error(w, "Invalid username format", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		http.Error(w, "Password length must be 8-72", http.StatusBadRequest)
		return
	}
	if len(req.FullName) > 80 {
		http.Error(w, "Full name too long", http.StatusBadRequest)
		return
	}
	if len(req.Role) > 40 {
		http.Error(w, "Role too long", http.StatusBadRequest)
		return
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	userID := uuid.New().String()
	now := time.Now().Unix()
	_, err = s.db.Exec(`INSERT INTO users (id, username, full_name, password_hash, role_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, req.Username, req.FullName, hash, req.Role, now, now)

	if err != nil {
		http.Error(w, "User already exists or DB error", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": userID})
}

// Login authenticates a user.
func (s *AuthService) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Missing credentials", http.StatusBadRequest)
		return
	}
	if len(req.Username) > 64 || len(req.Password) > 128 {
		http.Error(w, "Invalid credentials format", http.StatusBadRequest)
		return
	}

	var user User
	// Join with roles to get role name from role_id, fallback to role_id itself if roles table is empty
	err := s.db.QueryRow(`
		SELECT u.id, u.username, u.password_hash, COALESCE(r.name, u.role_id, 'user') as role 
		FROM users u 
		LEFT JOIN roles r ON u.role_id = r.id 
		WHERE u.username = ?`, req.Username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	match, err := VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.Username, user.Role)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token":   token,
		"user_id": user.ID,
		"role":    user.Role,
	})
}

func (s *AuthService) ensureDefaultUser(username, password, role string) error {
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	userID := uuid.New().String()
	now := time.Now().Unix()
	_, err = s.db.Exec(`INSERT INTO users (id, username, full_name, password_hash, role_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, username, "Administrator", hash, role, now, now)
	if err != nil {
		return err
	}

	log.Printf("Default user %s created", username)
	return nil
}

func main() {
	dbPath := os.Getenv("AUTH_DB_PATH")
	if dbPath == "" {
		dbPath = "data/users.db"
	}

	// Ensure directory for dbPath exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	svc, err := NewAuthService(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}

	if err := svc.ensureDefaultUser("admin", "password", "admin"); err != nil {
		log.Printf("Failed to ensure default user: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/register", withRequestTrace("register", defaultAuthBodyLimit, svc.RegisterHandler))
	mux.HandleFunc("/login", withRequestTrace("login", defaultAuthBodyLimit, svc.LoginHandler))
	mux.HandleFunc("/health", withRequestTrace("health", 0, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Auth Service is running")
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Auth Service started on :%s", port)
	log.Fatal(server.ListenAndServe())
}
