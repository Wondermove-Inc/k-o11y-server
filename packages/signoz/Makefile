##############################################################
# variables
##############################################################
SHELL                   := /bin/bash

# Load local environment overrides (optional, gitignored)
-include .env.local

SRC						?= $(shell pwd)
NAME					?= signoz
OS                      ?= $(shell uname -s | tr '[A-Z]' '[a-z]')
ARCH                    ?= $(shell uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')
COMMIT_SHORT_SHA        ?= $(shell git rev-parse --short HEAD)
BRANCH_NAME             ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))
VERSION                 ?= $(BRANCH_NAME)-$(COMMIT_SHORT_SHA)
TIMESTAMP               ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
ARCHS					?= amd64 arm64
TARGET_DIR              ?= $(shell pwd)/target

GO_BUILD_VERSION_LDFLAGS 		= -X github.com/SigNoz/signoz/pkg/version.version=$(VERSION) -X github.com/SigNoz/signoz/pkg/version.hash=$(COMMIT_SHORT_SHA) -X github.com/SigNoz/signoz/pkg/version.time=$(TIMESTAMP) -X github.com/SigNoz/signoz/pkg/version.branch=$(BRANCH_NAME)
GO_BUILD_ARCHS_COMMUNITY 		= $(addprefix go-build-community-,$(ARCHS))
GO_BUILD_CONTEXT_COMMUNITY 		= $(SRC)/cmd/community
GO_BUILD_LDFLAGS_COMMUNITY 		= $(GO_BUILD_VERSION_LDFLAGS) -X github.com/SigNoz/signoz/pkg/version.variant=community

DOCKER_BUILD_ARCHS_COMMUNITY 	= $(addprefix docker-build-community-,$(ARCHS))
DOCKERFILE_COMMUNITY 			= $(SRC)/cmd/community/Dockerfile
DOCKER_REGISTRY_COMMUNITY 		?= # set your registry (e.g., ghcr.io/your-org/signoz-community)
JS_BUILD_CONTEXT 				= $(SRC)/frontend

##############################################################
# runtime configuration
# Override via: make VAR=value | export VAR=value | .env.local
##############################################################
SIGNOZ_LOG_LEVEL                ?= debug
SIGNOZ_JWT_SECRET               ?= secret
SIGNOZ_CLICKHOUSE_DSN           ?= tcp://127.0.0.1:9000
SIGNOZ_CLICKHOUSE_CLUSTER       ?= cluster
SIGNOZ_SMTP_FROM                ?=
SIGNOZ_SMTP_HELLO               ?=
SIGNOZ_SMTP_SMARTHOST           ?=
SIGNOZ_SMTP_USERNAME            ?=
SIGNOZ_SMTP_PASSWORD            ?=
SIGNOZ_SMTP_REQUIRE_TLS         ?= true
K_O11Y_SSO_ENABLED              ?= false
K_O11Y_JWT_PUBLIC_KEY           ?=
K_O11Y_JWT_ISSUER               ?=
K_O11Y_DEFAULT_ROLE             ?= EDITOR

##############################################################
# directories
##############################################################
$(TARGET_DIR):
	mkdir -p $(TARGET_DIR)

##############################################################
# common commands
##############################################################
.PHONY: help
help: ## Displays help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-z0-9A-Z_-]+:.*?##/ { printf "  \033[36m%-40s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##############################################################
# devenv commands
##############################################################
.PHONY: devenv-clickhouse
devenv-clickhouse: ## Run clickhouse in devenv
	@cd .devenv/docker/clickhouse; \
	docker compose -f compose.yaml up -d

.PHONY: devenv-postgres
devenv-postgres: ## Run postgres in devenv
	@cd .devenv/docker/postgres; \
	docker compose -f compose.yaml up -d

.PHONY: devenv-signoz-otel-collector
devenv-signoz-otel-collector: ## Run signoz-otel-collector in devenv (requires clickhouse to be running)
	@cd .devenv/docker/signoz-otel-collector; \
	docker compose -f compose.yaml up -d

.PHONY: devenv-up
devenv-up: devenv-clickhouse devenv-signoz-otel-collector ## Start both clickhouse and signoz-otel-collector for local development
	@echo "Development environment is ready!"
	@echo "   - ClickHouse: http://localhost:8123"
	@echo "   - Signoz OTel Collector: grpc://localhost:4317, http://localhost:4318"

##############################################################
# go commands
##############################################################
.PHONY: go-test
go-test: ## Runs go unit tests
	@go test -race ./...

