package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// RaftNode represents a node in the Raft cluster.
type RaftNode struct {
	NodeID   string
	RaftDir  string
	RaftBind string
	State    string // Leader, Follower, Candidate
}

const requestIDHeader = "X-Request-ID"

var clusterLimiter = newIPRateLimiter(120, time.Minute)

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
	return "cluster-fallback-id"
}

func withRequestTrace(name string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := getOrCreateRequestID(r)
		w.Header().Set(requestIDHeader, rid)
		r.Header.Set(requestIDHeader, rid)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		if !clusterLimiter.Allow(clientIP(r)) {
			http.Error(rec, "Too many requests", http.StatusTooManyRequests)
		} else {
			next(rec, r)
		}
		entry := map[string]interface{}{
			"ts":         time.Now().UTC().Format(time.RFC3339Nano),
			"service":    "cluster",
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

func NewRaftNode(nodeID, raftDir, raftBind string) *RaftNode {
	return &RaftNode{
		NodeID:   nodeID,
		RaftDir:  raftDir,
		RaftBind: raftBind,
		State:    "Follower",
	}
}

// Start initializes the Raft node (Stub).
func (n *RaftNode) Start() error {
	log.Printf("Starting Raft Node %s on %s...", n.NodeID, n.RaftBind)

	// Simulate Raft Bootstrap
	go func() {
		time.Sleep(5 * time.Second)
		n.State = "Leader"
		log.Printf("Node %s is now the Leader", n.NodeID)
		n.startHeartbeat()
	}()

	return nil
}

func (n *RaftNode) startHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		// Log heartbeat to simulate leader activity
		// log.Printf("Leader Heartbeat...")
	}
}

// JoinHandler handles join requests from other nodes.
func (n *RaftNode) JoinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nodeID := r.URL.Query().Get("node_id")
	addr := r.URL.Query().Get("addr")
	nodeID = strings.TrimSpace(nodeID)
	addr = strings.TrimSpace(addr)
	if nodeID == "" || len(nodeID) > 64 {
		http.Error(w, "Invalid node_id", http.StatusBadRequest)
		return
	}
	if addr == "" || len(addr) > 128 {
		http.Error(w, "Invalid addr", http.StatusBadRequest)
		return
	}
	log.Printf("Received join request from %s at %s", nodeID, addr)
	w.WriteHeader(http.StatusOK)
}

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "node-1"
	}

	raftNode := NewRaftNode(nodeID, "./raft-data", ":10001")
	if err := raftNode.Start(); err != nil {
		log.Fatalf("Failed to start Raft node: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/join", withRequestTrace("join", raftNode.JoinHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Cluster Management Service started on :%s", port)
	log.Fatal(server.ListenAndServe())
}
