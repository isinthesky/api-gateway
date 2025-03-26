#!/bin/bash
# tests/run_tests.sh

# 환경 설정
export TEST_ENV=development
export GATEWAY_PORT=18080

# 필요한 디렉토리 생성
mkdir -p ./test-reports

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 함수: 섹션 헤더 출력
print_header() {
    echo -e "\n${YELLOW}===============================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}===============================================${NC}\n"
}

# 함수: 테스트 결과 출력
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2 성공${NC}"
    else
        echo -e "${RED}✗ $2 실패${NC}"
    fi
}

# 단위 테스트 실행
print_header "단위 테스트 실행 중..."
go test -v ./tests/unit/... -coverprofile=./test-reports/unit.out
UNIT_RESULT=$?
print_result $UNIT_RESULT "단위 테스트"

# 통합 테스트 실행
print_header "통합 테스트 실행 중..."
go test -v ./tests/integration/... -coverprofile=./test-reports/integration.out
INTEGRATION_RESULT=$?
print_result $INTEGRATION_RESULT "통합 테스트"

# E2E 테스트 실행
print_header "E2E 테스트 실행 중..."
go test -v ./tests/e2e/... -coverprofile=./test-reports/e2e.out
E2E_RESULT=$?
print_result $E2E_RESULT "E2E 테스트"

# 커버리지 보고서 생성
print_header "테스트 커버리지 보고서 생성 중..."
go tool cover -html=./test-reports/unit.out -o ./test-reports/unit_coverage.html
go tool cover -html=./test-reports/integration.out -o ./test-reports/integration_coverage.html
go tool cover -html=./test-reports/e2e.out -o ./test-reports/e2e_coverage.html

# 총 커버리지 계산
print_header "총 테스트 커버리지 계산 중..."
go test -coverprofile=./test-reports/total.out ./...
go tool cover -func=./test-reports/total.out | grep total: | awk '{print "총 커버리지: " $3}'

# 결과 요약
print_header "테스트 결과 요약"
echo -e "단위 테스트: $([ $UNIT_RESULT -eq 0 ] && echo "${GREEN}성공${NC}" || echo "${RED}실패${NC}")"
echo -e "통합 테스트: $([ $INTEGRATION_RESULT -eq 0 ] && echo "${GREEN}성공${NC}" || echo "${RED}실패${NC}")"
echo -e "E2E 테스트: $([ $E2E_RESULT -eq 0 ] && echo "${GREEN}성공${NC}" || echo "${RED}실패${NC}")"
echo ""
echo "테스트 보고서는 ./test-reports 디렉토리에 저장되었습니다."

# 최종 종료 코드 설정
if [ $UNIT_RESULT -ne 0 ] || [ $INTEGRATION_RESULT -ne 0 ] || [ $E2E_RESULT -ne 0 ]; then
    exit 1
fi

exit 0
