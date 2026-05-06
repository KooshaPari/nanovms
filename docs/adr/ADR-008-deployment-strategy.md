# ADR-008: Deployment Strategy

## Status

Accepted

## Context

NanoVMS needs to support multiple deployment scenarios:
- Local development
- Team server
- Multi-node cluster
- Cloud-native

## Decision

### Deployment Models

```
┌─────────────────────────────────────────────────────────────┐
│                    Deployment Spectrum                        │
│                                                             │
│  Local ─────────────────────────────────────────────── Cloud │
│                                                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │
│  │ Standalo│  │ Client/ │  │ Team    │  │ Cluster │     │
│  │ ne CLI  │  │ Server  │  │ Server  │  │ Mode    │     │
│  │         │  │         │  │         │  │         │     │
│  │ SQLite  │  │ SQLite  │  │Postgres │  │ Postgres│     │
│  │         │  │         │  │ + NATS  │  │ + NATS  │     │
│  │ No deps │  │ 1 node  │  │ 3+ nodes│  │ HA mode │     │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Local Development

```bash
# Single binary, no dependencies
curl -L https://github.com/nanovms/releases/latest | tar -xz
./nanovms run --flavor=microvm -- steamcmd
```

### Client/Server

```bash
# Server on one machine
nanovms server --store=sqlite --port=8080

# CLI connects to server
nanovms --server=http://localhost:8080 create vm --name=test
```

### Team Server

```bash
# docker-compose.yml
version: '3.8'
services:
  nanovms:
    image: nanovms/server:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - nanovms-data:/data
    environment:
      - STORE=postgres
      - POSTGRES_DSN=postgres://nanovms:password@postgres:5432/nanovms
      - EVENTS=nats
      - NATS_URL=nats://nats:4222

  postgres:
    image: postgres:16-alpine
    volumes:
      - postgres-data:/var/lib/postgresql/data

  nats:
    image: nats:latest
    volumes:
      - nats-data:/data

volumes:
  nanovms-data:
  postgres-data:
  nats-data:
```

### Cluster Mode (Kubernetes)

```yaml
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nanovms-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nanovms
  template:
    metadata:
      labels:
        app: nanovms
    spec:
      containers:
      - name: nanovms
        image: nanovms/server:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
        env:
        - name: STORE
          value: "postgres"
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef:
              name: nanovms-secrets
              key: postgres-dsn
```

## Release Strategy

### Semantic Versioning

```
v{major}.{minor}.{patch}
  │      │      │
  │      │      └── Bug fixes
  │      └────────── New features (backward compatible)
  └────────────────── Breaking changes
```

### Release Channels

| Channel | Frequency | Use Case |
|---------|-----------|----------|
| `stable` | Monthly | Production |
| `beta` | Weekly | Testing |
| `nightly` | Daily | Development |

### Artifact Distribution

```bash
# Binary releases
https://github.com/nanovms/nanovms/releases/tag/v1.0.0
├── nanovms-darwin-amd64
├── nanovms-darwin-arm64
├── nanovms-linux-amd64
├── nanovms-linux-arm64
├── nanovms-windows-amd64.exe
└── nanovms-source.tar.gz

# Package managers
brew install nanovms        # macOS
apt install nanovms        # Debian/Ubuntu
yum install nanovms        # RHEL/CentOS
choco install nanovms      # Windows
```

### Container Images

```dockerfile
# Dockerfile.server
FROM gcr.io/distroless/static:nonroot
COPY nanovms-server /nanovms-server
ENTRYPOINT ["/nanovms-server"]
```

```bash
# Image tags
docker pull nanovms/server:v1.0.0     # Specific version
docker pull nanovms/server:stable      # Stable
docker pull nanovms/server:latest      # Latest
docker pull nanovms/server:nightly     # Nightly
```

## Consequences

### Positive
- One tool for all deployment sizes
- Familiar patterns (SQLite for local, Postgres for team)
- Kubernetes-native for scale
- Easy migration between modes

### Negative
- Multiple code paths for different deployments
- More complex testing matrix
- Configuration management complexity
