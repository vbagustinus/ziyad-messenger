package system

import (
	"admin-service/internal/middleware"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type MetricsResponse struct {
	GeneratedAt int64                                `json:"generated_at"`
	Endpoints   map[string]middleware.EndpointMetric `json:"endpoints"`
}

func Metrics(c *gin.Context) {
	c.JSON(http.StatusOK, MetricsResponse{
		GeneratedAt: time.Now().Unix(),
		Endpoints:   middleware.SnapshotEndpointMetrics(),
	})
}
