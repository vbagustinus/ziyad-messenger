package protocol

import (
	"errors"
)

// RatchetSession represents the state of a double ratchet session.
type RatchetSession struct {
	RootKey   []byte
	ChainKey  []byte
	NextKey   []byte
	RatchetID string
}

// SignalProtocolHandler defines the interface for E2EE operations.
type SignalProtocolHandler interface {
	InitializeSession(recipientID string, identityKey []byte, signedPreKey []byte) (*RatchetSession, error)
	EncryptMessage(session *RatchetSession, plaintext []byte) ([]byte, error)
	DecryptMessage(session *RatchetSession, ciphertext []byte) ([]byte, error)
}

// DummySignalHandler is a placeholder implementation.
type DummySignalHandler struct{}

func (h *DummySignalHandler) InitializeSession(recipientID string, identityKey []byte, signedPreKey []byte) (*RatchetSession, error) {
	// In a real implementation, perform X3DH key exchange here.
	return &RatchetSession{
		RootKey:   []byte("root-key"),
		ChainKey:  []byte("chain-key"),
		RatchetID: "session-" + recipientID,
	}, nil
}

func (h *DummySignalHandler) EncryptMessage(session *RatchetSession, plaintext []byte) ([]byte, error) {
	// Simulate encryption (In reality: AES-GCM)
	ciphertext := make([]byte, len(plaintext))
	copy(ciphertext, plaintext)
	// Append dummy logic to show something happened
	return append(ciphertext, []byte("-encrypted")...), nil
}

func (h *DummySignalHandler) DecryptMessage(session *RatchetSession, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 10 {
		return nil, errors.New("ciphertext too short")
	}
	// Simulate decryption
	return ciphertext[:len(ciphertext)-10], nil
}
