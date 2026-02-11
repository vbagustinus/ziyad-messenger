package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"lan-chat/protocol"
)

const (
	// MulticastAddress is the address for service discovery.
	MulticastAddress = "224.0.0.251:5353"
	// BroadcastInterval is how often we announce our presence.
	BroadcastInterval = 5 * time.Second
)

type DiscoveryService struct {
	NodeID    string
	ClusterID string
	Port      int
	Peers     map[string]*protocol.PeerInfo
	mu        sync.RWMutex
	conn      *net.UDPConn
}

func NewDiscoveryService(nodeID, clusterID string, port int) *DiscoveryService {
	return &DiscoveryService{
		NodeID:    nodeID,
		ClusterID: clusterID,
		Port:      port,
		Peers:     make(map[string]*protocol.PeerInfo),
	}
}

func (s *DiscoveryService) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", MulticastAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to listen on multicast UDP: %w", err)
	}
	s.conn = conn
	defer conn.Close()

	log.Printf("Discovery Service started on %s", MulticastAddress)

	// Start Broadcaster
	go s.broadcastLoop(ctx, addr)

	// Start Listener
	go s.listenLoop(ctx)

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutting down Discovery Service...")
	return nil
}

func (s *DiscoveryService) broadcastLoop(ctx context.Context, addr *net.UDPAddr) {
	ticker := time.NewTicker(BroadcastInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.broadcast(addr)
		}
	}
}

func (s *DiscoveryService) broadcast(addr *net.UDPAddr) {
	packet := &protocol.DiscoveryPacket{
		ClusterID: s.ClusterID,
		NodeID:    s.NodeID,
		Address:   fmt.Sprintf(":%d", s.Port), // In real implementation, resolve local IP
		Priority:  1,
		PublicKey: []byte("dummy-key"),
	}

	data, err := packet.Encode()
	if err != nil {
		log.Printf("Error encoding packet: %v", err)
		return
	}

	// We need a separate dialer for sending multicast if listening on the same port causes issues,
	// but ListenMulticastUDP returns a connection that can ReadFrom. Writing usually needs a DialUDP.
	// For simplicity in this example, we re-dial to send.
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Error dialing UDP: %v", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("Error sending broadcast: %v", err)
	}
}

func (s *DiscoveryService) listenLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set a read deadline to allow checking for context cancellation
			s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, src, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Error reading UDP: %v", err)
				continue
			}

			go s.handlePacket(buf[:n], src)
		}
	}
}

func (s *DiscoveryService) handlePacket(data []byte, src *net.UDPAddr) {
	packet, err := protocol.DecodeDiscoveryPacket(data)
	if err != nil {
		log.Printf("Invalid packet from %s: %v", src, err)
		return
	}

	if packet.NodeID == s.NodeID {
		// Ignore own packets
		return
	}

	if packet.ClusterID != s.ClusterID {
		// Ignore other clusters
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	peer, exists := s.Peers[packet.NodeID]
	if !exists {
		log.Printf("New Peer Discovered: %s at %s (IP: %s)", packet.NodeID, packet.Address, src.IP)
		s.Peers[packet.NodeID] = &protocol.PeerInfo{
			NodeID:    packet.NodeID,
			Address:   packet.Address,
			LastSeen:  time.Now(),
			PublicKey: packet.PublicKey,
		}
	} else {
		peer.LastSeen = time.Now()
		// log.Printf("Peer Heartbeat: %s", packet.NodeID) // Verbose
	}
}

func main() {
	// Parse flags/env/config in real implementation
	nodeID := "node-1"
	clusterID := "local-cluster"
	port := 8080

	if len(os.Args) > 1 {
		nodeID = os.Args[1]
	}

	svc := NewDiscoveryService(nodeID, clusterID, port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	if err := svc.Start(ctx); err != nil {
		log.Fatalf("Service terminated: %v", err)
	}
}
