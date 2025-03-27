#!/bin/bash

# Golang API Gateway 테스트 스크립트

set -e  # 오류 발생 시 스크립트 중단

# 색상 정의
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 테스트 결과 폴더 생성
mkdir -p test-reports

# 변수 정의
ALL_PASSED=true

print_header() {
    echo -e "\n${BLUE}====================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}====================================================${NC}"
}

run_tests() {
    local test_type=$1
    local pkg_pattern=$2
    local output_file=$3
    local coverage_file=$4
    
    print_header "Running $test_type tests..."
    
    # 테스트 실행
    if go test -v $pkg_pattern -coverprofile=$coverage_file; then
        echo -e "${GREEN}✓ $test_type tests passed${NC}"
    else
        echo -e "${RED}✗ $test_type tests failed${NC}"
        ALL_PASSED=false
    fi
    
    # 커버리지 리포트 생성
    go tool cover -html=$coverage_file -o $output_file
    echo -e "${GREEN}✓ Coverage report generated at ${output_file}${NC}"
}

# 유닛 테스트 실행
run_tests "Unit" "./... -tags=unit" "test-reports/unit_coverage.html" "test-reports/unit.out"

# 통합 테스트 실행
run_tests "Integration" "./... -tags=integration" "test-reports/integration_coverage.html" "test-reports/integration.out"

# 전체 테스트 커버리지 계산
print_header "Calculating total coverage..."
echo "mode: set" > test-reports/total.out
grep -h -v "mode: set" test-reports/unit.out test-reports/integration.out >> test-reports/total.out
go tool cover -html=test-reports/total.out -o test-reports/total_coverage.html
echo -e "${GREEN}✓ Total coverage report generated at test-reports/total_coverage.html${NC}"

# 커버리지 비율 출력
coverage=$(go tool cover -func=test-reports/total.out | grep total | awk '{print $3}')
echo -e "\n${YELLOW}Total test coverage: ${coverage}${NC}"

# 테스트 요약
print_header "Test Summary"
if [ "$ALL_PASSED" = true ]; then
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please check the logs above for details.${NC}"
    exit 1
fi
