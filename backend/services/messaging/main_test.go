package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func seedMessagingTestData(t *testing.T, r *MessageRouter) {
	t.Helper()

	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL
		);`)
	if err != nil {
		t.Fatalf("create users table: %v", err)
	}

	_, _ = r.db.Exec(`DELETE FROM users`)
	_, _ = r.db.Exec(`DELETE FROM channels`)
	_, _ = r.db.Exec(`DELETE FROM channel_members`)
	_, _ = r.db.Exec(`DELETE FROM messages`)

	_, err = r.db.Exec(`INSERT INTO users (id, username) VALUES
		('u-alice', 'alice'),
		('u-bob', 'bob'),
		('u-charlie', 'charlie')`)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	_, err = r.db.Exec(`INSERT INTO channels (id, name, type) VALUES
		('general', 'General', 'public'),
		('priv-1', 'Private One', 'private')`)
	if err != nil {
		t.Fatalf("insert channels: %v", err)
	}

	_, err = r.db.Exec(`INSERT INTO channel_members (channel_id, user_id) VALUES
		('priv-1', 'u-alice'),
		('priv-1', 'u-bob')`)
	if err != nil {
		t.Fatalf("insert members: %v", err)
	}

	_, err = r.db.Exec(`INSERT INTO messages (id, channel_id, sender_id, timestamp, type, content, nonce, signature)
		VALUES ('m-1', 'priv-1', 'u-alice', ?, 1, ?, '', '')`, time.Now().UnixMilli(), []byte("aGVsbG8="))
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}
}

func newMessagingTestRouter(t *testing.T) *MessageRouter {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "messaging.db")
	r, err := NewMessageRouter(dbPath)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	seedMessagingTestData(t, r)
	return r
}

func tokenForTestUser(t *testing.T, username string) string {
	t.Helper()
	claims := &Claims{
		Username: username,
		Role:     "member",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	out, err := tok.SignedString(jwtSecret())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return out
}

func TestHistoryAuthorization(t *testing.T) {
	r := newMessagingTestRouter(t)

	unauthReq := httptest.NewRequest(http.MethodGet, "/history?channel_id=priv-1", nil)
	unauthRec := httptest.NewRecorder()
	r.HistoryHandler(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthRec.Code)
	}

	charlieReq := httptest.NewRequest(http.MethodGet, "/history?channel_id=priv-1", nil)
	charlieReq.Header.Set("Authorization", "Bearer "+tokenForTestUser(t, "charlie"))
	charlieRec := httptest.NewRecorder()
	r.HistoryHandler(charlieRec, charlieReq)
	if charlieRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", charlieRec.Code)
	}

	aliceReq := httptest.NewRequest(http.MethodGet, "/history?channel_id=priv-1", nil)
	aliceReq.Header.Set("Authorization", "Bearer "+tokenForTestUser(t, "alice"))
	aliceRec := httptest.NewRecorder()
	r.HistoryHandler(aliceRec, aliceReq)
	if aliceRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", aliceRec.Code, aliceRec.Body.String())
	}
	var messages []map[string]any
	if err := json.Unmarshal(aliceRec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(messages) == 0 {
		t.Fatalf("expected non-empty history")
	}
}

func TestChannelsAndDMFlow(t *testing.T) {
	r := newMessagingTestRouter(t)

	chReq := httptest.NewRequest(http.MethodGet, "/channels", nil)
	chReq.Header.Set("Authorization", "Bearer "+tokenForTestUser(t, "bob"))
	chRec := httptest.NewRecorder()
	r.ChannelsHandler(chRec, chReq)
	if chRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", chRec.Code)
	}
	var chPayload struct {
		Channels []struct {
			ID string `json:"id"`
		} `json:"channels"`
	}
	if err := json.Unmarshal(chRec.Body.Bytes(), &chPayload); err != nil {
		t.Fatalf("decode channels: %v", err)
	}
	ids := make([]string, 0, len(chPayload.Channels))
	for _, c := range chPayload.Channels {
		ids = append(ids, c.ID)
	}
	joined := strings.Join(ids, ",")
	if !strings.Contains(joined, "general") || !strings.Contains(joined, "priv-1") {
		t.Fatalf("expected general and priv-1 in channels, got %v", ids)
	}

	body := []byte(`{"target_user_id":"u-charlie"}`)
	dmReq := httptest.NewRequest(http.MethodPost, "/dm", bytes.NewReader(body))
	dmReq.Header.Set("Authorization", "Bearer "+tokenForTestUser(t, "bob"))
	dmReq.Header.Set("Content-Type", "application/json")
	dmRec := httptest.NewRecorder()
	r.CreateDMHandler(dmRec, dmReq)
	if dmRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", dmRec.Code, dmRec.Body.String())
	}
	var dmResp map[string]string
	if err := json.Unmarshal(dmRec.Body.Bytes(), &dmResp); err != nil {
		t.Fatalf("decode dm response: %v", err)
	}
	dmID := dmResp["channel_id"]
	if dmID == "" || !strings.HasPrefix(dmID, "dm:") {
		t.Fatalf("expected dm channel id, got %q", dmID)
	}

	member, err := r.isChannelMember(dmID, "u-bob")
	if err != nil || !member {
		t.Fatalf("expected bob as member in dm channel, err=%v member=%v", err, member)
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
		t.Fatalf("expected %s response header", requestIDHeader)
	}
}

func TestWithRequestTracePropagatesIncomingRequestID(t *testing.T) {
	h := withRequestTrace("test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(requestIDHeader, "rid-123")
	rec := httptest.NewRecorder()
	h(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != "rid-123" {
		t.Fatalf("expected propagated request id rid-123, got %q", got)
	}
}
