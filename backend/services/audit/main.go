package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
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

	http.HandleFunc("/log", svc.LogHandler)

	log.Println("Audit Service started on :8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}
