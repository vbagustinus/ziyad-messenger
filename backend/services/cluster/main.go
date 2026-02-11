package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

// RaftNode represents a node in the Raft cluster.
type RaftNode struct {
	NodeID   string
	RaftDir  string
	RaftBind string
	State    string // Leader, Follower, Candidate
}

func NewRaftNode(nodeID, raftDir, raftBind string) *RaftNode {
	return &RaftNode{
		NodeID:   nodeID,
		RaftDir:  raftDir,
		RaftBind: raftBind,
		State:    "Follower",
	}
}

// Start initializes the Raft node (Stub).
func (n *RaftNode) Start() error {
	log.Printf("Starting Raft Node %s on %s...", n.NodeID, n.RaftBind)

	// Simulate Raft Bootstrap
	go func() {
		time.Sleep(5 * time.Second)
		n.State = "Leader"
		log.Printf("Node %s is now the Leader", n.NodeID)
		n.startHeartbeat()
	}()

	return nil
}

func (n *RaftNode) startHeartbeat() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		// Log heartbeat to simulate leader activity
		// log.Printf("Leader Heartbeat...")
	}
}

// JoinHandler handles join requests from other nodes.
func (n *RaftNode) JoinHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	addr := r.URL.Query().Get("addr")
	log.Printf("Received join request from %s at %s", nodeID, addr)
	w.WriteHeader(http.StatusOK)
}

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "node-1"
	}

	raftNode := NewRaftNode(nodeID, "./raft-data", ":10001")
	if err := raftNode.Start(); err != nil {
		log.Fatalf("Failed to start Raft node: %v", err)
	}

	http.HandleFunc("/join", raftNode.JoinHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	log.Printf("Cluster Management Service started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
