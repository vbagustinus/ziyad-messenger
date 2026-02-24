package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func newPresenceTestService(t *testing.T) *PresenceService {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "presence.db")
	svc, err := NewPresenceService(dbPath)
	if err != nil {
		t.Fatalf("new presence service: %v", err)
	}
	return svc
}

func TestHeartbeatAndStatus(t *testing.T) {
	svc := newPresenceTestService(t)

	body := []byte(`{"user_id":"u-1","status":1}`)
	hbReq := httptest.NewRequest(http.MethodPost, "/heartbeat", bytes.NewReader(body))
	hbRec := httptest.NewRecorder()
	svc.HeartbeatHandler(hbRec, hbReq)
	if hbRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", hbRec.Code)
	}

	stReq := httptest.NewRequest(http.MethodGet, "/status?user_id=u-1", nil)
	stRec := httptest.NewRecorder()
	svc.StatusHandler(stRec, stReq)
	if stRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", stRec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(stRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got := int(payload["status"].(float64)); got != int(StatusOnline) {
		t.Fatalf("expected status online(%d), got %d", StatusOnline, got)
	}
}

func TestStatusAutoOfflineByLastSeen(t *testing.T) {
	svc := newPresenceTestService(t)

	old := time.Now().Add(-2 * time.Minute).Unix()
	_, err := svc.db.Exec(`INSERT INTO user_presence (user_id, status, last_seen, updated_at) VALUES (?, ?, ?, ?)`,
		"u-old", int(StatusOnline), old, old)
	if err != nil {
		t.Fatalf("insert old presence: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/status?user_id=u-old", nil)
	rec := httptest.NewRecorder()
	svc.StatusHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got := int(payload["status"].(float64)); got != int(StatusOffline) {
		t.Fatalf("expected status offline(%d), got %d", StatusOffline, got)
	}
}

func TestWithRequestTraceSetsRequestID(t *testing.T) {
	h := withRequestTrace("test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	if got := rec.Header().Get(requestIDHeader); got == "" {
		t.Fatalf("expected %s header", requestIDHeader)
	}
}

func TestWithRequestTracePropagatesRequestID(t *testing.T) {
	h := withRequestTrace("test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(requestIDHeader, "rid-presence")
	rec := httptest.NewRecorder()
	h(rec, req)
	if got := rec.Header().Get(requestIDHeader); got != "rid-presence" {
		t.Fatalf("expected propagated request id, got %q", got)
	}
}