.PHONY: go-run-community
go-run-community: ## Runs the community go backend server
	@SIGNOZ_INSTRUMENTATION_LOGS_LEVEL=$(SIGNOZ_LOG_LEVEL) \
	SIGNOZ_SQLSTORE_SQLITE_PATH=signoz.db \
	SIGNOZ_WEB_ENABLED=false \
	SIGNOZ_JWT_SECRET=$(SIGNOZ_JWT_SECRET) \
	SIGNOZ_ALERTMANAGER_PROVIDER=signoz \
	SIGNOZ_TELEMETRYSTORE_PROVIDER=clickhouse \
	SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN=$(SIGNOZ_CLICKHOUSE_DSN) \
	SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_CLUSTER=$(SIGNOZ_CLICKHOUSE_CLUSTER) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__FROM=$(SIGNOZ_SMTP_FROM) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__HELLO=$(SIGNOZ_SMTP_HELLO) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__SMARTHOST=$(SIGNOZ_SMTP_SMARTHOST) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__AUTH__USERNAME=$(SIGNOZ_SMTP_USERNAME) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__AUTH__PASSWORD=$(SIGNOZ_SMTP_PASSWORD) \
	SIGNOZ_ALERTMANAGER_SIGNOZ_GLOBAL_SMTP__REQUIRE__TLS=$(SIGNOZ_SMTP_REQUIRE_TLS) \
	K_O11Y_SSO_ENABLED=$(K_O11Y_SSO_ENABLED) \
	K_O11Y_JWT_PUBLIC_KEY=$(K_O11Y_JWT_PUBLIC_KEY) \
	K_O11Y_JWT_ISSUER=$(K_O11Y_JWT_ISSUER) \
	K_O11Y_DEFAULT_ROLE=$(K_O11Y_DEFAULT_ROLE) \
	go run -race \
		$(GO_BUILD_CONTEXT_COMMUNITY)/*.go server

.PHONY: go-build-community $(GO_BUILD_ARCHS_COMMUNITY)
go-build-community: ## Builds the go backend server for community
go-build-community: $(GO_BUILD_ARCHS_COMMUNITY)
$(GO_BUILD_ARCHS_COMMUNITY): go-build-community-%: $(TARGET_DIR)
	@mkdir -p $(TARGET_DIR)/$(OS)-$*
	@echo ">> building binary $(TARGET_DIR)/$(OS)-$*/$(NAME)-community"
	@if [ $* = "arm64" ]; then \
		GOARCH=$* GOOS=$(OS) go build -C $(GO_BUILD_CONTEXT_COMMUNITY) -tags timetzdata -o $(TARGET_DIR)/$(OS)-$*/$(NAME)-community -ldflags "-s -w $(GO_BUILD_LDFLAGS_COMMUNITY)"; \
	else \
		GOARCH=$* GOOS=$(OS) go build -C $(GO_BUILD_CONTEXT_COMMUNITY) -tags timetzdata -o $(TARGET_DIR)/$(OS)-$*/$(NAME)-community -ldflags "-s -w $(GO_BUILD_LDFLAGS_COMMUNITY)"; \
	fi



##############################################################
# js commands
##############################################################
.PHONY: js-build
js-build: ## Builds the js frontend
	@echo ">> building js frontend"
	@cd $(JS_BUILD_CONTEXT) && CI=1 yarn install && yarn build


##############################################################
# k-o11y hub build and push command
##############################################################
.PHONY: o11y-build-and-push
o11y-build-and-push: ## Build community docker image and push to registry (Usage: make o11y-build-and-push)
	@if [ -z "$(TAG)" ]; then \
		read -p "Enter TAG version (e.g., 0.1.20): " tag_input; \
		$(MAKE) o11y-build-and-push TAG=$$tag_input; \
	else \
		echo ">> Building and pushing community image with tag: $(TAG)"; \
		DOCKER_IMG_BUILD=true \
		VERSION=$(TAG) \
		DOCKER_REGISTRY_COMMUNITY=<YOUR_REGISTRY>/observability/hub \
		OS=linux \
		ARCHS=amd64 \
		$(MAKE) docker-build-community-amd64; \
		echo ">> Build completed. Pushing to registry..."; \
		docker push <YOUR_REGISTRY>/observability/hub:$(TAG)-amd64; \
		echo ">> Successfully pushed <YOUR_REGISTRY>/observability/hub:$(TAG)-amd64"; \
	fi


##############################################################
# docker commands
##############################################################
.PHONY: docker-build-community $(DOCKER_BUILD_ARCHS_COMMUNITY)
docker-build-community: ## Builds the docker image for community
docker-build-community: $(DOCKER_BUILD_ARCHS_COMMUNITY)
$(DOCKER_BUILD_ARCHS_COMMUNITY): docker-build-community-%: go-build-community-% js-build
	@echo ">> building docker image for $(NAME)-community"
	@docker build -t "$(DOCKER_REGISTRY_COMMUNITY):$(VERSION)-$*" \
		--build-arg TARGETARCH="$*" \
		-f $(DOCKERFILE_COMMUNITY) $(SRC)

.PHONY: docker-buildx-community
docker-buildx-community: ## Builds the docker image for community using buildx
docker-buildx-community: go-build-community js-build
	@echo ">> building docker image for $(NAME)-community"
	@docker buildx build --file $(DOCKERFILE_COMMUNITY) \
		--progress plain \
		--platform linux/arm64,linux/amd64 \
		--push \
		--tag $(DOCKER_REGISTRY_COMMUNITY):$(VERSION) $(SRC)


##############################################################
# python commands
##############################################################
.PHONY: py-fmt
py-fmt: ## Run black for integration tests
	@cd tests/integration && poetry run black .

.PHONY: py-lint
py-lint: ## Run lint for integration tests
	@cd tests/integration && poetry run isort .
	@cd tests/integration && poetry run autoflake .
	@cd tests/integration && poetry run pylint .

.PHONY: py-test
py-test: ## Runs integration tests
	@cd tests/integration && poetry run pytest --basetemp=./tmp/ -vv --capture=no src/
