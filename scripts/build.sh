#!/bin/bash

# Golang API Gateway 빌드 스크립트

set -e  # 오류 발생 시 스크립트 중단

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 환경 변수
BUILD_DIR="build"
APP_NAME="api-gateway"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS="-ldflags=\"-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}\""

# 헤더 출력 함수
print_header() {
    echo -e "\n${BLUE}====================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}====================================================${NC}"
}

# 빌드 디렉토리 준비
prepare_build_dir() {
    print_header "Preparing build directory..."
    
    # 빌드 디렉토리 생성
    mkdir -p ${BUILD_DIR}
    
    echo -e "${GREEN}✓ Build directory prepared${NC}"
}

# 기본 빌드 함수
build_default() {
    print_header "Building ${APP_NAME}..."
    
    echo -e "${YELLOW}Version: ${VERSION}${NC}"
    echo -e "${YELLOW}Commit: ${COMMIT}${NC}"
    echo -e "${YELLOW}Build Time: ${BUILD_TIME}${NC}"
    
    # 기본 빌드 실행
    eval "go build ${LD_FLAGS} -o ${BUILD_DIR}/${APP_NAME} cmd/gateway/main.go"
    
    echo -e "${GREEN}✓ Build completed: ${BUILD_DIR}/${APP_NAME}${NC}"
}

# 디버그 빌드 함수
build_debug() {
    print_header "Building ${APP_NAME} (debug mode)..."
    
    # 디버그 모드로 빌드
    eval "go build -gcflags \"all=-N -l\" ${LD_FLAGS} -o ${BUILD_DIR}/${APP_NAME}-debug cmd/gateway/main.go"
    
    echo -e "${GREEN}✓ Debug build completed: ${BUILD_DIR}/${APP_NAME}-debug${NC}"
}

# 릴리스 빌드 함수
build_release() {
    print_header "Building ${APP_NAME} (release mode)..."
    
    # 릴리스 모드로 빌드
    eval "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags=\"-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}\" -o ${BUILD_DIR}/${APP_NAME}-release cmd/gateway/main.go"
    
    echo -e "${GREEN}✓ Release build completed: ${BUILD_DIR}/${APP_NAME}-release${NC}"
}

# 크로스 컴파일 빌드 함수
build_cross_platform() {
    print_header "Cross-platform building..."
    
    # 다양한 플랫폼용 빌드
    PLATFORMS=("linux:amd64" "darwin:amd64" "darwin:arm64" "windows:amd64")
    
    for platform in "${PLATFORMS[@]}"; do
        IFS=":" read -r os arch <<< "${platform}"
        output="${BUILD_DIR}/${APP_NAME}-${os}-${arch}"
        
        if [ "${os}" = "windows" ]; then
            output="${output}.exe"
        fi
        
        echo -e "${YELLOW}Building for ${os}/${arch}...${NC}"
        eval "CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build ${LD_FLAGS} -o ${output} cmd/gateway/main.go"
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Built for ${os}/${arch}: ${output}${NC}"
        else
            echo -e "${RED}× Failed to build for ${os}/${arch}${NC}"
        fi
    done
}

# Docker 이미지 빌드 함수
build_docker() {
    print_header "Building Docker image..."
    
    echo -e "${YELLOW}Building Docker image: ${APP_NAME}:${VERSION}${NC}"
    
    # Docker 이미지 빌드
    docker build -t ${APP_NAME}:${VERSION} .
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Docker image built: ${APP_NAME}:${VERSION}${NC}"
        echo -e "${YELLOW}You can run the container with: docker run -p 8080:8080 ${APP_NAME}:${VERSION}${NC}"
    else
        echo -e "${RED}× Failed to build Docker image${NC}"
    fi
}

# 도움말 출력 함수
print_help() {
    print_header "API Gateway Build Script"
    
    echo -e "사용법: $0 [옵션]"
    echo -e "\n옵션:"
    echo -e "  ${YELLOW}--help, -h${NC}          이 도움말 표시"
    echo -e "  ${YELLOW}--default, -d${NC}       기본 빌드 실행"
    echo -e "  ${YELLOW}--debug${NC}             디버그 모드로 빌드"
    echo -e "  ${YELLOW}--release, -r${NC}       릴리스 모드로 빌드 (최적화)"
    echo -e "  ${YELLOW}--all, -a${NC}           모든 빌드 실행 (기본, 디버그, 릴리스)"
    echo -e "  ${YELLOW}--cross, -c${NC}         크로스 플랫폼 빌드 실행"
    echo -e "  ${YELLOW}--docker${NC}            Docker 이미지 빌드"
    echo -e "\n예시:"
    echo -e "  $0                  기본 빌드 실행"
    echo -e "  $0 --release        릴리스 모드로 빌드"
    echo -e "  $0 --all            모든 빌드 실행"
}

# 메인 함수
main() {
    # 인수가 없으면 기본 빌드
    if [ $# -eq 0 ]; then
        prepare_build_dir
        build_default
        exit 0
    fi
    
    # 인수 처리
    while [ $# -gt 0 ]; do
        case "$1" in
            --help|-h)
                print_help
                exit 0
                ;;
            --default|-d)
                prepare_build_dir
                build_default
                ;;
            --debug)
                prepare_build_dir
                build_debug
                ;;
            --release|-r)
                prepare_build_dir
                build_release
                ;;
            --all|-a)
                prepare_build_dir
                build_default
                build_debug
                build_release
                ;;
            --cross|-c)
                prepare_build_dir
                build_cross_platform
                ;;
            --docker)
                build_docker
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                print_help
                exit 1
                ;;
        esac
        shift
    done
}

# 스크립트 실행
main "$@"
