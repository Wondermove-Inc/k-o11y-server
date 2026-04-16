<h1 align="center" style="border-bottom: none">
    <br>K-O11y
</h1>

<p align="center">
Kubernetes 클러스터 모니터링 및 관찰성(Observability) 플랫폼
</p>

<p align="center">
메트릭, 로그, 트레이스 통합 모니터링
</p>

---

## 📋 개요

**K-O11y**는 SigNoz를 기반으로 한 Kubernetes 클러스터 관찰성 플랫폼입니다. K-O11y 생태계와 통합되어 클러스터의 메트릭, 로그, 트레이스를 실시간으로 수집하고 시각화합니다.

### 주요 기능

- 📊 **메트릭 모니터링**: Kubernetes 리소스 사용량 및 성능 메트릭
- 📝 **로그 분석**: 중앙 집중식 로그 수집 및 검색
- 🔍 **분산 트레이싱**: 마이크로서비스 간 요청 추적
- 📈 **대시보드**: 커스터마이징 가능한 실시간 대시보드
- 🔔 **알림**: 메트릭 기반 알림 및 알림 규칙

## 🚀 빠른 시작

### 필수 도구

- **Git** - [git-scm.com](https://git-scm.com/)
- **Go** - [go.dev/dl](https://go.dev/dl/) (버전: [go.mod](go.mod#L3) 참조)
- **Node.js** - [nodejs.org](https://nodejs.org) (버전: [frontend/.nvmrc](frontend/.nvmrc) 참조)
- **Yarn** - [yarnpkg.com](https://yarnpkg.com/getting-started/install)
- **Docker** - [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)

### 실행 순서

#### 1. 환경변수 설정

```bash
# .env.example을 복사하여 .env.local 생성 (최초 1회)
cp .env.example .env.local

# .env.local을 열어 실제 값으로 수정
# 필수: SIGNOZ_CLICKHOUSE_DSN (ClickHouse 접속 정보)
# 선택: SIGNOZ_SMTP_* (메일 발송 기능 필요 시)
```

> `.env.local`은 `.gitignore`에 포함되어 있어 git에 커밋되지 않습니다.

#### 2. 백엔드 서버 시작

```bash
# .env.local 기반 실행 (권장)
make go-run-community

# 또는 make CLI로 직접 전달
make go-run-community SIGNOZ_CLICKHOUSE_DSN=tcp://default:pass@host:9000

# 또는 환경변수 export 후 실행
export SIGNOZ_CLICKHOUSE_DSN=tcp://default:pass@host:9000
make go-run-community
```

**우선순위**: make CLI 인자 > env export > .env.local > 기본값

**실행 결과:**
- 백엔드 API 서버: http://localhost:8080

**검증:**
```bash
curl http://localhost:8080/api/v1/health
# 응답: {"status":"ok"}
```

#### 3. 프론트엔드 시작

새 터미널 창을 열고:

```bash
cd frontend

# .env 파일 생성 (처음 한 번만)
echo "FRONTEND_API_ENDPOINT=http://localhost:8080" > .env

# 의존성 설치 (처음 한 번만)
GIT_CONFIG_COUNT=1 \
GIT_CONFIG_KEY_0="url.https://github.com/.insteadOf" \
GIT_CONFIG_VALUE_0="ssh://git@github.com/" \
CI=1 yarn install

# 개발 서버 시작
yarn dev
```

**실행 결과:**
- 프론트엔드: http://localhost:3301

#### 4. K-O11y Core (구 mgmt BE) 연결

새 터미널에서 o11y-core 백엔드를 실행합니다:

```bash
# ko11y core 디렉토리로 이동
cd ~/skuberplus-observability/packages/core

# 백엔드 실행
APP_ENV=local \
APP_PORT=3001 \
LOG_LEVEL=info \
LOG_FILE=./logs/local-ko11y.log \
CLICKHOUSE_HOST=<your-clickhouse-host> \
CLICKHOUSE_PORT=9000 \
CLICKHOUSE_DATABASE=signoz_traces \
CLICKHOUSE_USER=default \
CLICKHOUSE_PASSWORD=<your-password> \
CLICKHOUSE_TIMEOUT=10s \
CLICKHOUSE_MAX_RETRIES=3 \
go run cmd/main.go
```
자세한 내용은 core README.md 파일을 참조해주세요.

#### 5. ClickHouse 포트포워딩

ClickHouse가 Kubernetes 클러스터에서 실행 중인 경우:

```bash
kubectl port-forward -n platform svc/my-release-clickhouse 8123:8123 9000:9000
```

#### 6. 브라우저 접속

http://localhost:3301 에 접속하여 K-O11y를 사용합니다.

## 📊 포트 정리

| 서비스 | 포트 | 설명 |
|--------|------|------|
| Frontend | 3301 | 프론트엔드 UI |
| Backend API | 8080 | SigNoz 백엔드 API |
| ClickHouse HTTP | 8123 | ClickHouse HTTP 인터페이스 |
| ClickHouse Native | 9000 | ClickHouse Native 프로토콜 |
| OTel Collector (gRPC) | 4317 | OpenTelemetry gRPC |
| OTel Collector (HTTP) | 4318 | OpenTelemetry HTTP |

## 🏗️ 아키텍처

```
┌─────────────────┐
│   Frontend UI   │ :3301
│   (React)       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Backend API    │ :8080
│  (Go)           │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│   ClickHouse    │◄─────┤ K-O11y Core  │
│   (Database)    │      │  (mgmt BE)   │
└─────────────────┘      └──────────────┘
         ▲
         │
┌─────────────────┐
│ OTel Collector  │ :4317, :4318
│                 │
└─────────────────┘
```

## 🛠️ 개발 가이드

### 백엔드 개발

```bash
# 테스트 실행
make go-test

# 빌드
make build

# 사용 가능한 모든 명령어 보기
make help
```

### 프론트엔드 개발

```bash
cd frontend

# 테스트 실행
yarn test

# 빌드
yarn build

# 린트
yarn lint
```

## Docker 이미지 빌드

### 방법 1: 빌드 + 푸시 한번에 (권장)

```bash
cd ~/skuberplus-observability/packages/signoz

# 이미지 빌드 및 레지스트리 푸시
make o11y-build-and-push TAG=0.1.20
```

**특징**:
- 이미지 빌드와 레지스트리 푸시를 하나의 명령어로 수행
- TAG 파라미터 필수 (예: `TAG=0.1.20`)
- push 전 `docker login <YOUR_REGISTRY>` 필요

### 방법 2: 수동 빌드

```bash
cd ~/skuberplus-observability/packages/signoz

DOCKER_IMG_BUILD=true \
VERSION={tag} \
DOCKER_REGISTRY_COMMUNITY=<YOUR_REGISTRY>/observability/hub \
OS=linux \
ARCHS=amd64 \
make docker-build-community-amd64
```

**참고사항**:
- (필수) DOCKER_IMG_BUILD = true 인 경우, .env 파일을 사용한 오버라이드를 방지합니다.
- VERSION을 태그로 쓰고, DOCKER_REGISTRY_COMMUNITY가 리포지토리 이름입니다.
- push 전 docker login <YOUR_REGISTRY> 필요합니다.
- amd64만 쓰면 make docker-build-community-amd64로 충분합니다.



## 🐛 문제 해결

### Yarn 502 에러 발생 시
```bash
cd frontend
yarn cache clean
yarn install
```

### Git SSH 권한 에러 시
프론트엔드 설치 시 GIT_CONFIG 환경변수 사용 (위 2단계 참조)

### ClickHouse 연결 실패 시
1. Kubernetes 클러스터가 실행 중인지 확인
2. ClickHouse 서비스가 배포되어 있는지 확인:
   ```bash
   kubectl get svc -n platform | grep clickhouse
   ```
3. 포트포워딩이 활성화되어 있는지 확인

### 백엔드 health check 실패 시
```bash
# 백엔드 로그 확인
# 8080 포트가 이미 사용 중인지 확인
lsof -i :8080
```


## 📁 프로젝트 구조

```
packages/signoz/
├── cmd/                    # 백엔드 엔트리포인트
├── pkg/                    # Go 백엔드 패키지
├── frontend/               # React 프론트엔드
│   ├── src/
│   │   ├── components/    # React 컴포넌트
│   │   ├── pages/         # 페이지 컴포넌트
│   │   └── hooks/         # 커스텀 훅
│   └── public/
├── deploy/                 # 배포 설정
├── Makefile               # 빌드 스크립트
└── README.md
```

## 📝 라이선스

이 프로젝트는 SigNoz를 기반으로 합니다. 자세한 내용은 [LICENSE](LICENSE)를 참조하세요.