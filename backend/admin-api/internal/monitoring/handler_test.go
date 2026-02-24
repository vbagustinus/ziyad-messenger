package monitoring

import (
	"admin-service/internal/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	resetOverviewCache()
	if db.DB != nil {
		_ = db.DB.Close()
	}
	dsn := filepath.Join(t.TempDir(), "monitoring_test.db")
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.DB = conn
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

func callOverview(t *testing.T) OverviewResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/overview", Overview)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/overview", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", w.Code, w.Body.String())
	}

	var resp OverviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

func TestOverviewFallbackWithoutTables(t *testing.T) {
	setupTestDB(t)

	resp := callOverview(t)
	if resp.Network.NodesOnline != 1 {
		t.Fatalf("expected fallback nodes_online=1, got %d", resp.Network.NodesOnline)
	}
	if resp.Users.TotalUsers != 0 {
		t.Fatalf("expected total_users=0, got %d", resp.Users.TotalUsers)
	}
	if resp.Messages.TotalMessages != 0 {
		t.Fatalf("expected total_messages=0, got %d", resp.Messages.TotalMessages)
	}
	if resp.Files.TotalFiles != 0 {
		t.Fatalf("expected total_files=0, got %d", resp.Files.TotalFiles)
	}
	if resp.GeneratedAt <= 0 {
		t.Fatalf("expected generated_at > 0, got %d", resp.GeneratedAt)
	}
}

func TestOverviewUsesDatabaseData(t *testing.T) {
	setupTestDB(t)

	now := time.Now()
	nowUnix := now.Unix()
	nowMs := now.UnixMilli()

	stmts := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY)`,
		`CREATE TABLE user_presence (user_id TEXT PRIMARY KEY, status INTEGER NOT NULL, last_seen INTEGER NOT NULL)`,
		`CREATE TABLE messages (sender_id TEXT NOT NULL, timestamp INTEGER NOT NULL, type INTEGER NOT NULL)`,
		`CREATE TABLE files (size_bytes INTEGER NOT NULL, created_at INTEGER NOT NULL)`,
		`CREATE TABLE cluster_nodes (id TEXT PRIMARY KEY)`,
		`CREATE TABLE audit_logs (timestamp INTEGER NOT NULL, action TEXT NOT NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.DB.Exec(s); err != nil {
			t.Fatalf("create table failed: %v", err)
		}
	}

	_, _ = db.DB.Exec(`INSERT INTO cluster_nodes (id) VALUES ('n1'), ('n2'), ('n3')`)
	_, _ = db.DB.Exec(`INSERT INTO users (id) VALUES ('u1'), ('u2'), ('u3')`)
	_, _ = db.DB.Exec(`INSERT INTO user_presence (user_id, status, last_seen) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		"u1", 1, nowUnix,
		"u2", 1, nowUnix-120,
		"u3", 0, nowUnix,
	)
	_, _ = db.DB.Exec(`INSERT INTO messages (sender_id, timestamp, type) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		"u1", nowMs, 1,
		"u2", nowMs-(2*60*60*1000), 1,
		"u3", nowMs-(26*60*60*1000), 1,
		"u1", nowMs-(10*60*1000), 3,
	)
	_, _ = db.DB.Exec(`INSERT INTO files (size_bytes, created_at) VALUES (?, ?), (?, ?)`,
		100, nowUnix,
		250, nowUnix-(2*24*60*60),
	)

	resp := callOverview(t)

	if resp.Network.NodesOnline != 3 {
		t.Fatalf("expected nodes_online=3, got %d", resp.Network.NodesOnline)
	}
	if resp.Users.TotalUsers != 3 {
		t.Fatalf("expected total_users=3, got %d", resp.Users.TotalUsers)
	}
	if resp.Users.OnlineNow != 1 {
		t.Fatalf("expected online_now=1, got %d", resp.Users.OnlineNow)
	}
	if resp.Users.ActiveToday != 2 {
		t.Fatalf("expected active_today=2, got %d", resp.Users.ActiveToday)
	}
	if resp.Messages.MessagesLastHour != 2 {
		t.Fatalf("expected messages_last_hour=2, got %d", resp.Messages.MessagesLastHour)
	}
	if resp.Messages.MessagesToday != 3 {
		t.Fatalf("expected messages_today=3, got %d", resp.Messages.MessagesToday)
	}
	if resp.Messages.TotalMessages != 4 {
		t.Fatalf("expected total_messages=4, got %d", resp.Messages.TotalMessages)
	}
	if resp.Files.TotalFiles != 2 {
		t.Fatalf("expected total_files=2, got %d", resp.Files.TotalFiles)
	}
	if resp.Files.TotalBytes != 350 {
		t.Fatalf("expected total_bytes=350, got %d", resp.Files.TotalBytes)
	}
	if resp.Files.TransfersToday != 1 {
		t.Fatalf("expected transfers_today=1, got %d", resp.Files.TransfersToday)
	}
}
