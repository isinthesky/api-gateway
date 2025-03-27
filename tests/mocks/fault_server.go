package mocks

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// FaultServer는 다양한 장애 시나리오를 시뮬레이션하는 서버입니다
type FaultServer struct {
    Server       *httptest.Server
    Router       *gin.Engine
    FailureCount int32
    SlowCount    int32
}

// NewFaultServer는 장애 시뮬레이션 서버를 생성합니다
func NewFaultServer() *FaultServer {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(gin.Recovery())

    fs := &FaultServer{
        Router:       router,
        FailureCount: 0,
        SlowCount:    0,
    }

    // 간헐적 장애 엔드포인트
    router.GET("/api/intermittent-failure", fs.intermittentFailureHandler)
    
    // 느린 응답 엔드포인트
    router.GET("/api/slow-response", fs.slowResponseHandler)
    
    // 서버 과부하 시뮬레이션 엔드포인트
    router.GET("/api/overload", fs.overloadHandler)
    
    // 백엔드 서비스 다운 시뮬레이션
    router.GET("/api/service-down", fs.serviceDownHandler)
    
    // 서킷 브레이커 테스트 엔드포인트
    router.GET("/api/circuit-test/failing-endpoint", fs.failingEndpointHandler)
    router.GET("/api/circuit-test/recovered-endpoint", fs.recoveredEndpointHandler)

    server := httptest.NewServer(router)
    fs.Server = server

    return fs
}

// Close는 장애 서버를 종료합니다
func (fs *FaultServer) Close() {
    if fs.Server != nil {
        fs.Server.Close()
    }
}

// URL은 장애 서버 URL을 반환합니다
func (fs *FaultServer) URL() string {
    return fs.Server.URL
}

// intermittentFailureHandler는 간헐적으로 실패하는 핸들러입니다
func (fs *FaultServer) intermittentFailureHandler(c *gin.Context) {
    count := atomic.AddInt32(&fs.FailureCount, 1)
    
    // 3회마다 500 오류 반환
    if count%3 == 0 {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "간헐적 백엔드 장애",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "성공적인 응답",
        "attempt": count,
    })
}

// slowResponseHandler는 느린 응답을 시뮬레이션하는 핸들러입니다
func (fs *FaultServer) slowResponseHandler(c *gin.Context) {
    count := atomic.AddInt32(&fs.SlowCount, 1)
    
    // 2회마다 지연 응답
    if count%2 == 0 {
        // 3초 지연
        time.Sleep(3 * time.Second)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "지연 후 성공적인 응답",
        "attempt": count,
    })
}

// overloadHandler는 서버 과부하 상태를 시뮬레이션합니다
func (fs *FaultServer) overloadHandler(c *gin.Context) {
    // CPU 부하 시뮬레이션
    end := time.Now().Add(500 * time.Millisecond)
    for time.Now().Before(end) {
        // CPU 사용량 증가를 위한 의미 없는 계산
        for i := 0; i < 1000000; i++ {
            _ = i * i
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "과부하 상태에서의 응답",
    })
}

// serviceDownHandler는 완전히 다운된 서비스를 시뮬레이션합니다
func (fs *FaultServer) serviceDownHandler(c *gin.Context) {
    // 응답 없이 연결 끊기 시뮬레이션
    c.Abort()
    c.Writer.WriteHeader(http.StatusServiceUnavailable)
    c.Writer.Flush()
}

// failingEndpointHandler는 서킷 브레이커 테스트를 위한 지속적 실패 핸들러입니다
func (fs *FaultServer) failingEndpointHandler(c *gin.Context) {
    circuitState := c.GetHeader("X-Circuit-State")
    if circuitState == "" {
        circuitState = "closed"
    }
    
    c.Header("X-Circuit-State", circuitState)
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": "서비스 장애",
        "circuitState": circuitState,
    })
}

// recoveredEndpointHandler는 서킷 브레이커 복구 테스트를 위한 성공 핸들러입니다
func (fs *FaultServer) recoveredEndpointHandler(c *gin.Context) {
    circuitState := c.GetHeader("X-Circuit-State")
    if circuitState == "" {
        circuitState = "half-open"
    }
    
    c.Header("X-Circuit-State", "closed")
    c.JSON(http.StatusOK, gin.H{
        "message": "서비스 복구됨",
        "circuitState": "closed",
    })
}
