package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const StorageDir = "./storage"
const requestIDHeader = "X-Request-ID"
const defaultUploadLimit = 20 << 20 // 20 MiB

var fileLimiter = newIPRateLimiter(45, time.Minute)

type FileTransferService struct {
	storagePath string
}

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
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)
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

func getOrCreateRequestID(r *http.Request) string {
	if rid := strings.TrimSpace(r.Header.Get(requestIDHeader)); rid != "" {
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
		if !fileLimiter.Allow(clientIP(r)) {
			http.Error(rec, "Too many requests", http.StatusTooManyRequests)
		} else {
			if maxBodyBytes > 0 {
				r.Body = http.MaxBytesReader(rec, r.Body, maxBodyBytes)
			}
			next(rec, r)
		}
		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "filetransfer",
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

func NewFileTransferService(path string) *FileTransferService {
	if err := os.MkdirAll(path, 0700); err != nil {
		log.Fatalf("Failed to create storage dir: %v", err)
	}
	return &FileTransferService{storagePath: path}
}

// UploadHandler handles file uploads and encrypts them at rest.
func (s *FileTransferService) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "multipart/form-data") {
		http.Error(w, "Content-Type must be multipart/form-data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileID := uuid.New().String()
	if header.Filename == "" || len(header.Filename) > 255 {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	ext := filepath.Ext(header.Filename)
	if len(ext) > 16 {
		http.Error(w, "Invalid file extension", http.StatusBadRequest)
		return
	}
	savePath := filepath.Join(s.storagePath, fileID+ext+".enc")

	// Create file
	out, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Generate DEK (Data Encryption Key)
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	iv := make([]byte, 12) // GCM IV size
	if _, err := rand.Read(iv); err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		http.Error(w, "Crypto error", http.StatusInternalServerError)
		return
	}

	// Read content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Read error", http.StatusInternalServerError)
		return
	}

	// Encrypt
	ciphertext := aesGCM.Seal(nil, iv, content, nil)

	// Write IV + Ciphertext
	if _, err := out.Write(iv); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}
	if _, err := out.Write(ciphertext); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	// Return File ID and Key (Key should be protected/wrapped in real app)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"file_id": "%s", "key": "%x"}`, fileID, key)
}

// DownloadHandler serves encrypted files.
func (s *FileTransferService) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("id")
	if fileID == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(fileID); err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// In a real app, strict path sanitization is needed
	matches, _ := filepath.Glob(filepath.Join(s.storagePath, fileID+"*.enc"))
	if len(matches) == 0 {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, matches[0])
}

func main() {
	svc := NewFileTransferService(StorageDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", withRequestTrace("upload", defaultUploadLimit+1024*1024, svc.UploadHandler))
	mux.HandleFunc("/download", withRequestTrace("download", 0, svc.DownloadHandler))
	mux.HandleFunc("/health", withRequestTrace("health", 0, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "File Transfer Service is running")
	}))

	server := &http.Server{
		Addr:              ":8082",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Println("File Transfer Service started on :8082")
	log.Fatal(server.ListenAndServe())
}
