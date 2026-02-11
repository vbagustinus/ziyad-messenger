# DevOps Flow

## Overview

- **CI**: Build, test, and produce artifacts (binaries, Docker images) from version control.
- **CD**: Deploy to on-premise or air-gapped environments via approved pipelines; no automatic pull from public internet in secure deployments.
- **No external SaaS**: Use self-hosted or local tooling (e.g. internal Git, CI runner, container registry).

## CI Pipeline

| Stage | Actions |
|-------|--------|
| **Lint** | Go vet, golangci-lint; Flutter analyze |
| **Test** | Unit tests (Go, Flutter); integration tests against local services |
| **Build** | Go: build for target OS/arch; Flutter: build for target platforms |
| **Image** | Docker build for each service; tag by git commit or version |
| **Artifact** | Store binaries and images in internal artifact store or registry |

Example (conceptual):

```bash
# Build all Go services
make build

# Run tests
make test

# Build Docker images
docker build -f deploy/Dockerfile -t lan-chat/services:${VERSION} .
# Or separate Dockerfile per service
```

## Versioning and Release

- **Semantic versioning**: MAJOR.MINOR.PATCH for releases.
- **Changelog**: Document breaking changes, new features, and security fixes.
- **Signing**: Sign binaries and images (e.g. cosign, GPG) for integrity in air-gapped delivery.

## Deployment Methods

| Method | Use case |
|--------|----------|
| **docker-compose** | Single node or dev; `docker-compose up -d` |
| **Helm** | Kubernetes; `helm install lan-chat ./deploy/helm/lan-chat -f overrides.yaml` |
| **Ansible / Salt / Scripts** | Bare metal or VMs; copy binaries, config, and systemd units; restart services |
| **Offline bundle** | Tar or USB with versioned binaries, images, config templates, and upgrade docs |

## Configuration Management

- **Config files**: YAML or env per service; templated with environment or cluster-specific values.
- **Secrets**: From files (e.g. mounted volumes) or secret manager (e.g. Vault) when available; never committed.
- **Discovery**: Cluster ID, peer list, and optional mDNS config in config file or env.

## Monitoring and Observability

- **Logs**: Structured (JSON) to stdout; collected by existing log pipeline (e.g. Fluentd, Loki) or tailed to files.
- **Metrics**: Prometheus endpoints on each service; scrape from admin network only.
- **Alerts**: Define alerts on failure rates, Raft leader changes, disk usage; notify via internal channel or pager.

## Access and Change Control

- **Deployments**: Only from designated pipeline or approved change process.
- **Production**: Require approval and rollback plan; use blue-green or rolling update where applicable (see [Upgrade Strategy](./upgrade-strategy.md)).
