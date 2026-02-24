# Build Stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

# Copy generic files
COPY go.work go.work.sum* ./
COPY backend/pkg ./backend/pkg

# Copy service sources
COPY backend/services ./backend/services
COPY backend/admin-api ./backend/admin-api

# Ensure dependencies are available (if any external)
ENV GOSUMDB=off
ENV CGO_ENABLED=1
RUN go mod download || true

# Build Discovery Service
RUN go build -o /bin/discovery ./backend/services/discovery

# Build Audit Service
RUN go build -o /bin/audit ./backend/services/audit

# Build Auth Service
RUN go build -o /bin/auth ./backend/services/auth

# Build Messaging Service
RUN go build -o /bin/messaging ./backend/services/messaging

# Build PKI Service
RUN go build -o /bin/pki ./backend/services/pki

# Build Presence Service
RUN go build -o /bin/presence ./backend/services/presence

# Build File Transfer Service
RUN go build -o /bin/filetransfer ./backend/services/filetransfer

# Build Cluster Service
RUN go build -o /bin/cluster ./backend/services/cluster

# Build Admin Service
RUN go build -o /bin/admin-service ./backend/admin-api/cmd/server

# Final Stage (Unified Image for Simplicity, or use separate targets)
FROM alpine:latest

WORKDIR /app

# Copy binaries
COPY --from=builder /bin/discovery /usr/local/bin/discovery
COPY --from=builder /bin/audit /usr/local/bin/audit
COPY --from=builder /bin/auth /usr/local/bin/auth
COPY --from=builder /bin/messaging /usr/local/bin/messaging
COPY --from=builder /bin/pki /usr/local/bin/pki
COPY --from=builder /bin/presence /usr/local/bin/presence
COPY --from=builder /bin/filetransfer /usr/local/bin/filetransfer
COPY --from=builder /bin/cluster /usr/local/bin/cluster
COPY --from=builder /bin/admin-service /usr/local/bin/admin-service

# Expose ports
EXPOSE 8080 8081 8082 8083 8084 8085 8086 8090 5353/udp

# Default entrypoint (overridden by K8s args)
CMD ["/usr/local/bin/discovery"]
