package protocol

import (
	"encoding/json"
	"time"
)

// MessageType defines the type of message being exchanged.
type MessageType int

const (
	MessageTypeUnknown MessageType = iota
	MessageTypeText
	MessageTypeImage
	MessageTypeFile
	MessageTypeSystem
	MessageTypeVoice
)

// DiscoveryPacket represents the payload sent over UDP for node discovery.
// In a real implementation, this would be generated from Protobuf.
type DiscoveryPacket struct {
	ClusterID string `json:"cluster_id"`
	NodeID    string `json:"node_id"`
	Address   string `json:"address"` // IP:Port
	Priority  int32  `json:"priority"`
	PublicKey []byte `json:"public_key"`
}

// PeerInfo represents a discovered node.
type PeerInfo struct {
	NodeID    string    `json:"node_id"`
	Address   string    `json:"address"`
	LastSeen  time.Time `json:"last_seen"`
	PublicKey []byte    `json:"public_key"`
}

// Encode converts the packet to bytes (JSON for simplicity without protoc).
func (p *DiscoveryPacket) Encode() ([]byte, error) {
	return json.Marshal(p)
}

// Decode parses bytes into a DiscoveryPacket.
func DecodeDiscoveryPacket(data []byte) (*DiscoveryPacket, error) {
	var p DiscoveryPacket
	err := json.Unmarshal(data, &p)
	return &p, err
}
