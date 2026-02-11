package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
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
	statuses map[string]UserStatus
	lastSeen map[string]time.Time
	mu       sync.RWMutex
}

func NewPresenceService() *PresenceService {
	return &PresenceService{
		statuses: make(map[string]UserStatus),
		lastSeen: make(map[string]time.Time),
	}
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

	s.mu.Lock()
	s.statuses[req.UserID] = req.Status
	s.lastSeen[req.UserID] = time.Now()
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// StatusHandler returns the status of a user.
func (s *PresenceService) StatusHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	status, exists := s.statuses[userID]
	lastSeen := s.lastSeen[userID]
	s.mu.RUnlock()

	if !exists {
		status = StatusOffline
	}

	// Auto-offline logic if heartbeat missing for > 1 minute
	if time.Since(lastSeen) > 1*time.Minute {
		status = StatusOffline
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
		s.mu.Lock()
		for user, last := range s.lastSeen {
			if time.Since(last) > 1*time.Minute {
				s.statuses[user] = StatusOffline
			}
		}
		s.mu.Unlock()
	}
}

func main() {
	svc := NewPresenceService()

	go svc.CleanupLoop()

	http.HandleFunc("/heartbeat", svc.HeartbeatHandler)
	http.HandleFunc("/status", svc.StatusHandler)

	log.Println("Presence Service started on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
