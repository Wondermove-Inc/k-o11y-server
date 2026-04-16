# K-O11y

[English](README.md) | [한국어](README.ko.md)

A self-hosted Kubernetes observability platform with metrics, logs, traces, and ServiceMap visualization.


Built by [Wondermove](https://wondermove.net), forked from [SigNoz](https://github.com/SigNoz/signoz) (MIT License) with custom extensions for S3 storage tiering, SSO tenant management, and ServiceMap.

## Key Features

- **ServiceMap**: Microservice dependency topology visualization
- **S3 3-Tier Storage**: Hot (EBS) / Warm (S3 Standard) / Cold (S3 Glacier IR) tiering
- **SSO Tenant Auto-Lock**: JWT-based multi-tenant SSO with automatic workspace binding
- **Distributed Tracing**: ClickHouse-based trace storage and query
- **Metrics Monitoring**: Prometheus-compatible metric collection and dashboards
- **Log Management**: Structured log collection and search
- **Alerting**: AlertManager-based alert rules and channel management

## Architecture

### 2-Tier Distributed Observability

```
┌─────────────────────────────────────────────────────────┐
│  Agent Clusters (multiple)                               │
│  ┌─────────────────┐  ┌─────────────────┐               │
│  │ OTel DaemonSet  │  │ OTel Deployment │               │
│  │ - hostmetrics   │  │ - kube-state    │               │
│  │ - kubeletstats  │  │ - k8s_cluster   │               │
│  │ - filelog       │  │ - k8sEvents     │               │
│  └────────┬────────┘  └────────┬────────┘               │
│           │    ┌───────────┐   │                         │
│           │    │   Beyla   │   │                         │
│           │    │  (eBPF)   │   │                         │
│           │    └─────┬─────┘   │                         │
└───────────┼──────────┼─────────┼─────────────────────────┘
            │          │         │
            └──────────┼─────────┘
                       │ OTLP gRPC :4317
                       ▼
┌─────────────────────────────────────────────────────────┐
│  Host Cluster (central)                                  │
│  ┌──────────────────┐                   ┌─────────────┐  │
│  │ SigNoz OTel      │                   │  SigNoz UI  │  │
│  │ Collector        │──┐            ┌───│ + Core API  │  │
│  └──────────────────┘  │            │   └─────────────┘  │
└────────────────────────┼────────────┼────────────────────┘
                         │            │
                         ▼            │
                   ┌────────────┐     │
                   │ ClickHouse │─────┘
                   └────────────┘
```

## Project Structure

```
k-o11y/
├── packages/
│   ├── core/                        # Go backend (ServiceMap, S3 Tiering API)
│   │   ├── cmd/main.go              # Entrypoint
│   │   ├── internal/
│   │   │   ├── batch/               # ServiceMap batch processor
│   │   │   ├── config/              # Environment-based configuration
│   │   │   ├── domain/servicemap/   # Domain models
│   │   │   ├── handler/             # HTTP handlers (Gin router)
│   │   │   ├── service/             # Business logic
│   │   │   └── repository/          # ClickHouse data access
│   │   └── Makefile
│   │
│   └── signoz/                      # SigNoz fork (UI + Query Service)
│       ├── frontend/                # React frontend
│       ├── pkg/                     # Go backend packages
│       │   ├── crypto/              # AES-256-GCM encryption (S3 creds)
│       │   ├── http/middleware/     # SSO + Tenant Auto-Lock
│       │   ├── k8s/                 # K8s Job management
│       │   └── query-service/       # ClickHouse query + S3 tiering
│       ├── cmd/community/           # Community build entrypoint
│       └── Makefile
│
├── NOTICE                           # Attribution
└── README.md
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.24, Gin, ClickHouse |
| Frontend | React 18, TypeScript, Ant Design |
| Telemetry | OpenTelemetry Collector, Beyla eBPF |
| Storage | ClickHouse + S3 (tiering) |
| Auth | JWT (RS256), SSO |

## Build

> **Pre-built Docker images and Helm charts are not provided.**
> You must build images from source and push to your own registry.

### Prerequisites

- Go 1.24+
- Node.js 16.15+
- Docker 20.10+
- A container registry (e.g., ghcr.io, Docker Hub, Harbor)

### Core Backend

```bash
cd packages/core
export CLICKHOUSE_HOST=<your-clickhouse-host>
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=signoz_traces
go run cmd/main.go
```

### SigNoz Backend (Community)

```bash
cd packages/signoz
make go-run-community
```

### Frontend

```bash
cd packages/signoz/frontend
CI=1 yarn install
yarn dev
```

### Docker Images

Build and push to your own registry:

```bash
# Core API
cd packages/core
docker build -t <your-registry>/observability/core:v1.0.0 -f deployments/Dockerfile .
docker push <your-registry>/observability/core:v1.0.0

# SigNoz Hub (community build)
cd packages/signoz
make go-build-community
docker build -t <your-registry>/observability/hub:v1.0.0 -f cmd/community/Dockerfile .
docker push <your-registry>/observability/hub:v1.0.0
```

### Helm Charts

Package and push to your own OCI registry:

```bash
# Package
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent

# Push to your registry
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

Then update `values.yaml` image registries to match your registry before installing.

## Related Repositories

| Repository | Description |
|-----------|-------------|
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm charts, install scripts, ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | Custom OTel Collector with CRD processor |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | SigNoz OTel Collector with License Guard |

## Maintainers

Built and maintained by [Wondermove](https://wondermove.net).

## License

MIT License - See [LICENSE](packages/signoz/LICENSE)

This project is derived from [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.).
See [NOTICE](NOTICE) for full attribution.
