package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/internal/middleware"
)

func TestRateLimitMiddleware(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 레이트 리미터 설정 (테스트를 위한 낮은 값)
    window := 1 * time.Second
    maxRequests := 3
    rateLimiter := middleware.NewRateLimiter(window, maxRequests)
    
    // 클리너 시작 (테스트 후 정리)
    cleanupInterval := 100 * time.Millisecond
    stopCleaner := rateLimiter.StartCleaner(cleanupInterval)
    defer stopCleaner()
    
    // 미들웨어 등록
    router.Use(middleware.RateLimitMiddleware(rateLimiter))
    
    // 테스트 엔드포인트 설정
    router.GET("/api/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })
    
    // 동일한 IP에서 여러 요청 테스트
    clientIP := "192.168.1.1"
    
    // 허용된 횟수만큼 요청
    for i := 0; i < maxRequests; i++ {
        req, _ := http.NewRequest("GET", "/api/test", nil)
        req.RemoteAddr = clientIP + ":12345" // IP 주소 설정
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusOK, w.Code, "요청 %d: 상태 코드가 OK여야 함", i+1)
        
        // 남은 요청 수 확인 (X-RateLimit-Remaining 헤더)
        remaining := maxRequests - (i + 1)
        assert.Equal(t, remaining, rateLimiter.GetRemaining(clientIP), "남은 요청 수가 일치해야 함")
    }
    
    // 한도 초과 요청
    req, _ := http.NewRequest("GET", "/api/test", nil)
    req.RemoteAddr = clientIP + ":12345"
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 한도 초과 시 429 상태 코드 예상
    assert.Equal(t, http.StatusTooManyRequests, w.Code, "한도 초과 시 429 상태 코드여야 함")
    
    // 재시도 시간 헤더 확인
    retryAfter := w.Header().Get("Retry-After")
    assert.NotEmpty(t, retryAfter, "Retry-After 헤더가 있어야 함")
    
    // 윈도우 기간 대기 후 리미터 재설정 확인
    time.Sleep(window + 100*time.Millisecond)
    
    // 대기 후 다시 요청 가능해야 함
    req, _ = http.NewRequest("GET", "/api/test", nil)
    req.RemoteAddr = clientIP + ":12345"
    
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code, "윈도우 기간 후 요청이 성공해야 함")
}

func TestRateLimitWithMultipleIPs(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 레이트 리미터 설정
    window := 1 * time.Second
    maxRequests := 2
    rateLimiter := middleware.NewRateLimiter(window, maxRequests)
    
    // 클리너 시작
    cleanupInterval := 100 * time.Millisecond
    stopCleaner := rateLimiter.StartCleaner(cleanupInterval)
    defer stopCleaner()
    
    // 미들웨어 등록
    router.Use(middleware.RateLimitMiddleware(rateLimiter))
    
    // 테스트 엔드포인트 설정
    router.GET("/api/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })
    
    // 여러 IP 주소
    ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
    
    // 각 IP에서 최대 요청 수만큼 요청
    for _, ip := range ips {
        for i := 0; i < maxRequests; i++ {
            req, _ := http.NewRequest("GET", "/api/test", nil)
            req.RemoteAddr = ip + ":12345"
            
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            
            assert.Equal(t, http.StatusOK, w.Code, "IP %s, 요청 %d: 상태 코드가 OK여야 함", ip, i+1)
        }
        
        // 한도 초과 요청
        req, _ := http.NewRequest("GET", "/api/test", nil)
        req.RemoteAddr = ip + ":12345"
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusTooManyRequests, w.Code, "IP %s: 한도 초과 시 429 상태 코드여야 함", ip)
    }
    
    // 윈도우 기간 대기 후 모든 IP의 리미터 재설정 확인
    time.Sleep(window + 100*time.Millisecond)
    
    // 대기 후 모든 IP에서 다시 요청 가능해야 함
    for _, ip := range ips {
        req, _ := http.NewRequest("GET", "/api/test", nil)
        req.RemoteAddr = ip + ":12345"
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        assert.Equal(t, http.StatusOK, w.Code, "IP %s: 윈도우 기간 후 요청이 성공해야 함", ip)
    }
}
