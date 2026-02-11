package system

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

type HealthResponse struct {
	Status        string  `json:"status"`
	Version       string  `json:"version"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
}

func Health(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.JSON(http.StatusOK, HealthResponse{
		Status:        "ok",
		Version:       "1.0",
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		MemoryAllocMB: float64(m.Alloc) / (1024 * 1024),
	})
}
