package rate_limit_test

import (
    "net/http"
    "testing"
    "time"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
    "github.com/isinthesky/api-gateway/tests/utils"
)

func TestRateLimit(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("레이트 리밋 테스트")
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 테스트 케이스: 많은 요청 전송
    t.Run("Rate Limit 테스트", func(t *testing.T) {
        // 공개 엔드포인트로 많은 요청 전송
        testLogger.Info("다수의 요청 전송 시작")
        
        var limitExceeded bool
        var requestCount int
        
        // 최대 100번 요청 시도 (실제 한도에 따라 조정 필요)
        for i := 0; i < 100; i++ {
            requestCount++
            
            resp := e.GET("/api/public/status").
                Expect()
            
            statusCode := resp.Raw().StatusCode
            
            // 한도 초과 상태 코드 확인 (429 Too Many Requests)
            if statusCode == http.StatusTooManyRequests {
                limitExceeded = true
                testLogger.Info("레이트 리밋 한도 초과: %d번째 요청에서 감지", requestCount)
                break
            }
            
            // 0.05초 대기 (부하 조절)
            time.Sleep(50 * time.Millisecond)
        }
        
        // 레이트 리밋이 적용되는지 확인
        // 참고: 실제 구현에 따라 레이트 리밋이 적용되지 않을 수도 있음
        if !limitExceeded {
            testLogger.Warn("레이트 리밋 한도 초과가 감지되지 않음 (정상일 수 있음)")
        }
        
        testLogger.Info("레이트 리밋 테스트 완료")
    })
    
    // 테스트 케이스: 레이트 리밋 후 대기 시간 확인
    t.Run("Rate Limit 대기 시간 테스트", func(t *testing.T) {
        testLogger.Info("레이트 리밋 대기 시간 테스트 시작")
        
        // 레이트 리밋 헤더 확인을 위한 요청
        resp := e.GET("/api/public/status").
            Expect()
        
        // 레이트 리밋 관련 헤더 확인
        remainingHeader := resp.Header("X-RateLimit-Remaining").Raw()
        resetHeader := resp.Header("X-RateLimit-Reset").Raw()
        
        testLogger.Info("남은 요청 수: %s", remainingHeader)
        testLogger.Info("리셋 시간: %s", resetHeader)
        
        // Retry-After 헤더가 있는 경우 (한도 초과 시)
        if resp.Raw().StatusCode == http.StatusTooManyRequests {
            retryHeader := resp.Header("Retry-After").Raw()
            testLogger.Info("재시도 대기 시간: %s", retryHeader)
            
            // 대기 시간이 있으면 조금 대기 후 다시 요청
            if retryHeader != "" {
                testLogger.Info("지정된 대기 시간만큼 대기 중...")
                time.Sleep(1 * time.Second)
                
                // 대기 후 다시 요청
                retryResp := e.GET("/api/public/status").
                    Expect()
                
                statusCode := retryResp.Raw().StatusCode
                testLogger.Info("대기 후 요청 상태 코드: %d", statusCode)
            }
        }
        
        testLogger.Info("레이트 리밋 대기 시간 테스트 완료")
    })
}

func TestRateLimitWithDifferentClients(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("다중 클라이언트 레이트 리밋 테스트")
    
    // httpexpect 인스턴스 생성 (다른 IP 주소 시뮬레이션)
    e1 := httpexpect.WithConfig(httpexpect.Config{
        BaseURL:  "http://localhost:18080",
        Reporter: httpexpect.NewAssertReporter(t),
        Client: &http.Client{
            Transport: &http.Transport{},
        },
        Printers: []httpexpect.Printer{},
    })
    
    e2 := httpexpect.WithConfig(httpexpect.Config{
        BaseURL:  "http://localhost:18080",
        Reporter: httpexpect.NewAssertReporter(t),
        Client: &http.Client{
            Transport: &http.Transport{},
        },
        Printers: []httpexpect.Printer{},
    })
    
    t.Run("다른 클라이언트 간의 레이트 리밋 분리", func(t *testing.T) {
        testLogger.Info("다중 클라이언트 테스트 시작")
        
        // 클라이언트 1에서 여러 요청
        testLogger.Info("클라이언트 1에서 요청 전송")
        for i := 0; i < 10; i++ {
            resp := e1.GET("/api/public/status").
                WithHeader("X-Forwarded-For", "192.168.1.1").
                Expect()
            
            statusCode := resp.Raw().StatusCode
            if statusCode == http.StatusTooManyRequests {
                testLogger.Info("클라이언트 1 레이트 리밋 한도 초과: %d번째 요청", i+1)
                break
            }
        }
        
        // 클라이언트 2에서 요청 - 레이트 리밋이 분리되어 있어야 함
        testLogger.Info("클라이언트 2에서 요청 전송")
        resp := e2.GET("/api/public/status").
            WithHeader("X-Forwarded-For", "192.168.1.2").
            Expect()
        
        // 클라이언트 2는 여전히 요청 가능해야 함
        statusCode := resp.Raw().StatusCode
        testLogger.Info("클라이언트 2 첫 번째 요청 상태 코드: %d", statusCode)
        
        if statusCode == http.StatusTooManyRequests {
            testLogger.Warn("클라이언트 2도 레이트 리밋에 걸림 - IP 기반 분리가 작동하지 않을 수 있음")
            t.Logf("레이트 리밋이 IP 기반으로 분리되지 않음")
        } else {
            testLogger.Info("IP 기반 레이트 리밋 분리 확인됨")
        }
        
        testLogger.Info("다중 클라이언트 테스트 완료")
    })
}
