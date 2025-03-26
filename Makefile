# Makefile에 추가할 내용

.PHONY: test test-unit test-integration test-e2e test-report test-coverage

test: ## 모든 테스트 실행
	./tests/run_tests.sh

test-unit: ## 단위 테스트만 실행
	go test -v ./tests/unit/... -coverprofile=./test-reports/unit.out

test-integration: ## 통합 테스트만 실행
	go test -v ./tests/integration/... -coverprofile=./test-reports/integration.out

test-e2e: ## E2E 테스트만 실행
	go test -v ./tests/e2e/... -coverprofile=./test-reports/e2e.out

test-report: ## 테스트 보고서 생성
	mkdir -p ./test-reports
	go tool cover -html=./test-reports/total.out -o ./test-reports/total_coverage.html

test-coverage: ## 테스트 커버리지 출력
	go tool cover -func=./test-reports/total.out
