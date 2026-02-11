# Enterprise Local Communication Platform — Documentation Index

**Product positioning**: *Enterprise-grade Local Communication Platform* — LAN-native, offline-first, zero cloud dependency, self-hosted, air-gapped capable.

**Use cases**: Military networks, industrial plants, hospitals, government buildings, secure campuses, offshore platforms, mining operations, air-gapped enterprises.

---

## Architecture

| Document | Description |
|----------|-------------|
| [System Architecture](architecture/system-architecture.md) | Enterprise system diagram, core services, design principles |
| [Network Topology](architecture/network-topology.md) | Single LAN, multi-LAN, VLAN-aware, air-gapped |
| [Cluster Design](architecture/cluster-design.md) | Raft consensus, failover, leader election, join protocol |
| [Service Mesh](architecture/service-mesh.md) | Inter-service gRPC/TLS, mTLS, observability |

## Identity & Access Management

| Document | Description |
|----------|-------------|
| [IAM Model](iam/iam-model.md) | Account lifecycle, RBAC, department segmentation, zero-trust |
| [RBAC Matrix](iam/rbac-matrix.md) | Roles, permissions, role→permission matrix |

## Security

| Document | Description |
|----------|-------------|
| [Security Model](security/security-model.md) | Trust zones, threat model, device provisioning |
| [Cryptography Flow](security/cryptography-flow.md) | E2EE, transport TLS, algorithms |
| [Key Management](security/key-management.md) | Key hierarchy, storage, rotation, revocation |

## Protocols & Schemas

| Document | Description |
|----------|-------------|
| [Protocol Definitions](protocols/protocol-definitions.md) | Discovery, auth, messaging, file, presence, audit, cluster |
| [Database Schemas](schemas/database-schemas.md) | SQLite tables for users, messages, channels, audit, files |
| [Message Schemas](schemas/message-schemas.md) | Wire and storage message format, types, routing |

## Deployment & Operations

| Document | Description |
|----------|-------------|
| [Deployment Architecture](deployment/deployment-architecture.md) | On-premise, cluster, multi-LAN, air-gapped, containers |
| [DevOps Flow](deployment/devops-flow.md) | CI/CD, versioning, config, monitoring |
| [Upgrade Strategy](deployment/upgrade-strategy.md) | Rolling upgrade, schema changes, rollback |
| [Disaster Recovery](deployment/disaster-recovery.md) | Backup scope, restore procedures, RTO/RPO |

## Compliance

| Document | Description |
|----------|-------------|
| [Compliance Logging](compliance/compliance-logging.md) | Audit event model, hash chain, retention, export |

---

## Tech Stack Summary

- **Backend**: Go; microservices (Discovery, Auth, Identity, Messaging Router, Channel, File Transfer, Presence, Audit, Device Registry, Cluster).
- **Networking**: QUIC, WebSocket, gRPC, UDP discovery, mDNS.
- **Data**: SQLite (edge/core), optional distributed DB; encrypted file storage; append-only audit log.
- **Clients**: Flutter (desktop/mobile), Web Admin Panel.
- **Deployment**: Docker, optional Kubernetes/Helm; on-premise, air-gapped, local cluster.

## Repository Layout (Reference)

- `services/` — Go microservices (discovery, auth, messaging, cluster, presence, audit, filetransfer, pki).
- `pkg/protocol/` — Shared protocol types (message, discovery, e2ee).
- `clients/flutter_app/` — Flutter client.
- `deploy/` — Dockerfile, Helm chart, docker-compose.
- `docs/` — This documentation set.
