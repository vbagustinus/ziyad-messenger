package main

import (
	"admin-service/internal/auth"
	"admin-service/internal/db"
	"admin-service/internal/middleware"
	"admin-service/internal/monitoring"
	"admin-service/internal/system"
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

func setupRouterTestDB(t *testing.T) {
	t.Helper()
	middleware.ResetObservabilityForTest()
	if db.DB != nil {
		_ = db.DB.Close()
	}
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "router_test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.DB = conn
	t.Cleanup(func() {
		_ = conn.Close()
	})
}

func newAdminMonitoringRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.AccessLogAndMetrics())
	api := r.Group("/admin")
	api.Use(middleware.JWTAuth(), middleware.RequireAdmin())
	api.GET("/monitoring/overview", monitoring.Overview)
	api.GET("/system/metrics", system.Metrics)
	return r
}

func TestMonitoringOverviewRequiresAuth(t *testing.T) {
	setupRouterTestDB(t)
	r := newAdminMonitoringRouter()

	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMonitoringOverviewRejectsNonAdminRole(t *testing.T) {
	setupRouterTestDB(t)
	r := newAdminMonitoringRouter()

	token, err := auth.GenerateToken(&auth.AdminUser{
		ID:       "u-viewer",
		Username: "viewer",
		Role:     "viewer",
	}, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestMonitoringOverviewAllowsAdminRole(t *testing.T) {
	setupRouterTestDB(t)
	r := newAdminMonitoringRouter()

	token, err := auth.GenerateToken(&auth.AdminUser{
		ID:       "u-admin",
		Username: "admin",
		Role:     "admin",
	}, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get(middleware.RequestIDHeader) == "" {
		t.Fatalf("expected %s response header", middleware.RequestIDHeader)
	}
}

func TestSystemMetricsRequiresAuth(t *testing.T) {
	setupRouterTestDB(t)
	r := newAdminMonitoringRouter()

	req := httptest.NewRequest(http.MethodGet, "/admin/system/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSystemMetricsShowsEndpointStats(t *testing.T) {
	setupRouterTestDB(t)
	r := newAdminMonitoringRouter()

	token, err := auth.GenerateToken(&auth.AdminUser{
		ID:       "u-admin",
		Username: "admin",
		Role:     "admin",
	}, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req1 := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w1.Code, w1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/admin/system/metrics", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w2.Code, w2.Body.String())
	}

	var resp struct {
		GeneratedAt int64                                `json:"generated_at"`
		Endpoints   map[string]middleware.EndpointMetric `json:"endpoints"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	key := "GET /admin/monitoring/overview"
	m, ok := resp.Endpoints[key]
	if !ok {
		t.Fatalf("expected endpoint metric %q in response", key)
	}
	if m.Count == 0 {
		t.Fatalf("expected count > 0 for %q", key)
	}
	if m.P95LatencyMs <= 0 {
		t.Fatalf("expected p95_latency_ms > 0 for %q, got %v", key, m.P95LatencyMs)
	}
	if m.P99LatencyMs <= 0 {
		t.Fatalf("expected p99_latency_ms > 0 for %q, got %v", key, m.P99LatencyMs)
	}
}
