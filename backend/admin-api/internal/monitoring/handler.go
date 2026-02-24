package monitoring

import (
	"admin-service/internal/db"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type NetworkStats struct {
	NodesOnline   int     `json:"nodes_online"`
	PeersKnown    int     `json:"peers_known"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	LatencyMs     float64 `json:"latency_ms"`
}

type UserStats struct {
	TotalUsers  int `json:"total_users"`
	OnlineNow   int `json:"online_now"`
	ActiveToday int `json:"active_today"`
}

type MessageStats struct {
	MessagesLastHour int64 `json:"messages_last_hour"`
	MessagesToday    int64 `json:"messages_today"`
	TotalMessages    int64 `json:"total_messages"`
}

type FileStats struct {
	TotalFiles     int64 `json:"total_files"`
	TotalBytes     int64 `json:"total_bytes"`
	TransfersToday int64 `json:"transfers_today"`
}

type SystemStats struct {
	GoVersion     string  `json:"go_version"`
	NumCPU        int     `json:"num_cpu"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

type OverviewResponse struct {
	GeneratedAt int64        `json:"generated_at"`
	Network     NetworkStats `json:"network"`
	Users       UserStats    `json:"users"`
	Messages    MessageStats `json:"messages"`
	Files       FileStats    `json:"files"`
	System      SystemStats  `json:"system"`
}

var startTime = time.Now()
var overviewCache = struct {
	mu        sync.Mutex
	value     OverviewResponse
	expiresAt time.Time
	ttl       time.Duration
}{
	ttl: 5 * time.Second,
}

func tableExists(name string) bool {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`, name).Scan(&count)
	return err == nil && count > 0
}

func Network(c *gin.Context) {
	nodes := 1
	if tableExists("cluster_nodes") {
		var detected int
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM cluster_nodes`).Scan(&detected)
		if detected > 0 {
			nodes = detected
		}
	}
	st := NetworkStats{
		NodesOnline:   nodes,
		PeersKnown:    nodes,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		LatencyMs:     0,
	}
	c.JSON(http.StatusOK, st)
}

