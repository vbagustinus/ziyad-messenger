package cluster

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type NodeInfo struct {
	NodeID    string `json:"node_id"`
	Address   string `json:"address"`
	Role      string `json:"role"`
	LastSeen  int64  `json:"last_seen"`
	IsLeader  bool   `json:"is_leader"`
}

type ClusterStatus struct {
	ClusterID   string     `json:"cluster_id"`
	LeaderID    string     `json:"leader_id"`
	Nodes       []NodeInfo `json:"nodes"`
	TotalNodes  int        `json:"total_nodes"`
	Healthy     bool       `json:"healthy"`
}

func Status(c *gin.Context) {
	now := time.Now().Unix()
	st := ClusterStatus{
		ClusterID:  "local-cluster",
		LeaderID:   "node-1",
		TotalNodes: 1,
		Healthy:    true,
		Nodes: []NodeInfo{
			{NodeID: "node-1", Address: "localhost:8085", Role: "leader", LastSeen: now, IsLeader: true},
		},
	}
	c.JSON(http.StatusOK, st)
}
