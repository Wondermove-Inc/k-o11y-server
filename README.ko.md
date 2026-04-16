# K-O11y

[English](README.md)


[Wondermove](https://wondermove.net)가 개발한 설치형 Kubernetes 관측성 솔루션입니다.
[SigNoz](https://github.com/SigNoz/signoz) (MIT License)를 기반으로 S3 Tiering, SSO, ServiceMap 등을 확장했습니다.

---

## 목차

- [프로젝트 개요](#프로젝트-개요)
- [아키텍처](#아키텍처)
- [프로젝트 구조](#프로젝트-구조)
- [기술 스택](#기술-스택)
- [개발 환경 설정](#개발-환경-설정)
- [빌드 및 배포](#빌드-및-배포)
- [API 엔드포인트](#api-엔드포인트)
- [환경 변수](#환경-변수)

---

## 프로젝트 개요

K-O11y는 설치형 Kubernetes 관측성 솔루션입니다. SigNoz를 기반으로 S3 Tiering, SSO, ServiceMap 등을 확장했습니다.

### 주요 기능

- **ServiceMap**: 마이크로서비스 간 의존관계 토폴로지 시각화
- **분산 트레이싱**: ClickHouse 기반 트레이스 저장 및 조회
- **메트릭 모니터링**: Prometheus 호환 메트릭 수집 및 대시보드
- **로그 관리**: 구조화된 로그 수집 및 검색
- **알림**: AlertManager 기반 알림 규칙 및 채널 관리

### 제품 포지셔닝

```
Skuber Client (Freelens 기반)
├── 무료: Kubernetes Management, AI 기능
└── K-O11y (오픈소스)
```

---

## 아키텍처

### 2-Tier 분산 관측성 구조

```
┌─────────────────────────────────────────────────────────┐
│  Agent Clusters (다수)                                   │
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
│  Host Cluster (중앙)                                     │
│  ┌──────────────────┐                   ┌─────────────┐  │
│  │ SigNoz OTel      │                   │  SigNoz UI  │  │
│  │ Collector        │──┐            ┌───│ + Core API  │  │
│  └──────────────────┘  │            │   └─────────────┘  │
└────────────────────────┼────────────┼────────────────────┘
                         │            │
                         ▼            │
                   ┌────────────┐     │
                   │ ClickHouse │─────┘
                   │ (별도 VM)  │
                   └────────────┘
```

### 모노레포 패키지 구조

| 패키지 | 역할 | 기술 |
|--------|------|------|
| `packages/core` | ServiceMap API, 배치 처리 | Go 1.24 + Gin + ClickHouse |
| `packages/signoz` | SigNoz 포크 (UI + Query Service) | React 18 + Go |

### Core 백엔드 레이어 구조

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

## 프로젝트 구조

```
k-o11y/
├── packages/
│   ├── core/                        # Go 백엔드 (ServiceMap)
│   │   ├── cmd/main.go              # 진입점
│   │   ├── internal/
│   │   │   ├── batch/               # ServiceMap 배치 프로세서
│   │   │   │   ├── processor.go
│   │   │   │   ├── metrics.go
│   │   │   │   └── sql/             # 배치 SQL 파일
│   │   │   ├── config/              # 환경변수 기반 설정
│   │   │   ├── domain/servicemap/   # 도메인 모델/DTO
│   │   │   ├── handler/             # HTTP 핸들러 + 라우터
│   │   │   ├── service/             # 비즈니스 로직
│   │   │   ├── repository/          # ClickHouse 데이터 접근
│   │   │   ├── infrastructure/      # DB 연결 관리
│   │   │   └── utils/               # 유틸리티
│   │   ├── pkg/                     # 공유 패키지 (logger, response, errors)
│   │   ├── deployments/
│   │   │   └── Dockerfile           # 멀티스테이지 빌드
│   │   ├── go.mod
│   │   └── Makefile
│   │
│   └── signoz/                      # SigNoz 포크
│       ├── frontend/                # React 프론트엔드
│       │   ├── src/
│       │   └── package.json
│       ├── pkg/                     # Go 백엔드 패키지
│       ├── cmd/community/           # Community 빌드 진입점
│       └── Makefile
│
├── Makefile                         # 루트 빌드
├── NOTICE                           # 출처 표기
└── README.md                        # 프로젝트 문서
```

---

## 기술 스택

### Core 패키지

| 구분 | 기술 | 버전 |
|------|------|------|
| 언어 | Go | 1.24 |
| HTTP 프레임워크 | Gin | 1.11 |
| 데이터베이스 | ClickHouse (Native API) | - |
| 메트릭 | Prometheus client_golang | 1.23 |
| 로깅 | Uber Zap + Lumberjack | 1.27 |
| API 문서 | Swagger (swaggo) | - |
| 컨테이너 | distroless/base-debian12 | - |

### SigNoz 패키지

| 구분 | 기술 | 버전 |
|------|------|------|
| 프론트엔드 | React + TypeScript | 18.2 / 4.0 |
| 번들러 | Webpack | 5.94 |
| UI 라이브러리 | Ant Design | - |
| 상태 관리 | Redux + React Query | - |
| 백엔드 | Go (gorilla/mux) | 1.24 |

---

## 개발 환경 설정

### 필수 도구

| 도구 | 최소 버전 | 용도 |
|------|----------|------|
| Go | 1.24.0 | Core 백엔드 빌드 |
| Node.js | 16.15.0 | SigNoz 프론트엔드 빌드 |
| Docker | 20.10+ | 이미지 빌드 및 배포 |
| kubectl | 1.25+ | K8s 클러스터 관리 |
| make | - | 빌드 자동화 |

### Core 패키지 로컬 실행

```bash
cd packages/core

# 필수 환경변수 설정
export CLICKHOUSE_HOST=<YOUR_IP>
export CLICKHOUSE_PORT=9000
export CLICKHOUSE_DATABASE=signoz_traces
export CLICKHOUSE_USER=default
export CLICKHOUSE_PASSWORD=<password>

# 선택 환경변수 (기본값이 있으므로 필요 시만 변경)
# export APP_PORT=3001                    # 기본값: 3001
# export APP_ENV=local                    # 기본값: local
# export BATCH_SERVICEMAP_ENABLED=true    # 기본값: true
# export BATCH_SERVICEMAP_INTERVAL=20s    # 기본값: 20s

# 실행
go run cmd/main.go
```

> BATCH 관련 환경변수는 모두 기본값이 설정되어 있어 별도 설정 없이 실행 가능합니다.
> ServiceMap 배치를 비활성화하려면 `BATCH_SERVICEMAP_ENABLED=false`로 설정하세요.

### Hub (SigNoz) 패키지 로컬 실행

```bash
cd packages/signoz

# 1. 환경변수 설정 (.env.local 파일 생성 — 최초 1회)
cp .env.example .env.local
# .env.local 파일을 열어 ClickHouse DSN 등 실제 값으로 수정

# 2. 개발 인프라 기동 (로컬 ClickHouse + OTel Collector)
make devenv-up

# 3. Go 백엔드 실행
make go-run-community
```

> **환경변수 주입 방식** (우선순위: make CLI > env export > .env.local > 기본값):
>
> ```bash
> # 방법 1: .env.local 파일 (권장 — gitignored)
> # 방법 2: make 명령어에 직접 전달
> make go-run-community SIGNOZ_CLICKHOUSE_DSN=tcp://default:'pass'@host:9000
> # 방법 3: 환경변수 export
> export SIGNOZ_CLICKHOUSE_DSN=tcp://default:pass@host:9000
> make go-run-community
> 
> # ex) make go-run-community SIGNOZ_CLICKHOUSE_DSN=tcp://default:'<CLICKHOUSE_PASSWORD>'@<YOUR_HOST>:9000
> ```

```bash
# 프론트엔드 개발 서버
cd frontend
CI=1 yarn install
yarn dev
```

### Swagger API 문서 확인

Core 서버 실행 후 브라우저에서 접속:

```
http://localhost:3001/swagger-ui/
```

---

## 빌드 및 배포

> **사전 빌드된 Docker 이미지와 Helm 차트는 제공되지 않습니다.**
> 소스에서 직접 빌드하고 자체 레지스트리에 push해야 합니다.

### Docker 이미지 빌드

```bash
# Core API
cd packages/core
docker build -t <your-registry>/observability/core:v1.0.0 -f deployments/Dockerfile .
docker push <your-registry>/observability/core:v1.0.0

# SigNoz Hub (community)
cd packages/signoz
make go-build-community
docker build -t <your-registry>/observability/hub:v1.0.0 -f cmd/community/Dockerfile .
docker push <your-registry>/observability/hub:v1.0.0
```

### Helm 차트 패키징

```bash
cd k-o11y-install/charts
helm package k-o11y-host
helm package k-o11y-agent
helm push k-o11y-host-*.tgz oci://<your-registry>/charts
helm push k-o11y-agent-*.tgz oci://<your-registry>/charts
```

배포 전 `values.yaml`의 이미지 레지스트리를 자체 레지스트리로 변경하세요.

### 대화형 빌드 (루트)

```bash
make build-and-push
# 선택:
#   1. core   → packages/core Docker 빌드 및 푸시
#   2. hub    → packages/signoz Docker 빌드 및 푸시
# TAG 버전 입력 (예: 0.1.3)
```

### Core 패키지 개별 빌드

```bash
cd packages/core

# Docker 이미지 빌드 및 푸시
make core-build-and-push TAG=0.1.3
# → <YOUR_REGISTRY>/observability/core:0.1.3

# GitHub Actions 워크플로우 트리거
make trigger-workflow CUSTOM_TAG=v0.1.3
```

### SigNoz 패키지 개별 빌드

```bash
cd packages/signoz

# Docker 이미지 빌드 및 푸시
make o11y-build-and-push TAG=0.1.20
```

### Kubernetes 배포

Helm chart를 사용하여 배포합니다. [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) 레포를 참조하세요.

### Docker 이미지 정보

| 패키지 | 레지스트리 | 베이스 이미지 |
|--------|-----------|-------------|
| core | `<YOUR_REGISTRY>/observability/core` | distroless/base-debian12 |
| signoz | `<YOUR_REGISTRY>/observability/hub` | - |

---

## API 엔드포인트

Base URL: `http://<host>:3001/api/v1`

| Method | Path | 설명 |
|--------|------|------|
| POST | `/servicemap/topology` | ServiceMap 토폴로지 조회 |
| POST | `/servicemap/workload/details` | 워크로드 상세 정보 |
| POST | `/servicemap/workload/hover-info` | 워크로드 호버 정보 (Top 5) |
| POST | `/servicemap/edge/trace/details` | Edge(연결) 트레이스 상세 |

> 모든 API는 POST 방식을 사용합니다 (대용량 필터 및 URL 인코딩 이슈 대응).

---

## 환경 변수

### Core 패키지 (`packages/core`)

#### 서버 설정

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `APP_PORT` | `3001` | | HTTP 서버 포트 |
| `APP_ENV` | `local` | | 환경 (local / dev / stg / prod) |

#### ClickHouse 연결

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `CLICKHOUSE_HOST` | - | **필수** | ClickHouse 호스트 주소 |
| `CLICKHOUSE_PORT` | - | **필수** | ClickHouse Native 포트 (일반적으로 9000) |
| `CLICKHOUSE_DATABASE` | - | **필수** | 데이터베이스명 (일반적으로 signoz_traces) |
| `CLICKHOUSE_USER` | - | | 사용자명 |
| `CLICKHOUSE_PASSWORD` | - | | 비밀번호 |
| `CLICKHOUSE_TIMEOUT` | `10s` | | 연결 타임아웃 |
| `CLICKHOUSE_MAX_RETRIES` | `3` | | 최대 재시도 횟수 |

#### 배치 처리 설정

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `BATCH_SERVICEMAP_ENABLED` | `true` | | ServiceMap 배치 활성화 |
| `BATCH_SERVICEMAP_INTERVAL` | `20s` | | 배치 실행 주기 (enabled=true일 때 > 0 필수) |
| `BATCH_INSERT_TIMEOUT` | `120s` | | INSERT 쿼리 타임아웃 |
| `BATCH_SAFETY_BUFFER` | `20s` | | 데이터 안정화 대기 시간 |
| `BATCH_MAX_WINDOW` | `30s` | | 단일 배치 최대 처리 윈도우 |

> 배치 설정은 모두 기본값이 있어 별도 설정 없이 실행 가능합니다.

#### 로깅 설정

| 변수 | 기본값 | 필수 | 설명 |
|------|--------|------|------|
| `LOG_LEVEL` | `info` | | 로그 레벨 (debug / info / warn / error) |
| `LOG_FILE` | `./logs/local-ko11y.log` | | 로그 파일 경로 |

### Hub (SigNoz) 패키지 (`packages/signoz`)

환경변수는 `.env.local` 파일, make CLI 인자, 또는 환경변수 export로 주입합니다.
설정 템플릿은 `.env.example`을 참조하세요.

#### 설정 가능한 변수 (Makefile `?=` 선언)

| Make 변수 | 기본값 | 설명 |
|-----------|--------|------|
| `SIGNOZ_CLICKHOUSE_DSN` | `tcp://127.0.0.1:9000` | ClickHouse 연결 DSN (`tcp://user:pass@host:port`) |
| `SIGNOZ_CLICKHOUSE_CLUSTER` | `cluster` | ClickHouse 클러스터명 |
| `SIGNOZ_JWT_SECRET` | `secret` | JWT 인증 시크릿 |
| `SIGNOZ_LOG_LEVEL` | `debug` | 로그 레벨 |
| `SIGNOZ_SMTP_FROM` | (빈 값) | 발신 이메일 주소 |
| `SIGNOZ_SMTP_HELLO` | (빈 값) | SMTP HELO 도메인 |
| `SIGNOZ_SMTP_SMARTHOST` | (빈 값) | SMTP 서버 (`host:port`) |
| `SIGNOZ_SMTP_USERNAME` | (빈 값) | SMTP 인증 사용자 |
| `SIGNOZ_SMTP_PASSWORD` | (빈 값) | SMTP 인증 비밀번호 |
| `SIGNOZ_SMTP_REQUIRE_TLS` | `true` | SMTP TLS 요구 여부 |

#### Makefile 자동 설정 (변경 불필요)

| 환경변수 | 값 | 설명 |
|----------|------|------|
| `SIGNOZ_TELEMETRYSTORE_PROVIDER` | `clickhouse` | 텔레메트리 저장소 |
| `SIGNOZ_ALERTMANAGER_PROVIDER` | `signoz` | AlertManager 프로바이더 |
| `SIGNOZ_SQLSTORE_SQLITE_PATH` | `signoz.db` | SQLite 경로 |
| `SIGNOZ_WEB_ENABLED` | `false` | 웹 UI (프론트엔드 별도 실행) |

---

## 관련 저장소

| 저장소 | 설명 |
|--------|------|
| [k-o11y-install](https://github.com/Wondermove-Inc/k-o11y-install) | Helm chart, Go CLI 설치 도구, ClickHouse DDL |
| [k-o11y-otel-collector](https://github.com/Wondermove-Inc/k-o11y-otel-collector) | OTel Collector fork (에이전트, CRD Processor) |
| [k-o11y-otel-gateway](https://github.com/Wondermove-Inc/k-o11y-otel-gateway) | SigNoz OTel Collector fork (게이트웨이, License Guard) |

## 관리

[Wondermove](https://wondermove.net)가 개발 및 관리합니다.

## 라이선스

MIT License - [LICENSE](packages/signoz/LICENSE) 참조

이 프로젝트는 [SigNoz](https://github.com/SigNoz/signoz) (MIT License, Copyright SigNoz Inc.)를 기반으로 합니다.
[NOTICE](NOTICE) 파일에서 전체 저작권 정보를 확인하세요.
