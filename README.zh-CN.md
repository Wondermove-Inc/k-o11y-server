# K-O11y

[English](README.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [中文](README.zh-CN.md)

一个支持指标、日志、链路追踪和 ServiceMap 可视化的自托管 Kubernetes 可观测性平台。

由 [Wondermove](https://wondermove.net) 构建，基于 [SigNoz](https://github.com/SigNoz/signoz) (MIT License) 进行二次开发，新增 S3 存储分层、SSO 租户管理和 ServiceMap 等功能。

## 主要功能

- **ServiceMap**：微服务依赖拓扑可视化
- **S3 3-Tier Storage**：Hot (EBS) / Warm (S3 Standard) / Cold (S3 Glacier IR) 分层存储
- **SSO Tenant Auto-Lock**：基于 JWT 的多租户 SSO 与工作空间自动绑定
- **Distributed Tracing**：基于 ClickHouse 的链路存储与查询
- **Metrics Monitoring**：Prometheus 兼容的指标采集与仪表盘
- **Log Management**：结构化日志采集与检索
- **Alerting**：基于 AlertManager 的告警规则与通道管理

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

> **不提供预构建的 Docker 镜像和 Helm Chart。**
> 需从源码自行构建镜像并推送至您的私有镜像仓库。

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

构建并推送至您的私有镜像仓库：

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

打包并推送至您的 OCI 镜像仓库：

```bash
# Package
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent

# Push to your registry
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

安装前请将 `values.yaml` 中的镜像仓库地址更新为您的私有仓库。

## Related Repositories

| Repository | Description |
|-----------|-------------|
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm charts、安装脚本、ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | 带 CRD 处理器的自定义 OTel Collector |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | 带 License Guard 的 SigNoz OTel Collector |

## Maintainers

由 [Wondermove](https://wondermove.net) 构建与维护。

## License

MIT License - 参见 [LICENSE](packages/signoz/LICENSE)

本项目派生自 [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.)。
完整版权信息请参见 [NOTICE](NOTICE)。
