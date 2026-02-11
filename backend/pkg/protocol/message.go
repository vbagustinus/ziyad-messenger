package protocol

import (
	"encoding/json"
)

// Message represents a chat message.
type Message struct {
	ID        string      `json:"id"`
	ChannelID string      `json:"channel_id"`
	SenderID  string      `json:"sender_id"`
	Timestamp int64       `json:"timestamp"`
	Type      MessageType `json:"type"`
	Content   []byte      `json:"content"` // Encrypted payload
	Nonce     []byte      `json:"nonce"`
	Signature []byte      `json:"signature"`
}

// SendMessageRequest is the payload for sending a message.
type SendMessageRequest struct {
	ChannelID string      `json:"channel_id"`
	Content   []byte      `json:"content"`
	Nonce     []byte      `json:"nonce"`
	Signature []byte      `json:"signature"`
	Type      MessageType `json:"type"`
}

// SendMessageResponse is the acknowledgment.
type SendMessageResponse struct {
	MessageID string `json:"message_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// EncodeMessage converts the message to bytes.
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMessage parses bytes into a Message.
func DecodeMessage(data []byte) (*Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	return &m, err
}
