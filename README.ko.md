<div align="center">

<img src="https://raw.githubusercontent.com/Wondermove-Inc/k-o11y-server/main/packages/signoz/frontend/public/Logos/ko11y_logo_large.svg" alt="K-O11y" width="120" />

# K-O11y Server

**K-O11y Server — 백엔드. ServiceMap, S3 Tiering, SSO를 지원하는 설치형 관측성 플랫폼.**

[English](README.md) | [한국어](README.ko.md)

[![Project Status: WIP](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Wondermove-Inc/k-o11y-server/blob/main/packages/signoz/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/Wondermove-Inc/k-o11y-server?style=social)](https://github.com/Wondermove-Inc/k-o11y-server/stargazers)
[![Release](https://img.shields.io/github/v/release/Wondermove-Inc/k-o11y-server?label=release)](https://github.com/Wondermove-Inc/k-o11y-server/releases)

[ClickHouse](https://clickhouse.com/) · [OpenTelemetry](https://opentelemetry.io/) 에코시스템 기반

</div>

---

## ✨ 주요 기능

- 🗺️ **ServiceMap** — 마이크로서비스 의존성 토폴로지 시각화 + 배치 처리
- 💾 **S3 3-Tier 스토리지** — Hot(EBS) → Warm(S3 Standard) → Cold(S3 Glacier IR) 자동 티어링
- 🔐 **SSO 테넌트 자동 락** — JWT 기반 멀티 테넌트 SSO + 자동 워크스페이스 바인딩
- 🔍 **분산 트레이싱** — ClickHouse 기반 트레이스 저장 및 조회
- 📊 **메트릭 모니터링** — Prometheus 호환 메트릭 수집 및 대시보드
- 📜 **로그 관리** — 구조화된 로그 수집 및 검색
- 🔔 **알림** — AlertManager 기반 알림 규칙 및 채널 관리
- 🔒 **AES-256-GCM 암호화** — S3 인증정보 및 민감 설정 암호화 저장

---

## 🏗️ 아키텍처

K-O11y Server는 두 개의 패키지로 구성된 **모노레포 백엔드**입니다. Go 기반 `core` API(ServiceMap 배치 프로세서, S3 Tiering API)와 `packages/signoz` 디렉토리가 React 프론트엔드와 Query Service를 담당하며, 공통 ClickHouse 클러스터를 공유합니다.

```mermaid
flowchart TB
    subgraph Clients["클라이언트 계층"]
        User[👤 사용자 / 운영자]
        OTel[OTel Collector<br/>Agent 클러스터에서 수신]
    end

    subgraph Server["K-O11y Server (본 레포)"]
        subgraph CorePkg["packages/core (Go 1.24)"]
            CoreAPI[Core HTTP API<br/>Gin Router]
            BatchProc[ServiceMap<br/>배치 프로세서]
            CoreRepo[ClickHouse<br/>Repository]
        end

        subgraph SigNozPkg["packages/signoz"]
            Frontend[Frontend UI<br/>React 18 + Ant Design]
            QueryService[Query Service<br/>+ S3 Tiering + SSO]
            Crypto[AES-256-GCM<br/>암호화 모듈]
            K8sJob[K8s Job Manager]
        end

        Frontend --> QueryService
        Frontend --> CoreAPI
        CoreAPI --> BatchProc
        CoreAPI --> CoreRepo
        QueryService --> Crypto
        QueryService --> K8sJob
    end

    subgraph Storage["스토리지 계층 (외부)"]
        CH[(ClickHouse<br/>전용 VM)]
        S3[(S3 티어링<br/>Hot / Warm / Cold)]
    end

    User --> Frontend
    OTel -->|Gateway 경유<br/>OTLP 수집| CH
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

**데이터 흐름:**

1. **수집(Ingest)** — Agent 클러스터의 OTel Collector가 OTel Gateway를 경유하여 ClickHouse로 원격측정 데이터를 전송
2. **배치(Batch)** — `packages/core`가 주기적으로 트레이스 스팬 기반 ServiceMap 토폴로지 데이터를 생성
3. **조회(Query)** — `packages/signoz` Query Service가 UI(메트릭/로그/트레이스)에 데이터를 제공하고 SSO + S3 티어링 설정을 관리
4. **시각화(Visualize)** — Frontend가 Core API(ServiceMap)와 표준 Query Service(메트릭/로그/트레이스 뷰)를 동시에 소비

### Core 백엔드 레이어

```
Handler (HTTP 엔드포인트, Gin Router)
    ↓
Service (비즈니스 로직, 토폴로지 빌드)
    ↓
Repository (ClickHouse 쿼리, 데이터 조회)
    ↓
Infrastructure (DB 연결 관리)
```

---

## 🚀 빠른 시작

> **더 큰 시스템의 일부입니다.** 전체 프로덕션 배포(Host + Agent 클러스터, ClickHouse VM, OTel Gateway)는 umbrella 레포에서 확인하세요: [Wondermove-Inc/k-o11y](https://github.com/Wondermove-Inc/k-o11y).
>
> 본 README는 서버 컴포넌트의 **로컬 개발 및 빌드**만 다룹니다.

### 사전 요구사항

| 도구 | 최소 버전 | 용도 |
|------|----------|------|
| Go | 1.24.0 | Core 백엔드 빌드 |
| Node.js | 16.15.0 | 프론트엔드 빌드 |
| Docker | 20.10+ | 이미지 빌드 및 푸시 |
| kubectl | 1.25+ | K8s 클러스터 관리 |
| make | - | 빌드 자동화 |

별도로 실행 중인 ClickHouse 인스턴스가 필요합니다 (VM 구축은 [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) 참고).

### Core API 로컬 실행

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

배치 관련 환경변수는 모두 기본값이 설정되어 있어 별도 설정 없이 실행 가능합니다.
ServiceMap 배치를 비활성화하려면 `BATCH_SERVICEMAP_ENABLED=false`로 설정하세요.

### 백엔드 로컬 실행 (Community 빌드)

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

**환경변수 우선순위(높은 순):** make CLI 인자 → shell export → `.env.local` → 기본값

```bash
# Example: pass DSN directly via make
make go-run-community SIGNOZ_CLICKHOUSE_DSN=tcp://default:'pass'@host:9000
```

### Frontend 실행

```bash
cd packages/signoz/frontend
CI=1 yarn install
yarn dev
```

### Swagger API 문서

Core 서버 실행 후 브라우저에서 접속:

```
http://localhost:3001/swagger-ui/
```

---

## 📦 패키지

K-O11y Server는 두 개 패키지로 구성된 모노레포입니다.

| 패키지 | 역할 | 기술 |
|--------|------|------|
| [`packages/core`](packages/core) | ServiceMap API, 배치 프로세서, S3 Tiering 헬퍼 | Go 1.24 + Gin + ClickHouse |
| [`packages/signoz`](packages/signoz) | Backend (React UI + Query Service) | React 18 + Go + Webpack 5 |

### 프로젝트 구조

```
k-o11y-server/
├── packages/
│   ├── core/                        # Go 백엔드 (ServiceMap, S3 Tiering API)
│   │   ├── cmd/main.go              # 진입점
│   │   ├── internal/
│   │   │   ├── batch/               # ServiceMap 배치 프로세서
│   │   │   ├── config/              # 환경변수 기반 설정
│   │   │   ├── domain/servicemap/   # 도메인 모델/DTO
│   │   │   ├── handler/             # HTTP 핸들러 + Gin Router
│   │   │   ├── service/             # 비즈니스 로직
│   │   │   ├── repository/          # ClickHouse 데이터 접근
│   │   │   ├── infrastructure/      # DB 연결 관리
│   │   │   └── utils/               # 유틸리티
│   │   ├── pkg/                     # 공유 패키지 (logger, response, errors)
│   │   ├── deployments/Dockerfile   # 멀티스테이지 빌드
│   │   └── Makefile
│   │
│   └── signoz/                      # Backend UI + Query Service
│       ├── frontend/                # React 프론트엔드
│       ├── pkg/                     # Go 백엔드 패키지
│       │   ├── crypto/              # AES-256-GCM 암호화 (S3 인증정보)
│       │   ├── http/middleware/     # SSO + Tenant Auto-Lock
│       │   ├── k8s/                 # K8s Job 관리
│       │   └── query-service/       # ClickHouse 쿼리 + S3 티어링
│       ├── cmd/community/           # Community 빌드 진입점
│       └── Makefile
│
├── Makefile                         # 루트 빌드 (대화형 build-and-push)
├── NOTICE                           # 출처 표기
└── README.md
```

### 기술 스택

**Core 패키지**

| 구분 | 기술 | 버전 |
|------|------|------|
| 언어 | Go | 1.24 |
| HTTP 프레임워크 | Gin | 1.11 |
| 데이터베이스 | ClickHouse (Native API) | - |
| 메트릭 | Prometheus client_golang | 1.23 |
| 로깅 | Uber Zap + Lumberjack | 1.27 |
| API 문서 | Swagger (swaggo) | - |
| 컨테이너 | distroless/base-debian12 | - |

**Hub 패키지**

| 구분 | 기술 | 버전 |
|------|------|------|
| 프론트엔드 | React + TypeScript | 18.2 / 4.0 |
| 번들러 | Webpack | 5.94 |
| UI 라이브러리 | Ant Design | - |
| 상태 관리 | Redux + React Query | - |
| 백엔드 | Go (gorilla/mux) | 1.24 |

---

## 🛠️ 빌드

> **사전 빌드된 Docker 이미지와 Helm 차트는 제공되지 않습니다.**
> 소스에서 직접 빌드하여 자체 레지스트리(GHCR, Harbor, Docker Hub 등)에 푸시해야 합니다.

### Docker 이미지

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

### 대화형 빌드 (루트)

```bash
make build-and-push
# Choose:
#   1. core   → build and push packages/core
#   2. hub    → build and push packages/signoz
# Enter TAG (e.g. 0.1.3)
```

### 패키지별 개별 빌드

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

### Helm 차트

Helm 차트는 별도의 [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) 레포에 있습니다. 패키징 및 푸시:

```bash
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

배포 전 `values.yaml`의 이미지 레지스트리를 자체 레지스트리 주소로 변경하세요.

### 이미지 레퍼런스

| 패키지 | 레지스트리 경로 | 베이스 이미지 |
|--------|-----------------|--------------|
| core | `<YOUR_REGISTRY>/observability/core` | distroless/base-debian12 |
| hub | `<YOUR_REGISTRY>/observability/hub` | - |

---

## 🔌 API 엔드포인트

Base URL: `http://<host>:3001/api/v1`

| Method | Path | 설명 |
|--------|------|------|
| POST | `/servicemap/topology` | ServiceMap 토폴로지 조회 |
| POST | `/servicemap/workload/details` | 워크로드 상세 정보 |
| POST | `/servicemap/workload/hover-info` | 워크로드 호버 정보 (Top 5) |
| POST | `/servicemap/edge/trace/details` | Edge(연결) 트레이스 상세 |

모든 API는 대용량 필터 페이로드 처리 및 URL 인코딩 이슈 방지를 위해 POST 방식을 사용합니다.

---

## ⚙️ 환경 변수

### Core 패키지 (`packages/core`)

**서버 설정**

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `APP_PORT` | `3001` | | HTTP 서버 포트 |
| `APP_ENV` | `local` | | 환경 (local / dev / stg / prod) |

**ClickHouse 연결**

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `CLICKHOUSE_HOST` | - | **필수** | ClickHouse 호스트 주소 |
| `CLICKHOUSE_PORT` | - | **필수** | ClickHouse Native 포트 (일반적으로 9000) |
| `CLICKHOUSE_DATABASE` | - | **필수** | 데이터베이스명 (일반적으로 `signoz_traces`) |
| `CLICKHOUSE_USER` | - | | 사용자명 |
| `CLICKHOUSE_PASSWORD` | - | | 비밀번호 |
| `CLICKHOUSE_TIMEOUT` | `10s` | | 연결 타임아웃 |
| `CLICKHOUSE_MAX_RETRIES` | `3` | | 최대 재시도 횟수 |

**배치 처리**

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `BATCH_SERVICEMAP_ENABLED` | `true` | | ServiceMap 배치 활성화 |
| `BATCH_SERVICEMAP_INTERVAL` | `20s` | | 배치 실행 주기 (enabled=true일 때 > 0 필수) |
| `BATCH_INSERT_TIMEOUT` | `120s` | | INSERT 쿼리 타임아웃 |
| `BATCH_SAFETY_BUFFER` | `20s` | | 데이터 안정화 대기 시간 |
| `BATCH_MAX_WINDOW` | `30s` | | 단일 배치 최대 처리 윈도우 |

배치 설정은 모두 기본값이 있어 별도 설정 없이 실행 가능합니다.

**로깅**

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `LOG_LEVEL` | `info` | | 로그 레벨 (debug / info / warn / error) |
| `LOG_FILE` | `./logs/local-ko11y.log` | | 로그 파일 경로 |

### Hub 패키지 (`packages/signoz`)

환경변수는 `.env.local` 파일, make CLI 인자, 또는 환경변수 export로 주입합니다.
설정 템플릿은 `.env.example`을 참조하세요.

**설정 가능한 변수 (Makefile `?=` 선언)**

| Make 변수 | 기본값 | 설명 |
|-----------|--------|------|
| `SIGNOZ_CLICKHOUSE_DSN` | `tcp://127.0.0.1:9000` | ClickHouse DSN (`tcp://user:pass@host:port`) |
| `SIGNOZ_CLICKHOUSE_CLUSTER` | `cluster` | ClickHouse 클러스터명 |
| `SIGNOZ_JWT_SECRET` | `secret` | JWT 인증 시크릿 |
| `SIGNOZ_LOG_LEVEL` | `debug` | 로그 레벨 |
| `SIGNOZ_SMTP_FROM` | (빈 값) | 발신 이메일 주소 |
| `SIGNOZ_SMTP_HELLO` | (빈 값) | SMTP HELO 도메인 |
| `SIGNOZ_SMTP_SMARTHOST` | (빈 값) | SMTP 서버 (`host:port`) |
| `SIGNOZ_SMTP_USERNAME` | (빈 값) | SMTP 인증 사용자 |
| `SIGNOZ_SMTP_PASSWORD` | (빈 값) | SMTP 인증 비밀번호 |
| `SIGNOZ_SMTP_REQUIRE_TLS` | `true` | SMTP TLS 요구 여부 |

**Makefile 자동 설정 (변경 불필요)**

| 환경변수 | 값 | 설명 |
|----------|----|------|
| `SIGNOZ_TELEMETRYSTORE_PROVIDER` | `clickhouse` | 텔레메트리 저장소 |
| `SIGNOZ_ALERTMANAGER_PROVIDER` | `signoz` | AlertManager 프로바이더 |
| `SIGNOZ_SQLSTORE_SQLITE_PATH` | `signoz.db` | SQLite 경로 |
| `SIGNOZ_WEB_ENABLED` | `false` | 웹 UI (프론트엔드 별도 실행) |

---

## 🔗 관련 저장소

K-O11y Server는 더 큰 관측성 플랫폼의 한 구성 요소입니다. 전체 그림은 umbrella 레포에서 확인하세요.

| 저장소 | 설명 |
|--------|------|
| [k-o11y](https://github.com/Wondermove-Inc/k-o11y) | 🌂 Umbrella — 전체 개요, 아키텍처, 6단계 설치 가이드 |
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm 차트, Go CLI 설치 도구, ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | CRD Processor를 갖춘 커스텀 OTel Collector |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | License Guard를 갖춘 OTel Collector |

---

## 🤝 기여하기

기여는 언제나 환영합니다. 특히 [good first issue](https://github.com/Wondermove-Inc/k-o11y-server/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) 라벨이 붙은 이슈부터 시작해보세요.

1. **이슈 찾기** — `good first issue` 또는 `help wanted` 라벨 확인
2. **이슈에 댓글** — 작업 의사를 밝혀 중복 작업을 피합니다
3. **Fork → branch → PR** — 범위는 좁게, 설명은 명확하게
4. **리뷰 반영** — 메인테이너가 수 일 이내 답변합니다

자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참고하세요.

본 프로젝트는 **passive maintenance** 모델입니다 — PR과 이슈는 시간이 허락하는 대로 검토됩니다. 7일 내 응답을 목표로 하지만 보장하지는 않습니다.

---

## 📄 라이선스

MIT License — [LICENSE](packages/signoz/LICENSE) 참조

이 프로젝트는 [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.)를 기반으로 합니다.
전체 저작권 정보는 [NOTICE](NOTICE) 파일을 참고하세요.

---

## 💬 연락처

- 🐛 **버그 리포트 & 기능 요청**: [GitHub Issues](https://github.com/Wondermove-Inc/k-o11y-server/issues)
- 💭 **질문 & 토론**: 이슈로 문의해주세요 (GitHub Discussions은 곧 활성화 예정)
- 🌐 **웹사이트**: [wondermove.net](https://wondermove.net)

---

<div align="center">

**[Wondermove](https://wondermove.net)가 개발 및 관리합니다**

</div>
