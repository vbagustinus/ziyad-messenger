# Enterprise-Grade Local Communication Platform

**LAN-native • Offline-first • Zero cloud dependency • Self-hosted • Air-gapped capable**

A distributed, secure internal messaging and collaboration platform inspired by Slack and IP Messenger, designed for high-security environments: military networks, industrial plants, hospitals, government buildings, secure campuses, offshore platforms, and air-gapped enterprises.

## Principles

- **LAN-native architecture** — All traffic stays on your network
- **Offline-first** — Operation without internet; local cache and sync
- **No external SaaS** — Fully self-hosted; no cloud dependency
- **Air-gapped support** — Static peer list; deploy via offline media
- **Security-first** — E2EE, local PKI, RBAC, full audit trail
- **Production-grade** — HA, failover, disaster recovery, compliance-ready logging

## Repository Layout

| Path                     | Contents                                                                                                             |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| **docs/**                | [Full documentation index](docs/README.md) — architecture, IAM, security, protocols, schemas, deployment, compliance |
| **services/**            | Go microservices: discovery, auth, cluster, messaging, presence, audit, filetransfer, pki                            |
| **admin-service/**       | Go Admin API: User/Role management, monitoring orchestrator, audit logging                                           |
| **admin-dashboard/**     | Next.js Admin UI: Enterprise control panel                                                                           |
| **pkg/protocol/**        | Shared protocol types: message, discovery, E2EE                                                                      |
| **clients/flutter_app/** | Flutter client (desktop/mobile)                                                                                      |
| **deploy/**              | Dockerfile, Helm chart, docker-compose                                                                               |

## Quick Start (Development)

**Full steps:** [Run and check locally](docs/development/local-run.md)

```bash
# Build and start all services (Docker)
make build
make run-all

# Quick checks (run after services are up)
make test-auth-register   # create user admin/password
make test-auth-login      # get token
make test-send-message    # send to channel "general"
make health               # health + presence
```

**Stop:** `make stop-all`

Services listen on: Discovery 8080/UDP 5353, Messaging 8081, File 8082, Presence 8083, Audit 8084, Cluster 8085, Auth 8086.

## Documentation

All deliverables are in **docs/**:

- **Architecture**: [System](docs/architecture/system-architecture.md), [Network topology](docs/architecture/network-topology.md), [Cluster](docs/architecture/cluster-design.md), [Service mesh](docs/architecture/service-mesh.md)
- **IAM**: [Model](docs/iam/iam-model.md), [RBAC matrix](docs/iam/rbac-matrix.md)
- **Security**: [Model](docs/security/security-model.md), [Cryptography](docs/security/cryptography-flow.md), [Key management](docs/security/key-management.md)
- **Protocols & schemas**: [Protocols](docs/protocols/protocol-definitions.md), [Database](docs/schemas/database-schemas.md), [Messages](docs/schemas/message-schemas.md)
- **Deployment**: [Deployment](docs/deployment/deployment-architecture.md), [DevOps](docs/deployment/devops-flow.md), [Upgrade](docs/deployment/upgrade-strategy.md), [Disaster recovery](docs/deployment/disaster-recovery.md)
- **Compliance**: [Logging design](docs/compliance/compliance-logging.md)

## License

Proprietary or as specified in the project.
