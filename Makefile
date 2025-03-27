.PHONY: all build clean test coverage lint run docker-build docker-run help

# 기본 변수 설정
APP_NAME := api-gateway
BUILD_DIR := build
MAIN_PATH := cmd/gateway/main.go
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")
PKG_LIST := $(shell go list ./... | grep -v /vendor/)

# 버전 및 빌드 정보
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# 기본 작업: 빌드
all: clean build

# 빌드 작업
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# 디버그 정보 포함 빌드
build-debug:
	@echo "Building $(APP_NAME) with debug info..."
	@mkdir -p $(BUILD_DIR)
	@go build -gcflags "all=-N -l" $(LD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Debug build complete: $(BUILD_DIR)/$(APP_NAME)"

# 릴리스용 빌드 (최적화)
build-release:
	@echo "Building $(APP_NAME) for release..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s $(LD_FLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Release build complete: $(BUILD_DIR)/$(APP_NAME)"

# 정리 작업
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean -cache
	@echo "Clean complete"

# 코드 테스트
test:
	@echo "Running tests..."
	@go test -v $(PKG_LIST)

# 테스트 커버리지 확인
coverage:
	@echo "Running test coverage..."
	@mkdir -p test-reports
	@go test -coverprofile=test-reports/coverage.out $(PKG_LIST)
	@go tool cover -html=test-reports/coverage.out -o test-reports/coverage.html
	@echo "Coverage report generated at test-reports/coverage.html"

# 유닛 테스트만 실행
test-unit:
	@echo "Running unit tests..."
	@mkdir -p test-reports
	@go test -v ./... -tags=unit -coverprofile=test-reports/unit.out
	@go tool cover -html=test-reports/unit.out -o test-reports/unit_coverage.html
	@echo "Unit test coverage report generated at test-reports/unit_coverage.html"

# 통합 테스트만 실행
test-integration:
	@echo "Running integration tests..."
	@mkdir -p test-reports
	@go test -v ./... -tags=integration -coverprofile=test-reports/integration.out
	@go tool cover -html=test-reports/integration.out -o test-reports/integration_coverage.html
	@echo "Integration test coverage report generated at test-reports/integration_coverage.html"

# 전체 테스트 실행 (유닛 + 통합)
test-all:
	@echo "Running all tests..."
	@mkdir -p test-reports
	@go test -v ./... -coverprofile=test-reports/total.out
	@go tool cover -html=test-reports/total.out -o test-reports/total_coverage.html
	@echo "Total test coverage report generated at test-reports/total_coverage.html"

# 코드 린트 검사
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 애플리케이션 실행
run:
	@echo "Running $(APP_NAME)..."
	@go run $(MAIN_PATH)

# 리팩토링 체크
refactor-check:
	@echo "Checking for refactoring opportunities..."
	@go vet ./...
	@staticcheck ./...

# 도커 이미지 빌드
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) .
	@echo "Docker image $(APP_NAME):$(VERSION) built"

# 도커 이미지 실행
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --name $(APP_NAME) --rm $(APP_NAME):$(VERSION)

# 종속성 업데이트
deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "Dependencies updated"

# 종속성 확인
deps-check:
	@echo "Checking dependencies..."
	@go mod verify
	@echo "Dependencies OK"

# 라이브 리로드 실행 (air 사용)
dev:
	@echo "Running with live reload..."
	@air -c .air.toml

# 도움말
help:
	@echo "Available commands:"
	@echo "  make              - Build the application"
	@echo "  make build        - Build the application"
	@echo "  make build-debug  - Build with debug information"
	@echo "  make build-release - Build optimized for release"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make test-unit    - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-all     - Run all tests with coverage"
	@echo "  make lint         - Run linter"
	@echo "  make run          - Run the application"
	@echo "  make refactor-check - Check for refactoring opportunities"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo "  make deps-update  - Update dependencies"
	@echo "  make deps-check   - Verify dependencies"
	@echo "  make dev          - Run with live reload"
	@echo "  make help         - Show this help message"
