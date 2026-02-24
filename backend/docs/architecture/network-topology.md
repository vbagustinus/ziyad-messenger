# Network Topology

## Single LAN

```mermaid
flowchart LR
    subgraph LAN["Single LAN (e.g. 192.168.1.0/24)"]
        N1["Node 1\nDiscovery + Core"]
        N2["Node 2\nCore"]
        N3["Node 3\nEdge"]
        C1["Client A"]
        C2["Client B"]
    end
    N1 <-->|UDP Broadcast\nmDNS| N2
    N1 <-->|QUIC/gRPC| N3
    C1 -->|QUIC| N1
    C2 -->|QUIC| N3
```

- **UDP broadcast** (e.g. 224.0.0.251:5353) and **mDNS** for service discovery within the subnet.
- **QUIC** (and WebSocket fallback) for client-to-node and node-to-node secure transport.
- All nodes share the same **ClusterID**; discovery filters by ClusterID to ignore other clusters.

## Multi-LAN (Multi-Subnet)

```mermaid
flowchart TB
    subgraph LAN1["LAN 1 - 10.0.1.0/24"]
        N1["Node 1"]
        N2["Node 2"]
    end
    subgraph LAN2["LAN 2 - 10.0.2.0/24"]
        N3["Node 3"]
        N4["Node 4"]
    end
    subgraph Router["L3 Router / Gateway"]
        R["Router"]
    end
    N1 <-->|UDP/mDNS| N2
    N3 <-->|UDP/mDNS| N4
    N1 <-->|QUIC/gRPC\nMulti-subnet routing| R
    R <-->|QUIC/gRPC| N3
```

- **Multi-subnet routing**: Discovery can use configured peer list or optional relay nodes that span subnets.
- **Relay nodes**: Optional nodes with interfaces in multiple VLANs/LANs to propagate discovery and route traffic.
- **Static peer configuration**: For air-gapped or restricted networks, peer addresses can be supplied via config (no broadcast dependency).

## VLAN-Aware Networking

```mermaid
flowchart TB
    subgraph VLAN10["VLAN 10 - Management"]
        M1["Mgmt Node"]
    end
    subgraph VLAN20["VLAN 20 - Core Services"]
        C1["Core 1"]
        C2["Core 2"]
    end
    subgraph VLAN30["VLAN 30 - Clients"]
        E1["Edge 1"]
        E2["Edge 2"]
    end
    M1 <-->|gRPC TLS| C1
    C1 <-->|gRPC TLS| C2
    C1 <-->|QUIC| E1
    C2 <-->|QUIC| E2
```

- **VLAN segmentation**: Core services, management, and client-facing edge can be placed in separate VLANs.
- **Firewall rules**: Allow only required ports (QUIC, gRPC, UDP discovery) between VLANs; deny by default.
- **mDNS**: Typically limited to single broadcast domain; cross-VLAN discovery via configured peers or relay.

## Port Matrix

| Purpose | Protocol | Port(s) | Direction |
|---------|----------|---------|-----------|
| mDNS discovery | UDP | 5353 | Within subnet |
| UDP broadcast discovery | UDP | Configurable (e.g. 5354) | Within subnet |
| QUIC (client & server) | UDP | 443 or configurable | Client ↔ Node, Node ↔ Node |
| WebSocket fallback | TCP | 443 or 8443 | Client ↔ Node |
| gRPC (internal) | TCP/TLS | 50051 (example) | Service ↔ Service |
| HTTP (admin/health) | TCP | 8080–8090 | Admin ↔ Services |

## Air-Gapped and Restricted Networks

- **No internet**: All binaries, CA roots, and config are deployed via offline media or internal package repo.
- **Discovery**: Rely on **static peer list** and **config-driven** cluster membership; disable UDP broadcast if not allowed.
- **Time sync**: Use local NTP server or manual sync; certificate validity and audit timestamps depend on it.
