package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

	http.HandleFunc("/register", svc.RegisterHandler)
	http.HandleFunc("/login", svc.LoginHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Auth Service is running")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	log.Printf("Auth Service started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
