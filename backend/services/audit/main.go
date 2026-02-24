package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AuditEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	ActorID        string    `json:"actor_id"`
	Action         string    `json:"action"`
	TargetResource string    `json:"target_resource"`
	Details        string    `json:"details"`
	PrevHash       string    `json:"prev_hash"`
}

type AuditService struct {
	logFile  *os.File
	lastHash string
	mu       sync.Mutex
}

const requestIDHeader = "X-Request-ID"
const defaultAuditBodyLimit = 64 << 10 // 64 KiB

var auditLimiter = newIPRateLimiter(120, time.Minute)
var actionPattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{2,64}$`)

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

func getOrCreateRequestID(r *http.Request) string {
	if rid := strings.TrimSpace(r.Header.Get(requestIDHeader)); rid != "" {
		return rid
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "audit-fallback-id"
}

func withRequestTrace(name string, maxBodyBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := getOrCreateRequestID(r)
		w.Header().Set(requestIDHeader, rid)
		r.Header.Set(requestIDHeader, rid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		if !auditLimiter.Allow(clientIP(r)) {
			http.Error(rec, "Too many requests", http.StatusTooManyRequests)
		} else {
			if maxBodyBytes > 0 {
				r.Body = http.MaxBytesReader(rec, r.Body, maxBodyBytes)
			}
			next(rec, r)
		}

		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "audit",
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

func NewAuditService(logPath string) *AuditService {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open audit log: %v", err)
	}

	// In a real implementation, we would read the last line to get the last hash
	// For simplicity, we start with a genesis hash
	return &AuditService{
		logFile:  f,
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
}

func (s *AuditService) LogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event AuditEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	event.ActorID = strings.TrimSpace(event.ActorID)
	event.Action = strings.TrimSpace(event.Action)
	event.TargetResource = strings.TrimSpace(event.TargetResource)
	if event.ActorID == "" || len(event.ActorID) > 64 {
		http.Error(w, "Invalid actor_id", http.StatusBadRequest)
		return
	}
	if !actionPattern.MatchString(event.Action) {
		http.Error(w, "Invalid action format", http.StatusBadRequest)
		return
	}
	if event.TargetResource == "" || len(event.TargetResource) > 128 {
		http.Error(w, "Invalid target_resource", http.StatusBadRequest)
		return
	}
	if len(event.Details) > 4096 {
		http.Error(w, "Details too long", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	event.Timestamp = time.Now()
	event.PrevHash = s.lastHash

	// Serialize
	data, err := json.Marshal(event)
	if err != nil {
		http.Error(w, "Marshaling error", http.StatusInternalServerError)
		return
	}

	// Calculate new hash
	hash := sha256.Sum256(data)
	s.lastHash = hex.EncodeToString(hash[:])

	// Write to file (Append only)
	if _, err := s.logFile.Write(append(data, '\n')); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func main() {
	logPath := "data/audit.log"
	// Ensure directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	svc := NewAuditService(logPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/log", withRequestTrace("log", defaultAuditBodyLimit, svc.LogHandler))

	server := &http.Server{
		Addr:              ":8084",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Println("Audit Service started on :8084")
	log.Fatal(server.ListenAndServe())
}
