# Compliance-Ready Logging Design

## Objectives

- **Audit trail**: Immutable record of who did what, when, and to which resource.
- **Integrity**: Log entries are hash-chained so tampering is detectable.
- **Retention**: Configurable retention period; secure deletion when no longer required.
- **Query and export**: Support for compliance queries and export in standard formats.

## Audit Event Model

Each event (aligns with existing `AuditEvent` where applicable) includes:

| Field | Type | Description |
|-------|------|-------------|
| **timestamp** | RFC3339 or Unix ms | Event time (server-authoritative) |
| **actor_id** | string | User or service account ID |
| **action** | string | Verb (e.g. login, channel.join, message.delete) |
| **target_resource** | string | Resource type and ID (e.g. channel:abc, user:xyz) |
| **details** | string or JSON | Additional context (IP, user agent, outcome) |
| **prev_hash** | string | SHA-256 of previous log entry (hash chain) |
| **entry_hash** | string | SHA-256 of this entry (optional; can be derived) |

## Events to Log (Minimum Set)

| Action | When | actor_id | target_resource |
|--------|------|----------|-----------------|
| **auth.login** | Successful login | user_id | — |
| **auth.login_failure** | Failed login | username or IP | — |
| **auth.logout** | Logout | user_id | — |
| **auth.token_revoke** | Token revoked | admin_id | user_id |
| **user.create** | User created | admin_id | user_id |
| **user.update** | User/role updated | admin_id | user_id |
| **user.delete** | User disabled/deleted | admin_id | user_id |
| **role.assign** | Role assigned | admin_id | user_id |
| **channel.create** | Channel created | user_id | channel_id |
| **channel.update** | Channel settings changed | user_id | channel_id |
| **channel.delete** | Channel deleted/archived | user_id | channel_id |
| **channel.join** | User joined channel | user_id | channel_id |
| **channel.leave** | User left channel | user_id | channel_id |
| **channel.member_add** | Member added | user_id | channel_id, member_id |
| **message.send** | Message sent (metadata only; not content) | user_id | channel_id, message_id |
| **message.delete** | Message deleted | user_id | message_id |
| **file.upload** | File uploaded | user_id | file_id, channel_id |
| **file.download** | File downloaded | user_id | file_id |
| **file.delete** | File deleted | user_id | file_id |
| **device.register** | Device bound | user_id | device_id |
| **device.revoke** | Device revoked | user_id or admin_id | device_id |
| **broadcast.send** | System broadcast sent | user_id | — |
| **system.config_change** | Config updated | admin_id | config_key |
| **audit.export** | Audit log exported | user_id | — |

## Hash Chain

- **prev_hash**: First entry has fixed genesis (e.g. zeros); each next entry’s hash is computed over (timestamp, actor_id, action, target_resource, details, prev_hash).
- **Verification**: On read, recompute hash for each entry and compare with next entry’s prev_hash; break indicates tampering.
- **Storage**: Append-only file; no in-place edit or delete (deletion = secure wipe of file or retention purge at end).

## Storage and Retention

- **Primary**: Append-only file (e.g. `audit.log`) on Audit Service; optional SQLite index for query by actor, action, time range.
- **Retention**: Policy (e.g. 7 years); after retention, secure delete (overwrite and truncate) or archive to offline storage.
- **Access control**: Only **auditor** role (and super_admin) can read or export; all access to audit log is itself logged (actor_id = reader, action = audit.read / audit.export).

## Export and Formats

- **Export API**: Filter by time range, actor, action; return JSON Lines or CSV; optionally signed or checksummed.
- **Compliance formats**: Map to standard schemas (e.g. CEF, JSON audit schema) if required by regulation.
- **Integrity**: Export can include chain of prev_hash for verification by auditor.

## Correlation and Log Aggregation

- **Request ID**: Each API request gets a correlation ID; same ID in service logs and (where applicable) in audit details for traceability.
- **Structured logs**: All services emit JSON logs (level, message, request_id, service); can be aggregated to central log store (e.g. Loki, Elasticsearch) in same LAN for ops and security analysis.

## Summary

- **Append-only**, **hash-chained** audit log.
- **Comprehensive** security-relevant actions with actor and resource.
- **Retention** and **secure deletion**; **export** with integrity for compliance.
- **Access to audit** restricted and itself audited.
