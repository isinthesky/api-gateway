#!/bin/bash

# Golang API Gateway 설치 스크립트

set -e  # 오류 발생 시 스크립트 중단

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 헤더 출력 함수
print_header() {
    echo -e "\n${BLUE}====================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}====================================================${NC}"
}

# 폴더 준비
create_directories() {
    print_header "Creating directories..."
    
    # 필요한 디렉토리 생성
    mkdir -p build
    mkdir -p configs
    mkdir -p logs
    
    echo -e "${GREEN}✓ Directories created${NC}"
}

# 종속성 설치
install_dependencies() {
    print_header "Installing dependencies..."
    
    # Go 모듈 다운로드
    go mod download
    go mod tidy
    
    echo -e "${GREEN}✓ Dependencies installed${NC}"
}

# 환경 설정
setup_environment() {
    print_header "Setting up environment..."
    
    # .env 파일이 없는 경우 예제에서 복사
    if [ ! -f .env ]; then
        cp .env.example .env
        echo -e "${YELLOW}! Created .env file from example. Please edit it with your settings.${NC}"
    else
        echo -e "${GREEN}✓ .env file already exists${NC}"
    fi
    
    # 설정 파일 확인
    if [ ! -f configs/routes.json ]; then
        echo -e "${RED}× Routes configuration file not found${NC}"
        echo -e "${YELLOW}! Please create configs/routes.json file${NC}"
    else
        echo -e "${GREEN}✓ Routes configuration file exists${NC}"
    fi
}

# 애플리케이션 빌드
build_application() {
    print_header "Building application..."
    
    # 릴리스 빌드
    go build -ldflags="-w -s" -o build/api-gateway cmd/gateway/main.go
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Application built successfully: build/api-gateway${NC}"
    else
        echo -e "${RED}× Build failed${NC}"
        exit 1
    fi
}

# 테스트 실행
run_tests() {
    print_header "Running tests..."
    
    # 간단한 테스트 실행
    go test -v ./... -short
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Tests passed${NC}"
    else
        echo -e "${RED}× Some tests failed${NC}"
        echo -e "${YELLOW}! Installation will continue, but you should check the test failures${NC}"
    fi
}

# 설치 도구 확인
check_requirements() {
    print_header "Checking requirements..."
    
    # Go 버전 확인
    GO_VERSION=$(go version 2>&1 | grep -o "go1\.[0-9]*\.[0-9]*" || echo "none")
    GO_MAJOR_VERSION=$(echo $GO_VERSION | cut -d. -f2)
    GO_MINOR_VERSION=$(echo $GO_VERSION | cut -d. -f3)
    
    if [[ $GO_MAJOR_VERSION -lt 16 ]]; then
        echo -e "${RED}× Go version is too old: $GO_VERSION${NC}"
        echo -e "${YELLOW}! Please upgrade Go to version 1.16 or later${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ Go version: $GO_VERSION${NC}"
    fi
    
    # Docker 확인 (선택 사항)
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version)
        echo -e "${GREEN}✓ Docker found: $DOCKER_VERSION${NC}"
    else
        echo -e "${YELLOW}! Docker not found. Docker is optional but recommended for containerized deployment${NC}"
    fi
}

# 설치 완료 메시지
print_success() {
    print_header "Installation completed!"
    
    echo -e "${GREEN}API Gateway has been successfully installed!${NC}"
    echo -e "\nTo run the API Gateway:"
    echo -e "  ${YELLOW}./build/api-gateway${NC}"
    echo -e "\nTo use with Docker:"
    echo -e "  ${YELLOW}docker-compose up -d${NC}"
    echo -e "\nFor more information, see the README.md file."
}

# 메인 설치 프로세스
main() {
    print_header "Installing API Gateway"
    
    check_requirements
    create_directories
    install_dependencies
    setup_environment
    run_tests
    build_application
    print_success
}

# 스크립트 실행
main
