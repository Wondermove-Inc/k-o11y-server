## K-O11y Core
Go 기반 K-O11y 백엔드. ClickHouse를 데이터 소스로 사용하며 REST API를 제공한다.

## 실행 전 요구사항
- Go 1.24+
- 외부 ClickHouse 접근 가능 (예: <YOUR_IP>:9000)
- 필요한 환경변수 설정 (아래 참조)

## 환경변수 설정
프로젝트 루트의 `env.template` 파일을 참조하여 환경변수를 설정하세요.

### 방법 1: direnv 사용 (권장)
```bash
# direnv 설치 (macOS)
brew install direnv

# .envrc 파일 생성
cp env.template .envrc

# 실제 값 입력 후 허용
direnv allow
```

### 방법 2: 환경변수 직접 설정
```bash
export APP_ENV=local
export APP_PORT=3001
export BATCH_SERVICEMAP_ENABLED=true
export BATCH_SERVICEMAP_INTERVAL=20s
# ... (env.template 참조)
```

## 로컬 실행
```bash
cd ~/skuberplus/skuberplus-observability/packages/core

APP_ENV=local \
APP_PORT=3001 \
LOG_LEVEL=info \
LOG_FILE=./logs/local-ko11y.log \
CLICKHOUSE_HOST=<YOUR_IP> \
CLICKHOUSE_PORT=9000 \
CLICKHOUSE_DATABASE=signoz_traces \
CLICKHOUSE_USER=default \
CLICKHOUSE_PASSWORD=<CLICKHOUSE_PASSWORD> \
CLICKHOUSE_TIMEOUT=10s \
CLICKHOUSE_MAX_RETRIES=3 \
BATCH_SERVICEMAP_ENABLED=true \
BATCH_SERVICEMAP_INTERVAL=20s \
go run cmd/main.go
```

## 설정 키
- `APP_ENV`: 실행 환경 (local/dev/stg/prod 등)
- `APP_PORT`: HTTP 포트
- `LOG_LEVEL`, `LOG_FILE`: 로깅 레벨/경로
- `CLICKHOUSE_*`: ClickHouse 접속 정보
- `BATCH_SERVICEMAP_ENABLED`: 배치 네트워크 처리 활성화 (true/false, 기본값: false)
- `BATCH_SERVICEMAP_INTERVAL`: 배치 처리 주기 (예: 20s, 1m, 기본값: 20s)

## 참고
- `.vscode/launch.json`에 동일한 환경변수 예시가 있으므로 VS Code 디버그 실행 시 그대로 사용 가능.
- K8s 배포 시에는 동일 키를 ConfigMap 주입하는 방향을 권장.


## Docker 이미지 빌드

### 방법 1: 빌드 + 푸시 한번에 (권장)

```bash
cd ~/skuberplus/skuberplus-observability/packages/core

# 이미지 빌드 및 레지스트리 푸시
make core-build-and-push TAG=0.1.3
```

**특징**:
- 이미지 빌드와 레지스트리 푸시를 하나의 명령어로 수행
- TAG 파라미터 필수 (예: `TAG=0.1.3`)
- push 전 `docker login <YOUR_REGISTRY>` 필요

### 방법 2: 수동 빌드

```bash
cd ~/skuberplus/skuberplus-observability/packages/core

docker build -t <YOUR_REGISTRY>/observability/core:{tag} -f deployments/Dockerfile --platform linux/amd64 .

docker push <YOUR_REGISTRY>/observability/core:{tag}
```
