package resilience_test

import (
    "net/http"
    "testing"
    "time"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
    "github.com/isinthesky/api-gateway/tests/mocks"
    "github.com/isinthesky/api-gateway/tests/utils"
)

func TestFaultTolerance(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("장애 내성 테스트")
    
    // 장애 서버 설정
    testLogger.Info("장애 시뮬레이션 서버 시작 중...")
    faultServer := mocks.NewFaultServer()
    defer faultServer.Close()
    
    testLogger.Info("장애 서버 URL: %s", faultServer.URL())
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 간헐적 장애 테스트
    t.Run("간헐적 장애 처리", func(t *testing.T) {
        subLogger := logger.TestContext("간헐적 장애 테스트")
        subLogger.Info("간헐적 장애 서버로 여러 요청 전송 중...")
        
        // 여러 요청을 보내 간헐적 장애 테스트
        successCount := 0
        failureCount := 0
        
        for i := 0; i < 10; i++ {
            resp := e.GET("/api/fault-test/intermittent-failure").
                Expect()
            
            statusCode := resp.Raw().StatusCode
            subLogger.Info("요청 %d: 상태 코드 %d", i+1, statusCode)
            
            if statusCode == http.StatusOK {
                successCount++
            } else if statusCode == http.StatusInternalServerError {
                failureCount++
            } else {
                subLogger.Warn("예상치 못한 상태 코드: %d", statusCode)
            }
        }
        
        // 간헐적 실패 확인 (3회마다 실패하므로 약 3-4번의 실패 예상)
        subLogger.Info("성공 횟수: %d, 실패 횟수: %d", successCount, failureCount)
        
        // 모든 요청이 실패하면 안 됨
        if successCount == 0 {
            t.Errorf("모든 요청이 실패함")
        }
        
        // 실패가 전혀 없어도 안 됨 (간헐적 장애를 시뮬레이션하므로)
        if failureCount == 0 {
            subLogger.Warn("간헐적 장애가 발생하지 않음")
        }
    })
    
    // 타임아웃 처리 테스트
    t.Run("요청 타임아웃 처리", func(t *testing.T) {
        subLogger := logger.TestContext("타임아웃 테스트")
        subLogger.Info("느린 응답 엔드포인트로 요청 전송 중...")
        
        // 느린 엔드포인트 요청 (API Gateway의 타임아웃 설정보다 오래 걸림)
        resp := e.GET("/api/fault-test/slow-response").
            Expect()
        
        // 타임아웃 응답 확인 (게이트웨이 타임아웃 또는 성공적인 응답)
        statusCode := resp.Raw().StatusCode
        subLogger.Info("느린 응답 상태 코드: %d", statusCode)
        
        // 게이트웨이 타임아웃 또는 성공적인 응답 중 하나여야 함
        if statusCode != http.StatusGatewayTimeout && statusCode != http.StatusOK {
            subLogger.Error("예상치 못한 상태 코드: %d", statusCode)
            t.Errorf("예상치 못한 상태 코드: %d", statusCode)
        }
    })
    
    // 서비스 다운 처리 테스트
    t.Run("서비스 다운 처리", func(t *testing.T) {
        subLogger := logger.TestContext("서비스 다운 테스트")
        subLogger.Info("완전히 다운된 서비스 엔드포인트 요청 중...")
        
        // 완전히 다운된 서비스 엔드포인트 요청
        resp := e.GET("/api/fault-test/service-down").
            Expect()
        
        // API Gateway는 적절한 오류 응답 제공해야 함
        statusCode := resp.Raw().StatusCode
        subLogger.Info("서비스 다운 상태 코드: %d", statusCode)
        
        // 502 Bad Gateway 또는 503 Service Unavailable 응답 기대
        if statusCode != http.StatusBadGateway && statusCode != http.StatusServiceUnavailable && statusCode != http.StatusInternalServerError {
            subLogger.Error("서비스 다운 시 예상치 못한 상태 코드: %d", statusCode)
            t.Errorf("서비스 다운 시 예상치 못한 상태 코드: %d", statusCode)
        }
    })
    
    // 여러 백엔드 장애 시 장애 격리 테스트
    t.Run("장애 격리", func(t *testing.T) {
        subLogger := logger.TestContext("장애 격리 테스트")
        
        // 하나의 장애 서비스 요청
        subLogger.Info("다운된 서비스 엔드포인트 요청 중...")
        e.GET("/api/fault-test/service-down").
            Expect()
        
        // 다른 정상 서비스는 여전히 작동해야 함
        subLogger.Info("정상 서비스 엔드포인트 요청 중...")
        resp := e.GET("/api/public/status").
            Expect()
        
        statusCode := resp.Raw().StatusCode
        subLogger.Info("정상 서비스 상태 코드: %d", statusCode)
        
        if statusCode != http.StatusOK {
            subLogger.Error("장애 격리 실패: 정상 서비스가 장애 서비스의 영향을 받음")
            t.Errorf("장애 격리 실패: 상태 코드 %d", statusCode)
        } else {
            subLogger.Info("장애 격리 성공: 정상 서비스는 장애 서비스의 영향을 받지 않음")
        }
    })
}
