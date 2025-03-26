package utils

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "runtime"
    "strings"
    "testing"
)

// HTTPResponseDebugInfo는 HTTP 응답에 대한 디버그 정보를 수집합니다
func HTTPResponseDebugInfo(t *testing.T, resp *http.Response) string {
    if resp == nil {
        return "응답이 nil입니다"
    }
    
    var builder strings.Builder
    
    // 응답 기본 정보
    builder.WriteString(fmt.Sprintf("상태 코드: %d (%s)\n", resp.StatusCode, resp.Status))
    builder.WriteString("응답 헤더:\n")
    
    for key, values := range resp.Header {
        builder.WriteString(fmt.Sprintf("  %s: %s\n", key, strings.Join(values, ", ")))
    }
    
    // 응답 본문
    body, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    
    if err != nil {
        builder.WriteString(fmt.Sprintf("응답 본문 읽기 오류: %v\n", err))
    } else {
        builder.WriteString("응답 본문:\n")
        
        // JSON 형식인 경우 예쁘게 출력
        if isJSON(string(body)) {
            var prettyJSON bytes.Buffer
            if err := json.Indent(&prettyJSON, body, "  ", "  "); err == nil {
                builder.WriteString(prettyJSON.String())
            } else {
                builder.WriteString(string(body))
            }
        } else {
            builder.WriteString(string(body))
        }
    }
    
    return builder.String()
}

// 문자열이 JSON인지 확인
func isJSON(str string) bool {
    var js json.RawMessage
    return json.Unmarshal([]byte(str), &js) == nil
}

// AssertWithDebugInfo는 조건을 검증하고 실패 시 디버그 정보를 기록합니다
func AssertWithDebugInfo(t *testing.T, logger *TestLogger, condition bool, debugInfo, format string, args ...interface{}) {
    if !condition {
        message := fmt.Sprintf(format, args...)
        logger.Error("%s\n디버그 정보:\n%s", message, debugInfo)
        t.Errorf("%s", message)
    }
}

// 실패 시 환경 상태 로깅 헬퍼
type SystemState struct {
    NumGoroutine int
    MemStats     runtime.MemStats
    Environment  map[string]string
}

// CaptureSystemState는 현재 시스템 상태를 캡처합니다
func CaptureSystemState() SystemState {
    state := SystemState{
        NumGoroutine: runtime.NumGoroutine(),
        Environment:  make(map[string]string),
    }
    
    runtime.ReadMemStats(&state.MemStats)
    
    // 주요 환경 변수 캡처
    for _, key := range []string{"PORT", "LOG_LEVEL", "TEST_ENV", "GATEWAY_PORT"} {
        state.Environment[key] = os.Getenv(key)
    }
    
    return state
}

// LogSystemState는 시스템 상태를 로그에 기록합니다
func LogSystemState(logger *TestLogger, state SystemState) {
    logger.Debug("===== 시스템 상태 =====")
    logger.Debug("고루틴 수: %d", state.NumGoroutine)
    logger.Debug("메모리 할당: %d MB", state.MemStats.Alloc/1024/1024)
    logger.Debug("총 메모리 할당: %d MB", state.MemStats.TotalAlloc/1024/1024)
    logger.Debug("환경 변수:")
    
    for k, v := range state.Environment {
        logger.Debug("  %s: %s", k, v)
    }
    
    logger.Debug("=====================")
}
