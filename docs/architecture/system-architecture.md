# Enterprise System Architecture

## Overview

The platform is a **LAN-native, offline-first, zero-cloud** communication system designed for high-security internal networks. All components run on-premise with no external SaaS dependency.

## High-Level Architecture Diagram

```mermaid
flowchart TB
    subgraph Clients["Client Layer"]
        Flutter["Flutter Client\n(Desktop/Mobile)"]
        WebAdmin["Web Admin Panel"]
    end

    subgraph Transport["Transport Layer"]
        QUIC["QUIC\n(UDP-based TLS 1.3)"]
        WS["WebSocket\nFallback"]
        gRPC["gRPC\n(Internal)"]
    end

    subgraph Gateway["API Gateway / Edge"]
        GW["Edge Gateway\nTLS Termination\nRate Limit"]
    end

    subgraph CoreServices["Core Services (Go Microservices)"]
        Discovery["Discovery Service\nUDP + mDNS"]
        Identity["Identity Service"]
        Auth["Auth Service"]
        MsgRouter["Messaging Router"]
        Channel["Channel Service"]
        FileTransfer["File Transfer Service"]
        Presence["Presence Service"]
        Audit["Audit & Logging"]
        DeviceReg["Device Registry"]
    end

    subgraph Cluster["Cluster Layer"]
        Raft["Raft Consensus\nLeader Election"]
        Failover["Failover Nodes"]
    end

    subgraph Data["Data Layer"]
        SQLite["SQLite\n(Edge Nodes)"]
        DistDB["Distributed DB\n(Optional Core)"]
        EncFS["Encrypted File Storage"]
    end

    Flutter --> QUIC
    Flutter --> WS
    WebAdmin --> QUIC
    WebAdmin --> WS
    QUIC --> GW
    WS --> GW
    GW --> Auth
    GW --> Identity
    GW --> MsgRouter
    GW --> Channel
    GW --> FileTransfer
    GW --> Presence
    Auth --> Identity
    MsgRouter --> Channel
    MsgRouter --> Audit
    Channel --> Audit
    Auth --> Audit
    CoreServices --> gRPC
    CoreServices --> Raft
    Raft --> Failover
    MsgRouter --> SQLite
    Channel --> SQLite
    Auth --> SQLite
    FileTransfer --> EncFS
    Discovery --> CoreServices
```

## Core Services

| Service | Responsibility | Transport | Storage |
|--------|----------------|-----------|---------|
| **Discovery Service** | Node discovery via UDP broadcast + mDNS | UDP 5353, mDNS | In-memory peer map |
| **Identity Service** | Account lifecycle, device binding | gRPC/HTTP | SQLite |
| **Auth Service** | Authentication, token issuance, RBAC | HTTP/gRPC | SQLite (users, roles) |
| **Messaging Router** | Topic-based routing, message broker, persistence | gRPC/WebSocket | SQLite |
| **Channel Service** | Channel CRUD, membership, governance | gRPC | SQLite |
| **File Transfer Service** | Chunked transfer, encrypted storage | HTTP/gRPC | Encrypted filesystem |
| **Presence Service** | Online/away/busy, last seen | gRPC/WebSocket | In-memory + optional SQLite |
| **Audit & Logging Service** | Append-only audit log, chain integrity | gRPC/HTTP | Append-only log file |
| **Device Registry** | Device identity, certificate binding | gRPC | SQLite |

## Design Principles

- **LAN-native**: All traffic stays on local network; no internet egress required.
- **Offline-first**: Clients and edge nodes operate with local cache when disconnected.
- **Zero-trust**: Every request is authenticated and authorized; no implicit trust by network location.
- **Fault tolerance**: Leader election, failover nodes, and local persistence for HA.
- **Security-first**: E2EE, local PKI, certificate-based auth, and full audit trail.

## Document Index

- [Network Topology](./network-topology.md) — Single LAN, multi-LAN, VLAN
- [Cluster Design](./cluster-design.md) — Raft, leader election, replication
- [Service Mesh](./service-mesh.md) — Inter-service communication, TLS, observability
