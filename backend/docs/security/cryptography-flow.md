# Cryptography Flow

## Overview

- **Transport**: TLS 1.3 (QUIC and TCP for WebSocket/gRPC); certificate-based server and optional client authentication.
- **Message content**: End-to-end encryption (E2EE) so that only sender and intended recipients can decrypt; server stores ciphertext.
- **At-rest**: Optional AES-GCM for sensitive DB fields and file storage; keys from local KMS or derived.

## E2EE High-Level Flow

```mermaid
sequenceDiagram
    participant A as Client A
    participant B as Client B
    participant S as Server (Messaging Router)

    Note over A,B: Key agreement (X3DH / Double Ratchet)
    A->>S: Publish identity key + signed prekeys
    B->>S: Publish identity key + signed prekeys
    A->>A: X3DH with B's keys → shared secret
    A->>A: Init Double Ratchet session for B

    Note over A,S: Send message
    A->>A: Encrypt payload (AES-GCM) with session key
    A->>S: Message (ciphertext + nonce + signature)
    S->>S: Store ciphertext; route by channel_id
    S->>B: Deliver ciphertext

    B->>B: Double Ratchet decrypt
    B->>B: Verify signature; display
```

- **Key exchange**: X3DH (or similar) for initial shared secret; then **Double Ratchet** (Signal-style) for forward secrecy and post-compromise security.
- **Algorithm**: AES-256-GCM for symmetric encryption; ECDH (P-256 or X25519) for key agreement; Ed25519 or ECDSA for signatures.

## Per-Message Flow (Current Protocol Alignment)

The existing `protocol.Message` and `SendMessageRequest` already carry:

- **Content**: Opaque (in practice, E2EE ciphertext from client).
- **Nonce**: Used for AES-GCM or passed through for verification.
- **Signature**: Sender’s signature over (channel_id, timestamp, content hash) to prove origin and integrity.

Server does **not** decrypt content; it stores and routes by `channel_id` and `sender_id`. Decryption is client-side only.

## Transport Layer

| Layer | Algorithm | Purpose |
|-------|-----------|---------|
| QUIC | TLS 1.3 | Server auth, confidentiality, integrity |
| gRPC | TLS 1.3 (mTLS) | Service-to-service auth and encryption |
| Certificate | ECDSA P-256 / RSA 2048+ | Server and client certs from local CA |

## Key Roles

| Key type | Owner | Use |
|----------|--------|-----|
| Identity key (long-term) | Client/Device | X3DH and signing |
| Signed prekey | Client/Device | X3DH one-time use |
| Ratchet root/chain keys | Session (client) | Double Ratchet message keys |
| Server TLS key | Node | QUIC/gRPC server auth |
| CA key | Local PKI | Sign server/client certs |

## Signature and Integrity

- **Message signature**: Computed over (channel_id, sender_id, timestamp, content_hash) so the server and recipients can verify origin without decrypting.
- **Audit log**: Each entry includes prev_hash (hash chain) so tampering is detectable.
