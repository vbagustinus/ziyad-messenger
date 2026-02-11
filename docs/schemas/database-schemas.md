# Database Schemas

## Overview

- **Edge/core nodes**: SQLite for users, channels, messages, device registry, and optional presence/audit cache.
- **Optional distributed DB**: For multi-node replication of channel metadata and device registry; not required for single-node or simple cluster.

All identifiers use **TEXT** UUIDs unless noted. Timestamps in **INTEGER** (Unix ms) or **TEXT** (RFC3339) as per service.

---

## Auth Service (`users`)

```sql
CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY,
    username        TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL,
    department_id   TEXT,
    created_at      INTEGER,
    updated_at      INTEGER
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_department ON users(department_id);
```

- **role**: One of super_admin, admin, moderator, member, auditor, guest, service.
- **department_id**: Optional; for segmentation.

---

## Identity / Device Registry (optional separate or in Auth)

```sql
CREATE TABLE IF NOT EXISTS devices (
    id              TEXT PRIMARY KEY,
    account_id      TEXT NOT NULL,
    device_name     TEXT,
    client_cert_cn  TEXT,
    fingerprint     TEXT,
    bound_at        INTEGER,
    revoked_at      INTEGER,
    FOREIGN KEY (account_id) REFERENCES users(id)
);

CREATE INDEX idx_devices_account ON devices(account_id);
CREATE INDEX idx_devices_fingerprint ON devices(fingerprint);
```

---

## Messaging Router (`messages`)

```sql
CREATE TABLE IF NOT EXISTS messages (
    id          TEXT PRIMARY KEY,
    channel_id  TEXT NOT NULL,
    sender_id   TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    type        INTEGER NOT NULL,
    content     BLOB,
    nonce       BLOB,
    signature   BLOB
);

CREATE INDEX idx_messages_channel_timestamp ON messages(channel_id, timestamp);
CREATE INDEX idx_messages_sender ON messages(sender_id);
```

- **type**: 0 Unknown, 1 Text, 2 Image, 3 File, 4 System, 5 Voice.
- **content**: Opaque (E2EE ciphertext from client).

---

## Channel Service (`channels`, `channel_members`)

```sql
CREATE TABLE IF NOT EXISTS channels (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    department_id   TEXT,
    created_at      INTEGER,
    updated_at      INTEGER,
    created_by      TEXT
);

CREATE TABLE IF NOT EXISTS channel_members (
    channel_id  TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    role        TEXT,
    joined_at   INTEGER,
    PRIMARY KEY (channel_id, user_id),
    FOREIGN KEY (channel_id) REFERENCES channels(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_channel_members_user ON channel_members(user_id);
```

---

## Presence (in-memory or optional persistence)

```sql
CREATE TABLE IF NOT EXISTS presence (
    user_id     TEXT PRIMARY KEY,
    status      INTEGER NOT NULL,
    last_seen   INTEGER NOT NULL,
    updated_at  INTEGER
);
```

- **status**: 0 Offline, 1 Online, 2 Busy, 3 Away.

---

## Audit (append-only log file; optional SQLite index)

Audit events are written to an **append-only log file** (e.g. `audit.log`) with hash chaining. Optional SQLite index for querying:

```sql
CREATE TABLE IF NOT EXISTS audit_index (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp       INTEGER,
    actor_id        TEXT,
    action          TEXT,
    target_resource TEXT,
    details         TEXT,
    line_offset     INTEGER,
    entry_hash      TEXT
);

CREATE INDEX idx_audit_timestamp ON audit_index(timestamp);
CREATE INDEX idx_audit_actor ON audit_index(actor_id);
CREATE INDEX idx_audit_action ON audit_index(action);
```

---

## File Transfer (metadata; files on disk encrypted)

```sql
CREATE TABLE IF NOT EXISTS file_metadata (
    id          TEXT PRIMARY KEY,
    channel_id  TEXT,
    uploader_id TEXT NOT NULL,
    filename    TEXT,
    mime_type   TEXT,
    size_bytes  INTEGER,
    stored_path TEXT,
    created_at  INTEGER
);

CREATE INDEX idx_file_metadata_channel ON file_metadata(channel_id);
CREATE INDEX idx_file_metadata_uploader ON file_metadata(uploader_id);
```

---

## Raft / Cluster (internal to Raft library)

- Raft state (log, term, votedFor) is stored in **RaftDir** (e.g. `./raft-data`) by the Raft implementation; not part of application SQLite.
