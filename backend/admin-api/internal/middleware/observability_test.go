package middleware

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPercentileNearestRank(t *testing.T) {
	v := percentile([]float64{10, 20, 30, 40, 50}, 0.95)
	if v != 50 {
		t.Fatalf("expected p95=50, got %v", v)
	}
	v2 := percentile([]float64{10, 20, 30, 40, 50}, 0.99)
	if v2 != 50 {
		t.Fatalf("expected p99=50, got %v", v2)
	}
}

func TestSnapshotEndpointMetricsHasPercentiles(t *testing.T) {
	resetEndpointMetrics()
	recordEndpointMetric("GET", "/x", 200, 10)
	recordEndpointMetric("GET", "/x", 200, 20)
	recordEndpointMetric("GET", "/x", 200, 30)
	recordEndpointMetric("GET", "/x", 200, 40)
	recordEndpointMetric("GET", "/x", 200, 50)

	metrics := SnapshotEndpointMetrics()
	m, ok := metrics["GET /x"]
	if !ok {
		t.Fatalf("missing endpoint metric")
	}
	if m.P95LatencyMs != 50 {
		t.Fatalf("expected p95=50, got %v", m.P95LatencyMs)
	}
	if m.P99LatencyMs != 50 {
		t.Fatalf("expected p99=50, got %v", m.P99LatencyMs)
	}
}

func TestWriteAccessLogToFile(t *testing.T) {
	resetAccessLogSink()
	tmp := t.TempDir()
	t.Setenv(accessLogDirEnv, tmp)

	writeAccessLogToFile(`{"event":"a"}`)
	writeAccessLogToFile(`{"event":"b"}`)
	resetAccessLogSink()

	matches, err := filepath.Glob(filepath.Join(tmp, "access-*.log"))
	if err != nil {
		t.Fatalf("glob log file: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one log file, got %d", len(matches))
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, `"event":"a"`) || !strings.Contains(content, `"event":"b"`) {
		t.Fatalf("expected both log lines written, got: %s", content)
	}
}
