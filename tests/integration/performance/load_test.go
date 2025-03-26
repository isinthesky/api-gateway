package performance_test

import (
    "fmt"
    "net/http"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/tests/integration"
    "github.com/isinthesky/api-gateway/tests/utils"
)

func TestAPIGatewayPerformance(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("API Gateway 성능 테스트")
    
    // 동시 요청 수
    concurrentRequests := 50
    
    // 각 요청의 결과를 저장할 배열
    type result struct {
        statusCode int
        duration   time.Duration
        err        error
    }
    
    results := make([]result, concurrentRequests)
    
    // WaitGroup으로 모든 요청 완료 대기
    var wg sync.WaitGroup
    wg.Add(concurrentRequests)
    
    // 시스템 상태 캡처 (테스트 전)
    initialState := utils.CaptureSystemState()
    utils.LogSystemState(testLogger, initialState)
    
    // 테스트 시작 시간
    testLogger.Info("부하 테스트 시작: %d개의 동시 요청", concurrentRequests)
    startTime := time.Now()
    
    // 동시 요청 실행
    for i := 0; i < concurrentRequests; i++ {
        go func(index int) {
            defer wg.Done()
            
            startReq := time.Now()
            
            // 테스트할 API 엔드포인트 (공개 엔드포인트)
            resp, err := http.Get("http://localhost:18080/api/public/status")
            
            if err != nil {
                testLogger.Error("요청 %d 오류: %v", index, err)
                results[index] = result{
                    statusCode: 0,
                    duration:   time.Since(startReq),
                    err:        err,
                }
                return
            }
            defer resp.Body.Close()
            
            results[index] = result{
                statusCode: resp.StatusCode,
                duration:   time.Since(startReq),
                err:        nil,
            }
            
            testLogger.Debug("요청 %d 완료: 상태 코드 %d, 소요 시간 %v", 
                index, resp.StatusCode, time.Since(startReq))
        }(i)
    }
    
    // 모든 요청이 완료될 때까지 대기
    wg.Wait()
    
    // 테스트 총 소요 시간
    totalDuration := time.Since(startTime)
    testLogger.Info("모든 요청 완료: 총 소요 시간 %v", totalDuration)
    
    // 시스템 상태 캡처 (테스트 후)
    finalState := utils.CaptureSystemState()
    utils.LogSystemState(testLogger, finalState)
    
    // 결과 분석
    var successCount int
    var totalResponseTime time.Duration
    var minResponseTime time.Duration = time.Hour
    var maxResponseTime time.Duration
    
    for i, r := range results {
        if r.err == nil && r.statusCode == http.StatusOK {
            successCount++
            totalResponseTime += r.duration
            
            if r.duration < minResponseTime {
                minResponseTime = r.duration
            }
            if r.duration > maxResponseTime {
                maxResponseTime = r.duration
            }
        } else {
            testLogger.Warn("요청 %d 실패: 상태 코드 %d, 오류 %v", 
                i, r.statusCode, r.err)
        }
    }
    
    // 성공률 계산
    successRate := float64(successCount) / float64(concurrentRequests) * 100
    
    // 평균 응답 시간 계산
    var avgResponseTime time.Duration
    if successCount > 0 {
        avgResponseTime = totalResponseTime / time.Duration(successCount)
    }
    
    // 결과 출력
    testLogger.Info("성공률: %.2f%%", successRate)
    testLogger.Info("평균 응답 시간: %v", avgResponseTime)
    testLogger.Info("최소 응답 시간: %v", minResponseTime)
    testLogger.Info("최대 응답 시간: %v", maxResponseTime)
    testLogger.Info("총 테스트 소요 시간: %v", totalDuration)
    
    // 초당 처리량 계산
    if totalDuration > 0 {
        throughput := float64(successCount) / totalDuration.Seconds()
        testLogger.Info("처리량: %.2f 요청/초", throughput)
    }
    
    // 결과 검증
    assert.GreaterOrEqual(t, successRate, 90.0, "성공률이 90%보다 낮습니다")
    assert.LessOrEqual(t, avgResponseTime, 500*time.Millisecond, "평균 응답 시간이 500ms를 초과합니다")
}

