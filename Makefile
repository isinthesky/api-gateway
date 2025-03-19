BINARY_NAME=api-gateway
BUILD_DIR=./build
GO_FILES=$(shell find . -name "*.go" -type f)
PKG_LIST=$(shell go list ./... | grep -v /vendor/)

.PHONY: all build clean test run lint help

all: test build

build: ## 실행 파일 빌드
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

run: ## 개발 모드로 실행
	go run main.go

clean: ## 빌드 파일 제거
	rm -rf $(BUILD_DIR)
	go clean

test: ## 테스트 실행
	go test $(PKG_LIST) -v

lint: ## 코드 린트 검사
	golangci-lint run

deps: ## 의존성 설치
	go mod tidy
	go mod vendor

help: ## 도움말 출력
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-docker: ## Docker 이미지 빌드
	docker build -t $(BINARY_NAME):latest .

run-docker: ## Docker 컨테이너로 실행
	docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

release: ## 릴리즈 빌드
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

mockoon-start: ## Mockoon을 사용하여 백엔드 목업 서버 시작
	mockoon-cli start --data config/mockoon.json 