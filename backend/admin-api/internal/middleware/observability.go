package middleware

import (
	"admin-service/internal/auth"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDKey = "request_id"

type EndpointMetric struct {
	Method         string  `json:"method"`
	Route          string  `json:"route"`
	Count          uint64  `json:"count"`
	ErrorCount     uint64  `json:"error_count"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	MaxLatencyMs   float64 `json:"max_latency_ms"`
	LastLatencyMs  float64 `json:"last_latency_ms"`
	LastStatusCode int     `json:"last_status_code"`
	UpdatedAt      int64   `json:"updated_at"`
}

type endpointCounter struct {
	method         string
	route          string
	count          uint64
	errorCount     uint64
	totalLatency   float64
	maxLatency     float64
	lastLatency    float64
	lastStatus     int
	lastUpdatedAt  int64
	latencySamples []float64
}

var metricsStore = struct {
	mu        sync.RWMutex
	endpoints map[string]endpointCounter
}{
	endpoints: map[string]endpointCounter{},
}

const maxLatencySamples = 512
const accessLogDirEnv = "ADMIN_ACCESS_LOG_DIR"

var accessLogSink = struct {
	mu   sync.Mutex
	date string
	file *os.File
}{
	date: "",
	file: nil,
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(RequestIDKey, rid)
		c.Writer.Header().Set(RequestIDHeader, rid)
		c.Next()
	}
}

func AccessLogAndMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		method := c.Request.Method
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		status := c.Writer.Status()
		requestID := c.GetString(RequestIDKey)

		userID := ""
		username := ""
		role := ""
		if v, ok := c.Get(auth.ClaimsKey); ok {
			if claims, ok := v.(*auth.Claims); ok {
				userID = claims.UserID
				username = claims.Username
				role = claims.Role
			}
		}

		recordEndpointMetric(method, route, status, latencyMs)

		entry := map[string]any{
			"ts":             time.Now().UTC().Format(time.RFC3339Nano),
			"request_id":     requestID,
			"method":         method,
			"route":          route,
			"path":           c.Request.URL.Path,
			"status":         status,
			"latency_ms":     latencyMs,
			"client_ip":      c.ClientIP(),
			"user_agent":     c.Request.UserAgent(),
			"admin_user_id":  userID,
			"admin_username": username,
			"admin_role":     role,
		}
		if b, err := json.Marshal(entry); err == nil {
			writeAccessLogLine(string(b))
		}
	}
}

func writeAccessLogLine(line string) {
	fmt.Fprintln(os.Stdout, line)
	writeAccessLogToFile(line)
}

func writeAccessLogToFile(line string) {
	dir := strings.TrimSpace(os.Getenv(accessLogDirEnv))
	if dir == "" {
		return
	}

	accessLogSink.mu.Lock()
	defer accessLogSink.mu.Unlock()

	nowDate := time.Now().Format("2006-01-02")
	if accessLogSink.file == nil || accessLogSink.date != nowDate {
		if accessLogSink.file != nil {
			_ = accessLogSink.file.Close()
			accessLogSink.file = nil
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return
		}
		p := filepath.Join(dir, "access-"+nowDate+".log")
		f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		accessLogSink.file = f
		accessLogSink.date = nowDate
	}
	_, _ = io.WriteString(accessLogSink.file, line+"\n")
}

func recordEndpointMetric(method, route string, status int, latencyMs float64) {
	key := method + " " + route
	now := time.Now().Unix()

	metricsStore.mu.Lock()
	defer metricsStore.mu.Unlock()

	cur := metricsStore.endpoints[key]
	cur.method = method
	cur.route = route
	cur.count++
	if status >= 400 {
		cur.errorCount++
	}
	cur.totalLatency += latencyMs
	if latencyMs > cur.maxLatency {
		cur.maxLatency = latencyMs
	}
	cur.lastLatency = latencyMs
	cur.lastStatus = status
	cur.lastUpdatedAt = now
	cur.latencySamples = append(cur.latencySamples, latencyMs)
	if len(cur.latencySamples) > maxLatencySamples {
		cur.latencySamples = cur.latencySamples[len(cur.latencySamples)-maxLatencySamples:]
	}
	metricsStore.endpoints[key] = cur
}

func SnapshotEndpointMetrics() map[string]EndpointMetric {
	metricsStore.mu.RLock()
	defer metricsStore.mu.RUnlock()

	out := make(map[string]EndpointMetric, len(metricsStore.endpoints))
	for k, v := range metricsStore.endpoints {
		avg := 0.0
		if v.count > 0 {
			avg = v.totalLatency / float64(v.count)
		}
		p95 := percentile(v.latencySamples, 0.95)
		p99 := percentile(v.latencySamples, 0.99)
		out[k] = EndpointMetric{
			Method:         v.method,
			Route:          v.route,
			Count:          v.count,
			ErrorCount:     v.errorCount,
			AvgLatencyMs:   avg,
			P95LatencyMs:   p95,
			P99LatencyMs:   p99,
			MaxLatencyMs:   v.maxLatency,
			LastLatencyMs:  v.lastLatency,
			LastStatusCode: v.lastStatus,
			UpdatedAt:      v.lastUpdatedAt,
		}
	}
	return out
}

func percentile(samples []float64, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	if p <= 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	cp := append([]float64(nil), samples...)
	sort.Float64s(cp)
	idx := int(math.Ceil(p*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func resetEndpointMetrics() {
	metricsStore.mu.Lock()
	metricsStore.endpoints = map[string]endpointCounter{}
	metricsStore.mu.Unlock()
}

func resetAccessLogSink() {
	accessLogSink.mu.Lock()
	if accessLogSink.file != nil {
		_ = accessLogSink.file.Close()
		accessLogSink.file = nil
	}
	accessLogSink.date = ""
	accessLogSink.mu.Unlock()
}

func ResetObservabilityForTest() {
	resetEndpointMetrics()
	resetAccessLogSink()
}