func Users(c *gin.Context) {
	var total int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total)

	var online int
	if tableExists("user_presence") {
		cutoff := time.Now().Add(-60 * time.Second).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM user_presence WHERE status = 1 AND last_seen >= ?`, cutoff).Scan(&online)
	} else if tableExists("messages") {
		cutoffMs := time.Now().Add(-5 * time.Minute).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(DISTINCT sender_id) FROM messages WHERE timestamp >= ?`, cutoffMs).Scan(&online)
	}

	var activeToday int
	if tableExists("messages") {
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(DISTINCT sender_id) FROM messages WHERE timestamp >= ?`, cutoffDayMs).Scan(&activeToday)
	} else {
		activeToday = total
	}

	st := UserStats{
		TotalUsers:  total,
		OnlineNow:   online,
		ActiveToday: activeToday,
	}
	c.JSON(http.StatusOK, st)
}

func Messages(c *gin.Context) {
	var hour, day, total int64
	if tableExists("messages") {
		cutoffHourMs := time.Now().Add(-1 * time.Hour).UnixMilli()
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE timestamp >= ?`, cutoffHourMs).Scan(&hour)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE timestamp >= ?`, cutoffDayMs).Scan(&day)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&total)
	} else {
		cutoffHour := time.Now().Add(-1 * time.Hour).Unix()
		cutoffDay := time.Now().Add(-24 * time.Hour).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%' AND timestamp >= ?`, cutoffHour).Scan(&hour)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%' AND timestamp >= ?`, cutoffDay).Scan(&day)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%'`).Scan(&total)
	}
	st := MessageStats{
		MessagesLastHour: hour,
		MessagesToday:    day,
		TotalMessages:    total,
	}
	c.JSON(http.StatusOK, st)
}

func Files(c *gin.Context) {
	var totalFiles int64
	var totalBytes int64
	var todayTransfers int64

	if tableExists("files") {
		_ = db.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(size_bytes), 0) FROM files`).Scan(&totalFiles, &totalBytes)
		cutoffDay := time.Now().Add(-24 * time.Hour).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE created_at >= ?`, cutoffDay).Scan(&todayTransfers)
	} else if tableExists("messages") {
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE type = 3`).Scan(&totalFiles)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE type = 3 AND timestamp >= ?`, cutoffDayMs).Scan(&todayTransfers)
	}

	st := FileStats{
		TotalFiles:     totalFiles,
		TotalBytes:     totalBytes,
		TransfersToday: todayTransfers,
	}
	c.JSON(http.StatusOK, st)
}

func System(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	st := SystemStats{
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		MemoryAllocMB: float64(m.Alloc) / (1024 * 1024),
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}
	c.JSON(http.StatusOK, st)
}

func buildOverview() OverviewResponse {
	var nodes int
	if tableExists("cluster_nodes") {
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM cluster_nodes`).Scan(&nodes)
	}
	if nodes <= 0 {
		nodes = 1
	}
	network := NetworkStats{
		NodesOnline:   nodes,
		PeersKnown:    nodes,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		LatencyMs:     0,
	}

	var totalUsers, onlineNow, activeToday int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	if tableExists("user_presence") {
		cutoff := time.Now().Add(-60 * time.Second).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM user_presence WHERE status = 1 AND last_seen >= ?`, cutoff).Scan(&onlineNow)
	} else if tableExists("messages") {
		cutoffMs := time.Now().Add(-5 * time.Minute).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(DISTINCT sender_id) FROM messages WHERE timestamp >= ?`, cutoffMs).Scan(&onlineNow)
	}
	if tableExists("messages") {
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(DISTINCT sender_id) FROM messages WHERE timestamp >= ?`, cutoffDayMs).Scan(&activeToday)
	} else {
		activeToday = totalUsers
	}
	users := UserStats{
		TotalUsers:  totalUsers,
		OnlineNow:   onlineNow,
		ActiveToday: activeToday,
	}

	var msgHour, msgDay, msgTotal int64
	if tableExists("messages") {
		cutoffHourMs := time.Now().Add(-1 * time.Hour).UnixMilli()
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE timestamp >= ?`, cutoffHourMs).Scan(&msgHour)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE timestamp >= ?`, cutoffDayMs).Scan(&msgDay)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&msgTotal)
	} else {
		cutoffHour := time.Now().Add(-1 * time.Hour).Unix()
		cutoffDay := time.Now().Add(-24 * time.Hour).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%' AND timestamp >= ?`, cutoffHour).Scan(&msgHour)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%' AND timestamp >= ?`, cutoffDay).Scan(&msgDay)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action LIKE '%message%'`).Scan(&msgTotal)
	}
	messages := MessageStats{
		MessagesLastHour: msgHour,
		MessagesToday:    msgDay,
		TotalMessages:    msgTotal,
	}

	var totalFiles, totalBytes, transfersToday int64
	if tableExists("files") {
		_ = db.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(size_bytes), 0) FROM files`).Scan(&totalFiles, &totalBytes)
		cutoffDay := time.Now().Add(-24 * time.Hour).Unix()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE created_at >= ?`, cutoffDay).Scan(&transfersToday)
	} else if tableExists("messages") {
		cutoffDayMs := time.Now().Add(-24 * time.Hour).UnixMilli()
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE type = 3`).Scan(&totalFiles)
		_ = db.DB.QueryRow(`SELECT COUNT(*) FROM messages WHERE type = 3 AND timestamp >= ?`, cutoffDayMs).Scan(&transfersToday)
	}
	files := FileStats{
		TotalFiles:     totalFiles,
		TotalBytes:     totalBytes,
		TransfersToday: transfersToday,
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	system := SystemStats{
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		MemoryAllocMB: float64(m.Alloc) / (1024 * 1024),
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}

	return OverviewResponse{
		GeneratedAt: time.Now().Unix(),
		Network:     network,
		Users:       users,
		Messages:    messages,
		Files:       files,
		System:      system,
	}
}

func Overview(c *gin.Context) {
	now := time.Now()
	overviewCache.mu.Lock()
	if now.Before(overviewCache.expiresAt) {
		cached := overviewCache.value
		overviewCache.mu.Unlock()
		c.JSON(http.StatusOK, cached)
		return
	}
	overviewCache.mu.Unlock()

	fresh := buildOverview()

	overviewCache.mu.Lock()
	overviewCache.value = fresh
	overviewCache.expiresAt = time.Now().Add(overviewCache.ttl)
	overviewCache.mu.Unlock()
	c.JSON(http.StatusOK, fresh)
}

func resetOverviewCache() {
	overviewCache.mu.Lock()
	overviewCache.value = OverviewResponse{}
	overviewCache.expiresAt = time.Time{}
	overviewCache.mu.Unlock()
}
