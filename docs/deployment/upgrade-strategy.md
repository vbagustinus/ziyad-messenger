# Upgrade Strategy

## Principles

- **Backward compatibility**: Newer nodes should tolerate older peers during rolling upgrade; avoid breaking wire or storage format when possible.
- **Rolling upgrade**: Update one node at a time in a cluster to avoid full outage.
- **Rollback**: Keep previous binary/image available; document rollback steps and data compatibility.

## Pre-Upgrade

1. **Backup**: Full backup of SQLite DBs, audit log, Raft data, and config (see [Disaster Recovery](./disaster-recovery.md)).
2. **Compatibility**: Check release notes for schema or protocol changes; run migrations if required.
3. **Drain**: If using LB, drain traffic from node to be upgraded (optional).

## Rolling Upgrade (Cluster)

1. Choose **non-leader** node first (or demote if single leader).
2. Stop service on that node; replace binary or image with new version.
3. Run any **DB migrations** (e.g. SQLite ALTER) before or after start, as documented for the version.
4. Start service; verify health and re-join to Raft if applicable.
5. Repeat for other followers.
6. Last: upgrade **leader** (trigger leader step-down, then upgrade; new leader elected).

## Single Node

1. Stop all services.
2. Backup data and config.
3. Replace binaries/images; run migrations.
4. Start services; smoke test.

## Schema and Protocol Changes

- **Additive only when possible**: New columns nullable or with defaults; new API fields optional.
- **Breaking changes**: Document in release notes; provide migration script or multi-phase upgrade (e.g. deploy version that supports both old and new, then switch, then remove old).
- **Raft**: Ensure Raft library version and log format are compatible across upgrade; otherwise one-time migration or new cluster.

## Version Skew

- **Short skew**: During rolling upgrade, mixed versions run; avoid deploying versions that cannot interoperate (e.g. new server that rejects old client messages).
- **Support window**: Document minimum client version supported by server and vice versa.

## Air-Gapped Upgrade

1. Build or obtain upgrade bundle (new binaries/images, migration scripts, docs) in connected environment.
2. Transfer bundle via approved channel (USB, internal net).
3. Apply using same rolling or single-node procedure; no pull from internet.

## Rollback

1. Stop current version.
2. Restore previous binary/image.
3. If DB schema was upgraded, run rollback migration if provided; otherwise restore from backup if schema reverted.
4. Start and verify.
