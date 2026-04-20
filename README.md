<div align="center">

<img src="https://raw.githubusercontent.com/Wondermove-Inc/k-o11y-server/main/packages/signoz/frontend/public/Logos/ko11y_logo_large.svg" alt="K-O11y" width="120" />

# K-O11y Server

**K-O11y Server — the backend. Self-hosted observability platform with ServiceMap, S3 Tiering, and SSO.**

[English](README.md) | [한국어](README.ko.md)

[![Project Status: WIP](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Wondermove-Inc/k-o11y-server/blob/main/packages/signoz/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/Wondermove-Inc/k-o11y-server?style=social)](https://github.com/Wondermove-Inc/k-o11y-server/stargazers)
[![Release](https://img.shields.io/github/v/release/Wondermove-Inc/k-o11y-server?label=release)](https://github.com/Wondermove-Inc/k-o11y-server/releases)

Built on [ClickHouse](https://clickhouse.com/) and the [OpenTelemetry](https://opentelemetry.io/) ecosystem.

</div>

---

## ✨ Features

- 🗺️ **ServiceMap** — Microservice dependency topology visualization with batch processing
- 💾 **S3 3-Tier Storage** — Hot (EBS) → Warm (S3 Standard) → Cold (S3 Glacier IR) tiering
- 🔐 **SSO Tenant Auto-Lock** — JWT-based multi-tenant SSO with automatic workspace binding
- 🔍 **Distributed Tracing** — ClickHouse-based trace storage and query
- 📊 **Metrics Monitoring** — Prometheus-compatible metric collection and dashboards
- 📜 **Log Management** — Structured log collection and search
- 🔔 **Alerting** — AlertManager-based alert rules and channel management
- 🔒 **AES-256-GCM Encryption** — Encrypted storage of S3 credentials and sensitive config

---

## 🏗️ Architecture

K-O11y Server is a **monorepo backend** composed of two packages: a Go-based `core` API (ServiceMap batch processor, S3 Tiering API) and the `packages/signoz` directory providing the React frontend and Query Service. Both talk to a shared ClickHouse cluster.

```mermaid
flowchart TB
    subgraph Clients["Client Layer"]
        User[👤 User / Operator]
        OTel[OTel Collectors<br/>from Agent clusters]
    end

    subgraph Server["K-O11y Server (this repo)"]
        subgraph CorePkg["packages/core (Go 1.24)"]
            CoreAPI[Core HTTP API<br/>Gin Router]
            BatchProc[ServiceMap<br/>Batch Processor]
            CoreRepo[ClickHouse<br/>Repository]
        end

        subgraph SigNozPkg["packages/signoz"]
            Frontend[Frontend UI<br/>React 18 + Ant Design]
            QueryService[Query Service<br/>+ S3 Tiering + SSO]
            Crypto[AES-256-GCM<br/>Crypto Module]
            K8sJob[K8s Job Manager]
        end

        Frontend --> QueryService
        Frontend --> CoreAPI
        CoreAPI --> BatchProc
        CoreAPI --> CoreRepo
        QueryService --> Crypto
        QueryService --> K8sJob
    end

    subgraph Storage["Storage Layer (external)"]
        CH[(ClickHouse<br/>dedicated VM)]
        S3[(S3 Tiered<br/>Hot / Warm / Cold)]
    end

    User --> Frontend
    OTel -->|OTLP ingest<br/>via Gateway| CH
    CoreRepo --> CH
    QueryService --> CH
    CH -.TTL MOVE / Lifecycle.-> S3

    classDef core fill:#fff3e0,stroke:#f57c00,color:#000
    classDef signoz fill:#e8f5e9,stroke:#2e7d32,color:#000
    classDef storage fill:#f3e5f5,stroke:#7b1fa2,color:#000
    class CoreAPI,BatchProc,CoreRepo core
    class Frontend,QueryService,Crypto,K8sJob signoz
    class CH,S3 storage
```

**Data flow:**

1. **Ingest** — OTel Collectors (from Agent clusters) ship telemetry through the OTel Gateway into ClickHouse
2. **Batch** — `packages/core` runs a periodic ServiceMap batch processor that builds topology data from trace spans
3. **Query** — `packages/signoz` Query Service serves the UI (metrics/logs/traces) and manages SSO + S3 tiering config
4. **Visualize** — Frontend consumes both Core API (ServiceMap) and the standard Query Service (metrics/logs/traces views)

### Core Backend Layers

```
Handler (HTTP endpoints, Gin Router)
    ↓
Service (business logic, topology build)
    ↓
Repository (ClickHouse queries, data access)
    ↓
Infrastructure (DB connection management)
```

---

## 🚀 Quick Start

> **Part of a larger system.** Full production deployment (Host + Agent clusters, ClickHouse VM, OTel Gateway) is documented in the umbrella repo: [Wondermove-Inc/k-o11y](https://github.com/Wondermove-Inc/k-o11y).
>
> This README covers **local development and building** of the server components only.

### Prerequisites

| Tool | Min Version | Purpose |
|------|-------------|---------|
| Go | 1.24.0 | Core backend build |
| Node.js | 16.15.0 | Frontend build |
| Docker | 20.10+ | Image build & push |
| kubectl | 1.25+ | K8s cluster ops |
| make | - | Build automation |

Also required: a running ClickHouse instance (see [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) for VM setup).

### Run Core API Locally

```bash
cd packages/core

# Required env vars
export CLICKHOUSE_HOST=<YOUR_IP>
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=signoz_traces
export CLICKHOUSE_USER=default
export CLICKHOUSE_PASSWORD=<password>

# Optional (defaults shown)
# export APP_PORT=3001                    # default: 3001
# export APP_ENV=local                    # default: local
# export BATCH_SERVICEMAP_ENABLED=true    # default: true
# export BATCH_SERVICEMAP_INTERVAL=20s    # default: 20s

go run cmd/main.go
```

All batch settings ship with sensible defaults — no tuning needed to start.
To disable the ServiceMap batch, set `BATCH_SERVICEMAP_ENABLED=false`.

### Run Backend (Community Build)

```bash
cd packages/signoz

# 1. Create local env file (first time only)
cp .env.example .env.local
# Edit .env.local — set ClickHouse DSN and other real values

# 2. Start dev infrastructure (local ClickHouse + OTel Collector)
make devenv-up

# 3. Run Go backend
make go-run-community
```

**Env precedence (highest first):** make CLI args → shell export → `.env.local` → defaults.

```bash
# Example: pass DSN directly via make
make go-run-community SIGNOZ_CLICKHOUSE_DSN=tcp://default:'pass'@host:9000
```

### Run Frontend

```bash
cd packages/signoz/frontend
CI=1 yarn install
yarn dev
```

### Swagger API Docs

Once the Core server is running, open:

```
http://localhost:3001/swagger-ui/
```

---

## 📦 Packages

K-O11y Server is a monorepo with two packages.

| Package | Role | Tech |
|---------|------|------|
| [`packages/core`](packages/core) | ServiceMap API, batch processor, S3 Tiering helpers | Go 1.24 + Gin + ClickHouse |
| [`packages/signoz`](packages/signoz) | Backend (React UI + Query Service) | React 18 + Go + Webpack 5 |

### Project Structure

```
k-o11y-server/
├── packages/
│   ├── core/                        # Go backend (ServiceMap, S3 Tiering API)
│   │   ├── cmd/main.go              # Entrypoint
│   │   ├── internal/
│   │   │   ├── batch/               # ServiceMap batch processor
│   │   │   ├── config/              # Environment-based configuration
│   │   │   ├── domain/servicemap/   # Domain models
│   │   │   ├── handler/             # HTTP handlers (Gin router)
│   │   │   ├── service/             # Business logic
│   │   │   ├── repository/          # ClickHouse data access
│   │   │   ├── infrastructure/      # DB connection management
│   │   │   └── utils/               # Utilities
│   │   ├── pkg/                     # Shared packages (logger, response, errors)
│   │   ├── deployments/Dockerfile   # Multi-stage build
│   │   └── Makefile
│   │
│   └── signoz/                      # Backend UI + Query Service
│       ├── frontend/                # React frontend
│       ├── pkg/                     # Go backend packages
│       │   ├── crypto/              # AES-256-GCM encryption (S3 creds)
│       │   ├── http/middleware/     # SSO + Tenant Auto-Lock
│       │   ├── k8s/                 # K8s Job management
│       │   └── query-service/       # ClickHouse query + S3 tiering
│       ├── cmd/community/           # Community build entrypoint
│       └── Makefile
│
├── Makefile                         # Root build (interactive build-and-push)
├── NOTICE                           # Attribution
└── README.md
```

### Tech Stack

**Core package**

| Category | Tech | Version |
|----------|------|---------|
| Language | Go | 1.24 |
| HTTP framework | Gin | 1.11 |
| Database | ClickHouse (Native API) | - |
| Metrics | Prometheus client_golang | 1.23 |
| Logging | Uber Zap + Lumberjack | 1.27 |
| API docs | Swagger (swaggo) | - |
| Container | distroless/base-debian12 | - |

**Hub package**

| Category | Tech | Version |
|----------|------|---------|
| Frontend | React + TypeScript | 18.2 / 4.0 |
| Bundler | Webpack | 5.94 |
| UI library | Ant Design | - |
| State | Redux + React Query | - |
| Backend | Go (gorilla/mux) | 1.24 |

---

## 🛠️ Build

> **Pre-built Docker images and Helm charts are not published yet.**
> You must build images from source and push to your own registry (GHCR, Harbor, Docker Hub, etc.).

### Docker Images

```bash
# Core API
cd packages/core
docker build -t <your-registry>/observability/core:v1.0.0 -f deployments/Dockerfile .
docker push <your-registry>/observability/core:v1.0.0

# Hub (community build)
cd packages/signoz
make go-build-community
docker build -t <your-registry>/observability/hub:v1.0.0 -f cmd/community/Dockerfile .
docker push <your-registry>/observability/hub:v1.0.0
```

### Interactive Build (root)

```bash
make build-and-push
# Choose:
#   1. core   → build & push packages/core
#   2. hub    → build & push packages/signoz
# Enter TAG (e.g. 0.1.3)
```

### Per-Package Build

```bash
# Core
cd packages/core
make core-build-and-push TAG=0.1.3
# → <YOUR_REGISTRY>/observability/core:0.1.3

# Trigger GitHub Actions workflow
make trigger-workflow CUSTOM_TAG=v0.1.3

# Hub package
cd packages/signoz
make o11y-build-and-push TAG=0.1.20
```

### Helm Charts

Helm charts live in the separate [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) repo. To package and push:

```bash
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

Update `values.yaml` image registries to match your registry before installing.

### Image Reference

| Package | Registry path | Base image |
|---------|---------------|------------|
| core | `<YOUR_REGISTRY>/observability/core` | distroless/base-debian12 |
| hub | `<YOUR_REGISTRY>/observability/hub` | - |

---

## 🔌 API Endpoints

Base URL: `http://<host>:3001/api/v1`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/servicemap/topology` | ServiceMap topology query |
| POST | `/servicemap/workload/details` | Workload detail info |
| POST | `/servicemap/workload/hover-info` | Workload hover info (Top 5) |
| POST | `/servicemap/edge/trace/details` | Edge (connection) trace details |

All endpoints use POST to handle large filter payloads and avoid URL-encoding issues.

---

## ⚙️ Environment Variables

### Core Package (`packages/core`)

**Server**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `APP_PORT` | `3001` | | HTTP server port |
| `APP_ENV` | `local` | | Environment (local / dev / stg / prod) |

**ClickHouse connection**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `CLICKHOUSE_HOST` | - | **required** | ClickHouse host |
| `CLICKHOUSE_PORT` | - | **required** | ClickHouse native port (typically 9000) |
| `CLICKHOUSE_DATABASE` | - | **required** | Database name (typically `signoz_traces`) |
| `CLICKHOUSE_USER` | - | | Username |
| `CLICKHOUSE_PASSWORD` | - | | Password |
| `CLICKHOUSE_TIMEOUT` | `10s` | | Connection timeout |
| `CLICKHOUSE_MAX_RETRIES` | `3` | | Max retry count |

**Batch processing**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `BATCH_SERVICEMAP_ENABLED` | `true` | | Enable ServiceMap batch |
| `BATCH_SERVICEMAP_INTERVAL` | `20s` | | Batch interval (must be > 0 when enabled) |
| `BATCH_INSERT_TIMEOUT` | `120s` | | INSERT query timeout |
| `BATCH_SAFETY_BUFFER` | `20s` | | Data stabilization wait |
| `BATCH_MAX_WINDOW` | `30s` | | Max single-batch processing window |

All batch settings have defaults — the server runs fine with none set.

**Logging**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `LOG_LEVEL` | `info` | | Log level (debug / info / warn / error) |
| `LOG_FILE` | `./logs/local-ko11y.log` | | Log file path |

### Hub Package (`packages/signoz`)

Env vars are injected via `.env.local`, make CLI args, or shell export.
See `.env.example` for the full template.

**Configurable variables (Makefile `?=`)**

| Make variable | Default | Description |
|---------------|---------|-------------|
| `SIGNOZ_CLICKHOUSE_DSN` | `tcp://127.0.0.1:9000` | ClickHouse DSN (`tcp://user:pass@host:port`) |
| `SIGNOZ_CLICKHOUSE_CLUSTER` | `cluster` | ClickHouse cluster name |
| `SIGNOZ_JWT_SECRET` | `secret` | JWT auth secret |
| `SIGNOZ_LOG_LEVEL` | `debug` | Log level |
| `SIGNOZ_SMTP_FROM` | (empty) | Sender email |
| `SIGNOZ_SMTP_HELLO` | (empty) | SMTP HELO domain |
| `SIGNOZ_SMTP_SMARTHOST` | (empty) | SMTP server (`host:port`) |
| `SIGNOZ_SMTP_USERNAME` | (empty) | SMTP auth user |
| `SIGNOZ_SMTP_PASSWORD` | (empty) | SMTP auth password |
| `SIGNOZ_SMTP_REQUIRE_TLS` | `true` | Require TLS for SMTP |

**Makefile-managed (do not override)**

| Variable | Value | Description |
|----------|-------|-------------|
| `SIGNOZ_TELEMETRYSTORE_PROVIDER` | `clickhouse` | Telemetry store |
| `SIGNOZ_ALERTMANAGER_PROVIDER` | `signoz` | AlertManager provider |
| `SIGNOZ_SQLSTORE_SQLITE_PATH` | `signoz.db` | SQLite path |
| `SIGNOZ_WEB_ENABLED` | `false` | Web UI (frontend runs separately) |

---

## 🔗 Related Repositories

K-O11y Server is one component of a larger observability platform. See the umbrella repo for the full picture.

| Repository | Description |
|------------|-------------|
| [k-o11y](https://github.com/Wondermove-Inc/k-o11y) | 🌂 Umbrella — overview, architecture, 6-step install guide |
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm charts, Go CLI installers, ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | Custom OTel Collector with CRD processor |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | OTel Collector with License Guard |

---

## 🤝 Contributing

Contributions are welcome — especially on [good first issues](https://github.com/Wondermove-Inc/k-o11y-server/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).

1. **Find an issue** labeled `good first issue` or `help wanted`
2. **Comment on the issue** to claim it (avoid duplicate work)
3. **Fork, branch, and send a PR** — scope narrowly, describe clearly
4. **Address review feedback** — maintainers will reply within a few days

See [CONTRIBUTING.md](CONTRIBUTING.md) for more.

This project follows **passive maintenance** — PRs and issues are reviewed as time allows. We aim to respond within 7 days but cannot guarantee faster turnaround.

---

## 📄 License

MIT License — see [LICENSE](packages/signoz/LICENSE).

This project is derived from [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.).
See [NOTICE](NOTICE) for full attribution.

---

## 💬 Contact

- 🐛 **Bug reports & feature requests**: [GitHub Issues](https://github.com/Wondermove-Inc/k-o11y-server/issues)
- 💭 **Questions & discussions**: Open an issue (GitHub Discussions coming soon)
- 🌐 **Website**: [wondermove.net](https://wondermove.net)

---

<div align="center">

**Built and maintained by [Wondermove](https://wondermove.net)**

</div>
