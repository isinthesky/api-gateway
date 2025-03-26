테스트 코드 리포트와 테스트 디렉토리를 확인한 후 단위 테스트를 실행해보겠습니다.

Read file: tests/run_tests.sh

Ran tool

Read file: go.mod

Ran tool

Ran tool

Ran tool

```bash
chmod +x tests/run_tests.sh
```

```bash
go test -v ./tests/unit/... -coverprofile=./test-reports/unit.out
```

Read file: tests/unit/config/config_test.go

Read file: config/config.go

Read file: tests/unit/middleware/rate_limit_test.go

Read file: middleware/rate_limiter.go

Ran tool

Read file: internal/middleware/ratelimit.go

Ran tool

Read file: tests/unit/proxy/reverseproxy_test.go

Read file: proxy/reverseproxy.go
단위 테스트 실행 결과를 분석한 후 실패 원인을 설명해드리겠습니다.

단위 테스트가 실패하는 원인은 다음 세 가지입니다:

1. **Config 구조체 필드 불일치**:
   - 테스트 코드: `tests/unit/config/config_test.go`에서 `MaxRequestSize` 필드를 참조하고 있지만
   - 실제 코드: `config/config.go`에서는 해당 필드가 없고 대신 `MaxContentSize` 필드가 존재합니다.

2. **RateLimiter 메서드 불일치**:
   - 테스트 코드: `tests/unit/middleware/rate_limit_test.go`에서 `rateLimiter.GetRemaining(clientIP)` 메서드를 호출하고 있지만
   - 실제 코드: `internal/middleware/ratelimit.go`에는 이 메서드가 구현되어 있지 않습니다.
   - 또한 `StartCleaner`는 반환 값을 사용하지 않는데 테스트에서는 반환값이 있는 것으로 가정합니다.

3. **HTTP 라이브러리 호환성 문제**:
   - `proxy/reverseproxy_test.go`에서 `httptest.ResponseRecorder`는 `http.CloseNotifier` 인터페이스를 구현하지 않아 패닉이 발생합니다.
   - 이는 Go 버전 업데이트 또는 Gin 프레임워크 버전 업데이트에 따른 호환성 문제로 보입니다.

이러한 문제들을 해결하기 위해서는:

1. Config 구조체에서 필드 이름을 일치시켜야 합니다 (`MaxContentSize` → `MaxRequestSize` 또는 테스트 코드 수정).
2. RateLimiter에 `GetRemaining` 메서드를 추가하고 `StartCleaner` 메서드가 중지 함수를 반환하도록 수정해야 합니다.
3. HTTP 프록시 테스트의 경우 최신 Go 버전과 호환되도록 `CloseNotify` 관련 코드를 업데이트해야 합니다.

테스트 코드와 실제 구현 코드 간의 불일치로 인해 단위 테스트가 실패하고 있습니다. 코드베이스가 진화하면서 테스트 코드가 업데이트되지 않은 것으로 보입니다.
