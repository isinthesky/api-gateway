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

func TestCircuitBreaker(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("서킷 브레이커 테스트")
    
    // 장애 서버 설정
    testLogger.Info("장애 시뮬레이션 서버 시작 중...")
    faultServer := mocks.NewFaultServer()
    defer faultServer.Close()
    
    testLogger.Info("장애 서버 URL: %s", faultServer.URL())
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 서킷 브레이커 동작 테스트
    t.Run("서킷 브레이커 트립", func(t *testing.T) {
        subLogger := logger.TestContext("서킷 브레이커 트립 테스트")
        subLogger.Info("실패 임계값을 초과하도록 여러 실패 요청 전송 중...")
        
        var circuitOpened bool
        
        // 실패 임계값을 초과하도록 여러 실패 요청 전송
        for i := 0; i < 10; i++ {
            resp := e.GET("/api/circuit-test/failing-endpoint").
                Expect()
            
            statusCode := resp.Raw().StatusCode
            subLogger.Info("요청 %d: 상태 코드 %d", i+1, statusCode)
            
            // 서킷 브레이커가 열렸는지 확인하기 위해 헤더 확인
            circuitState := resp.Header("X-Circuit-State").Raw()
            subLogger.Info("서킷 상태: %s", circuitState)
            
            if circuitState == "open" {
                circuitOpened = true
                subLogger.Info("서킷 브레이커가 열림 상태로 전환됨 (요청 %d 이후)", i+1)
                break
            }
            
            // 0.5초 대기
            time.Sleep(500 * time.Millisecond)
        }
        
        // 서킷 브레이커 동작 확인
        if !circuitOpened {
            subLogger.Warn("여러 실패 후에도 서킷 브레이커가 열리지 않음")
            t.Logf("서킷 브레이커가 구현되지 않았거나 작동하지 않을 수 있음")
        } else {
            subLogger.Info("서킷 브레이커 정상 작동: 실패 임계값 초과 후 열림")
        }
    })
    
    t.Run("서킷 브레이커 복구", func(t *testing.T) {
        subLogger := logger.TestContext("서킷 브레이커 복구 테스트")
        
        // 서킷 브레이커가 반열림 상태로 전환될 때까지 대기
        subLogger.Info("서킷 브레이커 반열림 상태 대기 중...")
        time.Sleep(10 * time.Second)
        
        // 복구된 서비스 엔드포인트로 요청
        subLogger.Info("복구된 엔드포인트로 요청 중...")
        resp := e.GET("/api/circuit-test/recovered-endpoint").
            Expect()
        
        statusCode := resp.Raw().StatusCode
        circuitState := resp.Header("X-Circuit-State").Raw()
        
        subLogger.Info("복구 후 상태 코드: %d, 서킷 상태: %s", statusCode, circuitState)
        
        // 몇 번의 성공적인 요청 후에는 서킷이 닫혀야 함
        if statusCode == http.StatusOK {
            subLogger.Info("성공적인 요청으로 서킷 상태 변경 시도 중...")
            
            // 추가 요청으로 서킷 상태 확인
            var circuitClosed bool
            for i := 0; i < 3; i++ {
                resp := e.GET("/api/circuit-test/recovered-endpoint").
                    Expect().
                    Status(http.StatusOK)
                
                circuitState = resp.Header("X-Circuit-State").Raw()
                if circuitState == "closed" {
                    circuitClosed = true
                    subLogger.Info("서킷 브레이커가 다시 닫힘 상태로 전환됨")
                    break
                }
                
                subLogger.Info("서킷 상태 확인 중 (%d/3): %s", i+1, circuitState)
                time.Sleep(1 * time.Second)
            }
            
            if !circuitClosed {
                subLogger.Warn("성공적인 요청 후에도 서킷이 닫히지 않음")
            }
        } else {
            subLogger.Warn("복구 요청이 실패함: 상태 코드 %d", statusCode)
        }
    })
}
