# K-O11y

[English](README.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [中文](README.zh-CN.md)

メトリクス、ログ、トレース、ServiceMap 可視化を備えたセルフホスト型 Kubernetes 可観測性プラットフォームです。

[Wondermove](https://wondermove.net) が開発し、[SigNoz](https://github.com/SigNoz/signoz) (MIT License) をフォークして S3 ストレージティアリング、SSO テナント管理、ServiceMap などの機能を拡張しています。

## 主な機能

- **ServiceMap**: マイクロサービス依存関係トポロジーの可視化
- **S3 3-Tier Storage**: Hot (EBS) / Warm (S3 Standard) / Cold (S3 Glacier IR) ティアリング
- **SSO Tenant Auto-Lock**: JWT ベースのマルチテナント SSO とワークスペースの自動バインディング
- **Distributed Tracing**: ClickHouse ベースのトレース保存とクエリ
- **Metrics Monitoring**: Prometheus 互換のメトリクス収集とダッシュボード
- **Log Management**: 構造化ログの収集と検索
- **Alerting**: AlertManager ベースのアラートルールおよびチャンネル管理

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

> **ビルド済み Docker イメージおよび Helm チャートは提供されていません。**
> ソースからイメージをビルドし、ご自身のレジストリに push してください。

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

ご自身のレジストリにビルドして push してください:

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

ご自身の OCI レジストリにパッケージして push してください:

```bash
# Package
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent

# Push to your registry
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

インストール前に `values.yaml` のイメージレジストリをご自身のレジストリに変更してください。

## Related Repositories

| Repository | Description |
|-----------|-------------|
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm charts、インストールスクリプト、ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | CRD プロセッサー付きカスタム OTel Collector |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | License Guard 付き SigNoz OTel Collector |

## Maintainers

[Wondermove](https://wondermove.net) が開発・管理しています。

## License

MIT License - [LICENSE](packages/signoz/LICENSE) を参照

このプロジェクトは [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.) から派生しています。
完全な著作権情報は [NOTICE](NOTICE) をご確認ください。