func TestAPIGatewayScalability(t *testing.T) {
    // 부하 증가에 따른 성능 변화 테스트
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("API Gateway 확장성 테스트")
    
    // 단계별 부하 증가 테스트
    loadLevels := []int{1, 5, 10, 20, 50}
    
    type loadTestResult struct {
        concurrentRequests int
        successRate        float64
        avgResponseTime    time.Duration
        throughput         float64
    }
    
    results := make([]loadTestResult, len(loadLevels))
    
    // 각 부하 레벨에서 테스트 실행
    for i, numRequests := range loadLevels {
        testLogger.Info("부하 레벨 %d: %d개의 동시 요청 시작", i+1, numRequests)
        
        // 각 요청의 결과를 저장할 배열
        type requestResult struct {
            statusCode int
            duration   time.Duration
            err        error
        }
        
        reqResults := make([]requestResult, numRequests)
        
        // WaitGroup으로 모든 요청 완료 대기
        var wg sync.WaitGroup
        wg.Add(numRequests)
        
        // 테스트 시작 시간
        startTime := time.Now()
        
        // 동시 요청 실행
        for j := 0; j < numRequests; j++ {
            go func(index int) {
                defer wg.Done()
                
                startReq := time.Now()
                
                // 테스트할 API 엔드포인트 (공개 엔드포인트)
                resp, err := http.Get("http://localhost:18080/api/public/status")
                
                if err != nil {
                    reqResults[index] = requestResult{
                        statusCode: 0,
                        duration:   time.Since(startReq),
                        err:        err,
                    }
                    return
                }
                defer resp.Body.Close()
                
                reqResults[index] = requestResult{
                    statusCode: resp.StatusCode,
                    duration:   time.Since(startReq),
                    err:        nil,
                }
            }(j)
        }
        
        // 모든 요청이 완료될 때까지 대기
        wg.Wait()
        
        // 테스트 총 소요 시간
        totalDuration := time.Since(startTime)
        
        // 결과 분석
        var successCount int
        var totalResponseTime time.Duration
        
        for _, r := range reqResults {
            if r.err == nil && r.statusCode == http.StatusOK {
                successCount++
                totalResponseTime += r.duration
            }
        }
        
        // 성공률 계산
        successRate := float64(successCount) / float64(numRequests) * 100
        
        // 평균 응답 시간 계산
        var avgResponseTime time.Duration
        if successCount > 0 {
            avgResponseTime = totalResponseTime / time.Duration(successCount)
        }
        
        // 초당 처리량 계산
        var throughput float64
        if totalDuration > 0 {
            throughput = float64(successCount) / totalDuration.Seconds()
        }
        
        // 결과 저장
        results[i] = loadTestResult{
            concurrentRequests: numRequests,
            successRate:        successRate,
            avgResponseTime:    avgResponseTime,
            throughput:         throughput,
        }
        
        testLogger.Info("부하 레벨 %d 결과: 성공률 %.2f%%, 평균 응답 시간 %v, 처리량 %.2f 요청/초",
            i+1, successRate, avgResponseTime, throughput)
        
        // 부하 테스트 간 간격
        time.Sleep(1 * time.Second)
    }
    
    // 확장성 분석
    testLogger.Info("=== 확장성 분석 결과 ===")
    testLogger.Info("동시 요청 수 | 성공률(%) | 평균 응답 시간 | 처리량(요청/초)")
    testLogger.Info("------------------------------------------")
    
    for _, r := range results {
        testLogger.Info("%14d | %10.2f | %14v | %17.2f",
            r.concurrentRequests, r.successRate, r.avgResponseTime, r.throughput)
    }
    
    // 요청 증가에 따른 응답 시간 증가 여부 확인
    // 이상적으로는 선형 증가보다 작아야 함
    if len(results) >= 2 {
        firstResult := results[0]
        lastResult := results[len(results)-1]
        
        requestsRatio := float64(lastResult.concurrentRequests) / float64(firstResult.concurrentRequests)
        timeRatio := float64(lastResult.avgResponseTime) / float64(firstResult.avgResponseTime)
        
        testLogger.Info("요청 수 증가 비율: %.2f배", requestsRatio)
        testLogger.Info("응답 시간 증가 비율: %.2f배", timeRatio)
        
        if timeRatio > requestsRatio {
            testLogger.Warn("응답 시간이 요청 수보다 더 빠르게 증가: 확장성 문제 가능성")
        } else {
            testLogger.Info("확장성 양호: 응답 시간이 요청 수에 비례하여 증가하지 않음")
        }
    }
}
