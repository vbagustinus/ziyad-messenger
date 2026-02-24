# Disaster Recovery Plan

## Objectives

- **RTO (Recovery Time Objective)**: Target time to restore service (e.g. 4 hours for full cluster).
- **RPO (Recovery Point Objective)**: Maximum acceptable data loss (e.g. last 1 hour; depends on backup frequency).

## Backup Scope

| Data | Location | Backup method | Frequency |
|------|----------|----------------|------------|
| **Auth DB** | `backend/deploy/data/shared/platform.db` (or configured path) | File copy or SQLite backup | Daily or before changes |
| **Messaging DB** | `backend/deploy/data/shared/platform.db` | File copy or `.backup` | Continuous or hourly |
| **Channels** | Channel service DB | Same as above | Same |
| **Audit log** | `backend/deploy/data/audit/audit.log` | Append-only copy; do not truncate | Real-time or hourly |
| **File storage** | Encrypted files on disk | Filesystem backup | Daily |
| **Raft data** | `backend/deploy/data/raft/` (or configured) | Full directory copy | Before upgrade; optional continuous |
| **PKI** | CA cert and key (offline) | Secure offline copy | Once + after any issuance |
| **Config** | YAML/env and secrets | Versioned copy | On change |

## Backup Procedures

- **SQLite**: Use `sqlite3 .backup` or stop service and copy file; ensure no corruption.
- **Audit log**: Copy entire file; verify last line hash chain if possible.
- **Encrypted files**: Backup as opaque blobs; keys (if stored server-side) must be backed up separately and securely.

## Restore Procedures

### Single Node Failure

1. Replace or repair node.
2. Restore config and secrets.
3. Restore SQLite and audit log from latest backup.
4. Restore file storage if applicable.
5. Start services; verify health and connectivity.

### Full Cluster Loss

1. Provision new nodes (same or larger count).
2. Restore one node from backup (e.g. latest Raft snapshot + DBs + audit).
3. Start that node as **single-node cluster** (bootstrap).
4. Join remaining nodes to cluster (Raft join).
5. Optionally restore DBs on other nodes from same backup and let Raft reconcile, or replicate from first node if replication is configured.
6. Restore audit log to central audit service or append to new log with note.
7. Restore file storage to File Transfer service.
8. Verify discovery, auth, messaging, and audit.

### Data Center / Site Loss

- **DR site**: Maintain secondary site with periodic backup transfer (sneakernet or isolated link for air-gap).
- **Restore at DR**: Same as full cluster loss; use latest backup transferred to DR.
- **RPO**: Determined by backup transfer frequency.

## CA and PKI Recovery

- **Root CA**: Restore from offline backup; keep offline.
- **Server/client certs**: Re-issue from restored CA if private keys lost; revoke old certs.
- **CRL**: Regenerate and publish after recovery.

## Testing

- **Restore drill**: Periodically restore from backup to test environment; verify DBs and audit integrity.
- **Failover test**: Kill leader or node; verify election and client reconnect.
- **Document**: Keep runbooks for single-node, cluster, and site recovery; assign owners.
