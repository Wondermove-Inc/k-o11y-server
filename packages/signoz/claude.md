# SigNoz 로컬 개발 환경 실행 가이드

## 📋 필수 도구

- **Git** - [git-scm.com](https://git-scm.com/)
- **Go** - [go.dev/dl](https://go.dev/dl/) (버전: [go.mod](go.mod#L3) 참조)
- **Node.js** - [nodejs.org](https://nodejs.org) (버전: [frontend/.nvmrc](frontend/.nvmrc) 참조)
- **Yarn** - [yarnpkg.com](https://yarnpkg.com/getting-started/install)
- **Docker** - [docs.docker.com/get-docker](https://docs.docker.com/get-docker/)

## 🚀 실행 순서

### 1단계: 개발 환경 시작 (ClickHouse + OTel Collector)

프로젝트 루트 디렉토리에서 실행:

```bash
# ClickHouse와 OTel Collector 한번에 시작
make devenv-up
```

**또는 개별 실행:**
```bash
# ClickHouse만 시작
make devenv-clickhouse

# OTel Collector만 시작
make devenv-signoz-otel-collector
```

**실행 결과:**
- ClickHouse: http://localhost:8123
- OTel Collector: grpc://localhost:4317, http://localhost:4318

### 2단계: 백엔드 서버 실행

새 터미널 창을 열고:

```bash
# 프로젝트 루트에서 실행
make go-run-community
```

**실행 결과:**
- 백엔드 API 서버: http://localhost:8080

**검증:**
```bash
curl http://localhost:8080/api/v1/health
# 응답: {"status":"ok"}
```

### 3단계: 프론트엔드 실행

새 터미널 창을 열고:

```bash
# frontend 디렉토리로 이동
cd frontend

# .env 파일 생성 (처음 한 번만)
echo "FRONTEND_API_ENDPOINT=http://localhost:8080" > .env

# 의존성 설치 (처음 한 번만)
# Git SSH 이슈 우회하면서 설치
GIT_CONFIG_COUNT=1 \
GIT_CONFIG_KEY_0="url.https://github.com/.insteadOf" \
GIT_CONFIG_VALUE_0="ssh://git@github.com/" \
yarn install

# 개발 서버 시작
yarn dev
```

**실행 결과:**
- 프론트엔드: http://localhost:3301

## ✅ 전체 서비스 확인

```bash
# ClickHouse
curl http://localhost:8123/ping
# 응답: Ok.

# OTel Collector
curl http://localhost:13133

# Backend
curl http://localhost:8080/api/v1/health
# 응답: {"status":"ok"}

# Frontend
# 브라우저에서 http://localhost:3301 열기
```

## 🛠️ 유용한 명령어

```bash
# 사용 가능한 모든 make 명령어 보기
make help

# Go 테스트 실행
make go-test

# Frontend 테스트 실행
cd frontend && yarn test
```

## 🐛 문제 해결

### Yarn 502 에러 발생 시
```bash
cd frontend
yarn cache clean
yarn install
```

### npm 의존성 충돌 시 (대안)
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install --legacy-peer-deps
npm run dev
```

### Git SSH 권한 에러 시
위의 3단계에 나온 GIT_CONFIG 환경변수 방식 사용

## 📊 테스트 데이터 전송

로컬 SigNoz에 텔레메트리 데이터 전송:
- OTLP gRPC: `localhost:4317`
- OTLP HTTP: `localhost:4318`

**예시: curl로 테스트 trace 전송**
```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"test-service"}}]},"scopeSpans":[{"spans":[{"traceId":"12345678901234567890123456789012","spanId":"1234567890123456","name":"test-span","startTimeUnixNano":"1609459200000000000","endTimeUnixNano":"1609459201000000000"}]}]}]}'
```

## 🎯 요약

**실행 순서:**
1. `make devenv-up` - 인프라 시작
2. `make go-run-community` - 백엔드 시작
3. `cd frontend && yarn dev` - 프론트엔드 시작
4. 브라우저에서 http://localhost:3301 접속

**포트 정리:**
- 3301: Frontend
- 8080: Backend API
- 8123: ClickHouse HTTP
- 9000: ClickHouse Native
- 4317: OTel Collector (gRPC)
- 4318: OTel Collector (HTTP)
