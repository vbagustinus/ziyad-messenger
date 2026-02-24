# Protocol Definitions

## Transport Protocols

| Protocol | Layer | Port (example) | Use |
|----------|--------|----------------|-----|
| **QUIC** | UDP | 443 | Primary client and inter-node secure transport |
| **WebSocket** | TCP (TLS) | 443 / 8443 | Fallback for clients where QUIC is blocked |
| **gRPC** | TCP (TLS) | 50051 | Internal service-to-service |
| **UDP** | UDP | 5353 (mDNS), configurable | Discovery broadcast/unicast |
| **HTTP** | TCP | 8080–8090 | Health, admin, legacy REST where needed |

## Discovery Protocol

**Purpose**: Nodes and clients discover cluster members and gateway addresses.

### UDP Discovery Packet (current JSON alignment)

```json
{
  "cluster_id": "string",
  "node_id": "string",
  "address": "host:port",
  "priority": 1,
  "public_key": "base64 or raw bytes"
}
```

- **cluster_id**: Same for all nodes in one cluster; receivers ignore other clusters.
- **node_id**: Unique node identifier.
- **address**: Reachable address (IP:port or host:port) for QUIC/gRPC.
- **priority**: Hint for preferred node (e.g. leader or edge).
- **public_key**: Node’s public key for TLS or identity verification.

### mDNS

- Service type: e.g. `_lanchat._udp.local`
- TXT: `cluster_id`, `node_id`, `priority`
- A/AAAA: Resolve host for connection.

### Peer Info (in-memory / API)

- **node_id**, **address**, **last_seen**, **public_key** (see `protocol.PeerInfo`).

---

## Authentication Protocol

### Login (HTTP POST /login)

**Request:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "token": "JWT or opaque",
  "user_id": "string",
  "role": "string"
}
```

### Register (HTTP POST /register)

**Request:**
```json
{
  "username": "string",
  "password": "string",
  "role": "string"
}
```

**Response:** `201` with `{"id": "user-uuid"}`.

### Token Usage

- **Header**: `Authorization: Bearer <token>` or `X-User-ID` for legacy.
- **Validation**: Auth Service or gateway validates JWT (signature, expiry, optional revocation); extracts `user_id`, `role`, permissions.

---

## Messaging Protocol

### Send Message (HTTP POST /send or gRPC)

**Request (aligns with `SendMessageRequest`):**
```json
{
  "channel_id": "string",
  "content": "base64 bytes (E2EE payload)",
  "nonce": "base64",
  "signature": "base64",
  "type": 1
}
```

**MessageType enum:** `0` Unknown, `1` Text, `2` Image, `3` File, `4` System, `5` Voice.

**Response:**
```json
{
  "message_id": "string",
  "success": true,
  "error": ""
}
```

### Message Object (stored and delivered)

- **id**, **channel_id**, **sender_id**, **timestamp** (ms), **type**, **content** (ciphertext), **nonce**, **signature** (see [Message Schemas](../schemas/message-schemas.md)).

### Real-time Delivery

- **WebSocket** or **QUIC stream**: After connect and auth, client subscribes to channel(s); server pushes messages as they are persisted (topic-based routing).

---

## File Transfer Protocol

### Upload (HTTP POST /upload, multipart/form-data)

- **file**: binary file.
- **Response**: `201` with `{"file_id": "string", "key": "hex?"}` (key only in dev; production uses wrapped key or E2EE).

### Download (HTTP GET /download?id=<file_id>)

- Returns encrypted file (or decrypted if key managed server-side per policy).

---

## Presence Protocol

### Heartbeat (HTTP POST /heartbeat)

**Request:**
```json
{
  "user_id": "string",
  "status": 0
}
```

**Status:** `0` Offline, `1` Online, `2` Busy, `3` Away.

### Status (HTTP GET /status?user_id=)

**Response:**
```json
{
  "user_id": "string",
  "status": 0,
  "last_seen": "RFC3339"
}
```

---

## Audit Protocol

### Log Event (HTTP POST /log)

**Request (aligns with `AuditEvent`):**
```json
{
  "timestamp": "RFC3339",
  "actor_id": "string",
  "action": "string",
  "target_resource": "string",
  "details": "string",
  "prev_hash": "string"
}
```

- **prev_hash**: Hash of previous log entry (hash chain).
- Server computes new hash and appends line to audit log.

---

## Cluster Join Protocol

### Join (HTTP GET/POST /join)

- **Query**: `node_id`, `addr`.
- Leader adds node to Raft config and replicates; response `200 OK`.

---

## gRPC Conventions (Internal)

- **Package naming**: e.g. `lanchat.v1`.
- **Service names**: `AuthService`, `MessagingRouter`, `ChannelService`, `AuditService`, etc.
- **mTLS**: All calls use client cert; server validates cert and optional token in metadata.
- **Deadlines**: Every call has a deadline (e.g. 5–30s); propagated where applicable.
