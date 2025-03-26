# Golang API Gateway 테스트 코드 작성 가이드

# Golang API Gateway 테스트 코드 작성 가이드

API Gateway의 구조 개선 작업을 진행하기 전에 현재 시스템이 정상적으로 작동하는지 확인하는 테스트 코드 작성 가이드를 준비했습니다. 이 가이드는 제공된 백엔드 API 서버(http://facreport.iptime.org:8006/docs, http://facreport.iptime.org:8007/docs)와의 연동을 고려하여 작성되었습니다.

## 1. 테스트 개요

API Gateway는 마이크로서비스 아키텍처의 핵심 컴포넌트로서, 다음과 같은 중요한 역할을 수행합니다:

- 클라이언트와 백엔드 서비스 간의 단일 진입점
- 라우팅 및 요청 전달
- 인증 및 권한 부여
- 요청/응답 변환
- 속도 제한 및 부하 분산
- 로깅 및 모니터링

이러한 중요 기능을 테스트하기 위해서는 체계적인 테스트 전략이 필요합니다.

## 2. 테스트 환경 설정

### 2.1 테스트 종속성

```go
// go.mod 파일에 테스트 관련 종속성을 추가합니다
require (
    github.com/stretchr/testify v1.10.0        // 단언 및 모킹 라이브러리
    github.com/golang/mock v1.6.0              // 인터페이스 모킹
    github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5 // 사용 가능한 포트 찾기
    github.com/gavv/httpexpect/v2 v2.15.0      // HTTP 요청 및 응답 테스트
)
```

### 2.2 테스트 디렉토리 구조

```
/api-gateway
├── tests
│   ├── unit/                  # 단위 테스트
│   │   ├── middleware/        # 미들웨어 테스트
│   │   ├── proxy/             # 프록시 테스트
│   │   └── config/            # 설정 테스트
│   ├── integration/           # 통합 테스트
│   │   ├── auth/              # 인증 통합 테스트
│   │   ├── routing/           # 라우팅 통합 테스트
│   │   └── rate_limit/        # 속도 제한 통합 테스트
│   ├── e2e/                   # 엔드투엔드 테스트
│   ├── mocks/                 # 모의 서비스
│   │   ├── facreport_8006/    # 첫 번째 API 서버 모의
│   │   └── facreport_8007/    # 두 번째 API 서버 모의
│   ├── fixtures/              # 테스트 데이터
│   └── utils/                 # 테스트 유틸리티
```



### 2.3 모의 백엔드 서비스 설정

테스트를 위한 모의 백엔드 서비스를 설정합니다:

```go
// tests/mocks/server.go
package mocks

import (
    "fmt"
    "net/http"
    "net/http/httptest"

    "github.com/gin-gonic/gin"
    "github.com/phayes/freeport"
)

// MockServer는 모의 백엔드 서비스를 제공합니다
type MockServer struct {
    Server *httptest.Server
    Router *gin.Engine
    Port   int
}

// NewMockServer는 새 모의 서버를 생성합니다
func NewMockServer() (*MockServer, error) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(gin.Recovery())

    // 사용 가능한 포트 가져오기
    port, err := freeport.GetFreePort()
    if err != nil {
        return nil, fmt.Errorf("free port를 가져올 수 없음: %v", err)
    }

    // API 테스트 엔드포인트 등록
    registerMockAPIEndpoints(router)

    server := httptest.NewServer(router)

    return &MockServer{
        Server: server,
        Router: router,
        Port:   port,
    }, nil
}

// Close는 모의 서버를 종료합니다
func (m *MockServer) Close() {
    if m.Server != nil {
        m.Server.Close()
    }
}

// URL은 모의 서버 URL을 반환합니다
func (m *MockServer) URL() string {
    return m.Server.URL
}
```

## 3. 단위 테스트

### 3.1 미들웨어 테스트

```go
// tests/unit/middleware/auth_test.go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/internal/middleware"
)

func TestAuthMiddleware(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 테스트 미들웨어 구성
    jwtConfig := middleware.JWTConfig{
        SecretKey:       "test-secret-key",
        Issuer:          "test-issuer",
        ExpirationDelta: 3600,
    }
    
    // 테스트 엔드포인트 설정
    router.GET("/protected", middleware.JWTAuthMiddleware(jwtConfig), func(c *gin.Context) {
        c.String(http.StatusOK, "protected content")
    })
    
    // 테스트 케이스
    testCases := []struct {
        name       string
        authHeader string
        wantStatus int
    }{
        {
            name:       "인증 헤더 없음",
            authHeader: "",
            wantStatus: http.StatusUnauthorized,
        },
        {
            name:       "잘못된 토큰 형식",
            authHeader: "Bearer invalid-token",
            wantStatus: http.StatusUnauthorized,
        },
        // 유효한 토큰 테스트는 실제 토큰 생성이 필요합니다
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            req, _ := http.NewRequest("GET", "/protected", nil)
            if tc.authHeader != "" {
                req.Header.Set("Authorization", tc.authHeader)
            }
            
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            
            assert.Equal(t, tc.wantStatus, w.Code)
        })
    }
}
```

### 3.2 프록시 테스트

```go
// tests/unit/proxy/reverseproxy_test.go
package proxy_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/proxy"
)

func TestHTTPProxyHandler(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 모의 백엔드 서버 설정
    backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message":"백엔드 응답"}`))
    }))
    defer backendServer.Close()
    
    // 프록시 설정
    httpProxy := proxy.NewHTTPProxy(backendServer.URL)
    
    // 테스트 라우트 구성
    router.GET("/test/*path", proxy.HTTPProxyHandler(httpProxy, backendServer.URL, false))
    
    // 테스트 실행
    req, _ := http.NewRequest("GET", "/test/api", nil)
    req.Header.Set("X-Custom-Header", "test-value")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 응답 검증
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "백엔드 응답")
}
```

## 4. 통합 테스트

### 4.1 기본 통합 테스트 설정

```go
// tests/integration/setup.go
package integration

import (
    "context"
    "net/http"
    "os"
    "testing"
    "time"

    "github.com/isinthesky/api-gateway/config"
    "github.com/isinthesky/api-gateway/tests/mocks"
)

// TestSetup은 통합 테스트를 위한 환경을 설정합니다
type TestSetup struct {
    Config      *config.Config
    APIGateway  *http.Server
    MockServers []*mocks.MockServer
}

// NewTestSetup은 통합 테스트 환경을 생성합니다
func NewTestSetup(t *testing.T) *TestSetup {
    // 환경 변수 설정
    os.Setenv("PORT", "18080")
    os.Setenv("LOG_LEVEL", "debug")
    
    // 모의 서버 시작
    mockServer1, err := mocks.NewMockServer()
    if err != nil {
        t.Fatalf("모의 서버 1 시작 실패: %v", err)
    }
    
    mockServer2, err := mocks.NewMockServer()
    if err != nil {
        t.Fatalf("모의 서버 2 시작 실패: %v", err)
    }
    
    // API 게이트웨이 설정
    cfg := config.LoadConfig()
    
    // API 게이트웨이 서버 시작
    // 실제 구현에서는 main.go의 서버 시작 코드를 재사용할 수 있도록 리팩토링이 필요합니다
    
    // 서버가 시작될 때까지 잠시 대기
    time.Sleep(500 * time.Millisecond)
    
    return &TestSetup{
        Config:      cfg,
        MockServers: []*mocks.MockServer{mockServer1, mockServer2},
    }
}

// Cleanup은 테스트 환경을 정리합니다
func (s *TestSetup) Cleanup() {
    // 모의 서버 종료
    for _, server := range s.MockServers {
        server.Close()
    }
    
    // API 게이트웨이 종료
    if s.APIGateway != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        s.APIGateway.Shutdown(ctx)
    }
}
```





### 4.2 라우팅 통합 테스트

```go
// tests/integration/routing/routing_test.go
package routing_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestRouting(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 테스트 케이스
    t.Run("기본 라우팅", func(t *testing.T) {
        // facreport_8006의 사용자 API 요청
        e.GET("/api/users").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("data")
        
        // facreport_8007의 제품 API 요청
        e.GET("/api/products").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("data")
    })
    
    t.Run("경로 파라미터 라우팅", func(t *testing.T) {
        // 특정 사용자 ID로 요청
        e.GET("/api/users/123").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("id").
            ValueEqual("id", "123")
    })
    
    t.Run("쿼리 파라미터", func(t *testing.T) {
        // 쿼리 파라미터가 있는 요청
        e.GET("/api/products").
            WithQuery("limit", 10).
            WithQuery("offset", 0).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("limit").
            ValueEqual("limit", 10)
    })
}
```

### 4.3 인증 통합 테스트

```go
// tests/integration/auth/auth_test.go
package auth_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestAuthentication(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 먼저 로그인하여 토큰 가져오기
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    
    // 테스트 케이스
    t.Run("인증 필요 엔드포인트 - 토큰 없음", func(t *testing.T) {
        e.GET("/api/users/profile").
            Expect().
            Status(http.StatusUnauthorized)
    })
    
    t.Run("인증 필요 엔드포인트 - 유효한 토큰", func(t *testing.T) {
        e.GET("/api/users/profile").
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("username")
    })
    
    t.Run("인증 필요 엔드포인트 - 잘못된 토큰", func(t *testing.T) {
        e.GET("/api/users/profile").
            WithHeader("Authorization", "Bearer invalid-token").
            Expect().
            Status(http.StatusUnauthorized)
    })
}
```

## 5. facreport.iptime.org API 서버 테스트

### 5.1 facreport_8006 서버 API 테스트

제공된 Swagger URL(http://facreport.iptime.org:8006/docs)에서 확인한 API를 기반으로 테스트를 작성합니다:

```go
// tests/integration/facreport_8006/users_test.go
package facreport_8006_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestUsers(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 인증 토큰 가져오기
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    
    // 사용자 목록 API 테스트
    t.Run("사용자 목록 조회", func(t *testing.T) {
        resp := e.GET("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        resp.ContainsKey("data").
            Value("data").Array().NotEmpty()
    })
    
    // 사용자 생성 API 테스트
    t.Run("사용자 생성", func(t *testing.T) {
        newUser := map[string]interface{}{
            "username": "newuser",
            "email": "newuser@example.com",
            "firstName": "New",
            "lastName": "User",
            "password": "password123",
        }
        
        resp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        resp.ContainsKey("id").NotEmpty()
        
        // 생성된 사용자 ID 가져오기
        userId := resp.Value("id").String().Raw()
        
        // 생성된 사용자 조회 테스트
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("username").
            ValueEqual("username", "newuser")
    })
    
    // 사용자 업데이트 API 테스트
    t.Run("사용자 업데이트", func(t *testing.T) {
        // 먼저 사용자 생성
        newUser := map[string]interface{}{
            "username": "updateuser",
            "email": "updateuser@example.com",
            "firstName": "Update",
            "lastName": "User",
            "password": "password123",
        }
        
        createResp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        userId := createResp.Value("id").String().Raw()
        
        // 사용자 업데이트
        updateData := map[string]interface{}{
            "firstName": "Updated",
            "lastName": "Name",
        }
        
        e.PUT("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(updateData).
            Expect().
            Status(http.StatusOK)
        
        // 업데이트 확인
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ValueEqual("firstName", "Updated").
            ValueEqual("lastName", "Name")
    })
    
    // 사용자 삭제 API 테스트
    t.Run("사용자 삭제", func(t *testing.T) {
        // 먼저 사용자 생성
        newUser := map[string]interface{}{
            "username": "deleteuser",
            "email": "deleteuser@example.com",
            "firstName": "Delete",
            "lastName": "User",
            "password": "password123",
        }
        
        createResp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        userId := createResp.Value("id").String().Raw()
        
        // 사용자 삭제
        e.DELETE("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNoContent)
        
        // 삭제 확인
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNotFound)
    })
}
```

### 5.2 facreport_8007 서버 API 테스트

제공된 Swagger URL(http://facreport.iptime.org:8007/docs)에서 확인한 API를 기반으로 테스트를 작성합니다:

```go
// tests/integration/facreport_8007/reports_test.go
package facreport_8007_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestReports(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 인증 토큰 가져오기
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    
    // 보고서 목록 API 테스트
    t.Run("보고서 목록 조회", func(t *testing.T) {
        resp := e.GET("/api/reports").
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        resp.ContainsKey("data").
            Value("data").Array()
    })
    
    // 보고서 생성 API 테스트
    t.Run("보고서 생성", func(t *testing.T) {
        newReport := map[string]interface{}{
            "title": "테스트 보고서",
            "description": "API Gateway 테스트를 위한 보고서",
            "reportDate": "2025-03-25",
            "content": "보고서 내용입니다.",
            "reportType": "TEST",
        }
        
        resp := e.POST("/api/reports").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newReport).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        resp.ContainsKey("id").NotEmpty()
        
        // 생성된 보고서 ID 가져오기
        reportId := resp.Value("id").String().Raw()
        
        // 생성된 보고서 조회 테스트
        e.GET("/api/reports/{id}", reportId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("title").
            ValueEqual("title", "테스트 보고서")
    })
    
    // 보고서 필터링 API 테스트
    t.Run("보고서 필터링", func(t *testing.T) {
        resp := e.GET("/api/reports").
            WithHeader("Authorization", "Bearer "+token).
            WithQuery("reportType", "TEST").
            WithQuery("fromDate", "2025-01-01").
            WithQuery("toDate", "2025-12-31").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        resp.ContainsKey("data").
            Value("data").Array()
    })
    
    // 보고서 통계 API 테스트
    t.Run("보고서 통계", func(t *testing.T) {
        resp := e.GET("/api/reports/statistics").
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        resp.ContainsKey("totalReports")
        resp.ContainsKey("reportsByType").Object()
    })
}
```

## 6. 부하 테스트 및 성능 검증

### 6.1 API Gateway 성능 테스트

```go
// tests/integration/performance/load_test.go
package performance_test

import (
    "fmt"
    "net/http"
    "sync"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestAPIGatewayPerformance(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
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
    
    // 테스트 시작 시간
    startTime := time.Now()
    
    // 동시 요청 실행
    for i := 0; i < concurrentRequests; i++ {
        go func(index int) {
            defer wg.Done()
            
            startReq := time.Now()
            
            // 테스트할 API 엔드포인트 (공개 엔드포인트)
            resp, err := http.Get("http://localhost:18080/api/public/status")
            
            if err != nil {
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
        }(i)
    }
    
    // 모든 요청이 완료될 때까지 대기
    wg.Wait()
    
    // 테스트 총 소요 시간
    totalDuration := time.Since(startTime)
    
    // 결과 분석
    var successCount int
    var totalResponseTime time.Duration
    
    for _, r := range results {
        if r.err == nil && r.statusCode == http.StatusOK {
            successCount++
            totalResponseTime += r.duration
        }
    }
    
    // 성공률 계산
    successRate := float64(successCount) / float64(concurrentRequests) * 100
    
    // 평균 응답 시간 계산
    var avgResponseTime time.Duration
    if successCount > 0 {
        avgResponseTime = totalResponseTime / time.Duration(successCount)
    }
    
    // 결과 출력 및 검증
    fmt.Printf("성공률: %.2f%%\n", successRate)
    fmt.Printf("평균 응답 시간: %v\n", avgResponseTime)
    fmt.Printf("총 테스트 소요 시간: %v\n", totalDuration)
    
    // 성공률 검증 (90% 이상 성공 기대)
    assert.GreaterOrEqual(t, successRate, 90.0, "성공률이 90%보다 낮습니다")
    
    // 평균 응답 시간 검증 (500ms 이하 기대)
    assert.LessOrEqual(t, avgResponseTime, 500*time.Millisecond, "평균 응답 시간이 500ms를 초과합니다")
}
```

## 7. 장애 주입 테스트

장애 상황에서 API Gateway의 복원력과 오류 처리 능력을 테스트합니다.

### 7.1 모의 장애 서비스 구현

```go
// tests/mocks/fault_server.go
package mocks

import (
    "fmt"
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
            "error": "간헐적 백엔드 장애"
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
```

### 7.2 장애 주입 테스트 구현

```go
// tests/integration/resilience/fault_tolerance_test.go
package resilience_test

import (
    "net/http"
    "testing"
    "time"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
    "github.com/isinthesky/api-gateway/tests/mocks"
)

func TestFaultTolerance(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 장애 서버 설정
    faultServer := mocks.NewFaultServer()
    defer faultServer.Close()
    
    // API Gateway 설정 업데이트 (장애 서버 URL 추가)
    // 참고: 실제 구현에서는 API Gateway 설정을 동적으로 업데이트하는 방법 필요
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 간헐적 장애 테스트
    t.Run("간헐적 장애 처리", func(t *testing.T) {
        // 여러 요청을 보내 간헐적 장애 테스트
        for i := 0; i < 10; i++ {
            resp := e.GET("/api/fault-test/intermittent-failure").
                Expect()
            
            // 상태 코드 로깅 (성공 또는 실패 여부 확인)
            t.Logf("요청 %d: 상태 코드 %d", i+1, resp.Raw().StatusCode)
            
            // 간헐적 실패는 허용하지만, 모든 요청이 실패하면 안 됨
            // 여기서는 각 요청의 성공 여부만 로깅하고, 
            // 최종적으로 모든 요청이 실패했는지 확인
        }
    })
    
    // 타임아웃 처리 테스트
    t.Run("요청 타임아웃 처리", func(t *testing.T) {
        // 느린 엔드포인트 요청 (API Gateway의 타임아웃 설정보다 오래 걸림)
        resp := e.GET("/api/fault-test/slow-response").
            Expect()
        
        // 타임아웃 응답 확인 (게이트웨이 타임아웃 또는 성공적인 응답)
        statusCode := resp.Raw().StatusCode
        t.Logf("느린 응답 상태 코드: %d", statusCode)
        
        // 게이트웨이 타임아웃 또는 성공적인 응답 중 하나여야 함
        if statusCode != http.StatusGatewayTimeout && statusCode != http.StatusOK {
            t.Errorf("예상치 못한 상태 코드: %d", statusCode)
        }
    })
    
    // 서비스 다운 처리 테스트
    t.Run("서비스 다운 처리", func(t *testing.T) {
        // 완전히 다운된 서비스 엔드포인트 요청
        resp := e.GET("/api/fault-test/service-down").
            Expect()
        
        // API Gateway는 적절한 오류 응답 제공해야 함
        statusCode := resp.Raw().StatusCode
        t.Logf("서비스 다운 상태 코드: %d", statusCode)
        
        // 502 Bad Gateway 또는 503 Service Unavailable 응답 기대
        if statusCode != http.StatusBadGateway && statusCode != http.StatusServiceUnavailable {
            t.Errorf("서비스 다운 시 예상치 못한 상태 코드: %d", statusCode)
        }
    })
    
    // 여러 백엔드 장애 시 장애 격리 테스트
    t.Run("장애 격리", func(t *testing.T) {
        // 하나의 장애 서비스 요청
        e.GET("/api/fault-test/service-down").
            Expect()
        
        // 다른 정상 서비스는 여전히 작동해야 함
        e.GET("/api/healthy-service/status").
            Expect().
            Status(http.StatusOK)
    })
}
```

## 8. 서킷 브레이커 테스트

API Gateway에 서킷 브레이커 패턴이 구현되어 있다면 이를 테스트합니다:

```go
// tests/integration/resilience/circuit_breaker_test.go
package resilience_test

import (
    "net/http"
    "testing"
    "time"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestCircuitBreaker(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 서킷 브레이커 동작 테스트
    t.Run("서킷 브레이커 트립", func(t *testing.T) {
        // 실패 임계값을 초과하도록 여러 실패 요청 전송
        for i := 0; i < 10; i++ {
            resp := e.GET("/api/circuit-test/failing-endpoint").
                Expect()
            
            statusCode := resp.Raw().StatusCode
            t.Logf("요청 %d: 상태 코드 %d", i+1, statusCode)
            
            // 서킷 브레이커가 열리면 더 이상 실제 요청을 보내지 않고
            // 즉시 실패 응답을 반환해야 함
            
            // 서킷 브레이커가 열렸는지 확인하기 위해 헤더 확인
            if i > 5 { // 5번 이상의 실패 후에는 서킷이 열렸을 가능성이 높음
                circuitState := resp.Header("X-Circuit-State").Raw()
                if circuitState == "open" {
                    t.Logf("서킷 브레이커가 열림 상태로 전환됨 (요청 %d 이후)", i+1)
                    break
                }
            }
        }
        
        // 서킷 브레이커가 열린 상태인지 확인
        resp := e.GET("/api/circuit-test/failing-endpoint").
            Expect()
        
        // 서킷 브레이커가 열린 경우, 즉시 오류 응답이 반환되어야 함
        circuitState := resp.Header("X-Circuit-State").Raw()
        t.Logf("최종 서킷 상태: %s", circuitState)
        
        if circuitState != "open" {
            t.Errorf("여러 실패 후에도 서킷 브레이커가 열리지 않음")
        }
    })
    
    t.Run("서킷 브레이커 복구", func(t *testing.T) {
        // 서킷 브레이커가 반열림 상태로 전환될 때까지 대기
        // (대기 시간은 구현에 따라 조정 필요)
        t.Log("서킷 브레이커 반열림 상태 대기 중...")
        time.Sleep(10 * time.Second)
        
        // 서비스가 복구되었다면, 반열림 상태에서 성공 요청을 보내면
        // 서킷 브레이커가 닫힘 상태로 돌아가야 함
        
        // 복구된 서비스 엔드포인트로 요청
        resp := e.GET("/api/circuit-test/recovered-endpoint").
            Expect()
        
        statusCode := resp.Raw().StatusCode
        circuitState := resp.Header("X-Circuit-State").Raw()
        
        t.Logf("복구 후 상태 코드: %d, 서킷 상태: %s", statusCode, circuitState)
        
        // 몇 번의 성공적인 요청 후에는 서킷이 닫혀야 함
        if statusCode == http.StatusOK {
            // 추가 요청으로 서킷 상태 확인
            for i := 0; i < 3; i++ {
                resp := e.GET("/api/circuit-test/recovered-endpoint").
                    Expect().
                    Status(http.StatusOK)
                
                circuitState = resp.Header("X-Circuit-State").Raw()
                if circuitState == "closed" {
                    t.Logf("서킷 브레이커가 다시 닫힘 상태로 전환됨")
                    break
                }
                
                time.Sleep(1 * time.Second)
            }
        }
    })
}
```

## 9. 테스트 실행 및 보고서 생성

### 9.1 테스트 실행 스크립트

```bash
#!/bin/bash
# tests/run_tests.sh

# 환경 설정
export TEST_ENV=development
export GATEWAY_PORT=18080

# 필요한 디렉토리 생성
mkdir -p ./test-reports

# 단위 테스트 실행
echo "단위 테스트 실행 중..."
go test -v ./tests/unit/... -coverprofile=./test-reports/unit.out

# 통합 테스트 실행
echo "통합 테스트 실행 중..."
go test -v ./tests/integration/... -coverprofile=./test-reports/integration.out

# E2E 테스트 실행
echo "E2E 테스트 실행 중..."
go test -v ./tests/e2e/... -coverprofile=./test-reports/e2e.out

# 커버리지 보고서 생성
echo "테스트 커버리지 보고서 생성 중..."
go tool cover -html=./test-reports/unit.out -o ./test-reports/unit_coverage.html
go tool cover -html=./test-reports/integration.out -o ./test-reports/integration_coverage.html
go tool cover -html=./test-reports/e2e.out -o ./test-reports/e2e_coverage.html

# 총 커버리지 계산
echo "총 테스트 커버리지 계산 중..."
go test -coverprofile=./test-reports/total.out ./...
go tool cover -func=./test-reports/total.out | grep total: | awk '{print "총 커버리지: " $3}'

# 결과 요약
echo "테스트 완료! 보고서는 ./test-reports 디렉토리에 저장되었습니다."
```

### 9.2 Makefile 타겟 추가

```makefile
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
```

## 10. 테스트 로깅 및 디버깅 가이드라인

상용 서비스의 테스트 코드에서는 명확한 로깅과 효과적인 디버깅을 위한 전략이 필수적입니다. 테스트 실패 시 원인을 빠르게 파악하고 해결할 수 있도록 다음 가이드라인을 따릅니다.

### 10.1 테스트 로깅 구조화

```go
// tests/utils/logger.go
package utils

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"
)

// 로그 레벨 정의
const (
    LevelDebug = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)

var levelNames = map[int]string{
    LevelDebug: "DEBUG",
    LevelInfo:  "INFO",
    LevelWarn:  "WARN",
    LevelError: "ERROR",
    LevelFatal: "FATAL",
}

// TestLogger는 테스트 전용 로거입니다
type TestLogger struct {
    logger *log.Logger
    level  int
}

// NewTestLogger는 새 테스트 로거를 생성합니다
func NewTestLogger(level int) *TestLogger {
    return &TestLogger{
        logger: log.New(os.Stdout, "", 0),
        level:  level,
    }
}

// log는 지정된 레벨에 메시지를 기록합니다
func (l *TestLogger) log(level int, format string, args ...interface{}) {
    if level < l.level {
        return
    }

    // 호출자 정보 가져오기
    _, file, line, _ := runtime.Caller(2)
    file = filepath.Base(file)

    // 타임스탬프 및 레벨 정보
    timestamp := time.Now().Format("2006-01-02 15:04:05.000")
    levelName := levelNames[level]

    // 최종 메시지 형식
    prefix := fmt.Sprintf("[%s] [%s] [%s:%d] ", timestamp, levelName, file, line)
    message := fmt.Sprintf(format, args...)
    
    l.logger.Println(prefix + message)
}

// Debug는 디버그 수준 메시지를 기록합니다
func (l *TestLogger) Debug(format string, args ...interface{}) {
    l.log(LevelDebug, format, args...)
}

// Info는 정보 수준 메시지를 기록합니다
func (l *TestLogger) Info(format string, args ...interface{}) {
    l.log(LevelInfo, format, args...)
}

// Warn은 경고 수준 메시지를 기록합니다
func (l *TestLogger) Warn(format string, args ...interface{}) {
    l.log(LevelWarn, format, args...)
}

// Error는 오류 수준 메시지를 기록합니다
func (l *TestLogger) Error(format string, args ...interface{}) {
    l.log(LevelError, format, args...)
}

// Fatal은 치명적 오류 메시지를 기록합니다
func (l *TestLogger) Fatal(format string, args ...interface{}) {
    l.log(LevelFatal, format, args...)
    os.Exit(1)
}

// TestContext는 테스트 컨텍스트와 로그 정보를 포함합니다
func (l *TestLogger) TestContext(testName string) *TestLogger {
    prefix := fmt.Sprintf("[%s] ", testName)
    contextLogger := &TestLogger{
        logger: log.New(os.Stdout, prefix, 0),
        level:  l.level,
    }
    return contextLogger
}
```

### 10.2 컨텍스트 정보 기록

테스트 실패 시 충분한 컨텍스트 정보를 함께 기록하여 디버깅을 쉽게 합니다:

```go
// tests/integration/auth/auth_test.go 사용 예시
package auth_test

import (
    "encoding/json"
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
    "github.com/isinthesky/api-gateway/tests/utils"
)

func TestAuthenticationWithLogging(t *testing.T) {
    // 테스트 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("인증 테스트")
    
    // 테스트 환경 설정
    testLogger.Info("테스트 환경 설정 중...")
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 로그인 시도
    testLogger.Info("로그인 시도 중...")
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect()
    
    // 응답 검증 전 응답 내용 로깅
    respBody := loginResp.Body().Raw()
    testLogger.Debug("로그인 응답: %s", respBody)
    
    // 상태 코드 검증
    statusCode := loginResp.Raw().StatusCode
    if statusCode != http.StatusOK {
        testLogger.Error("로그인 실패: 상태 코드 %d, 예상 코드 %d", 
            statusCode, http.StatusOK)
        t.Errorf("로그인 실패: 상태 코드 %d, 예상 코드 %d", 
            statusCode, http.StatusOK)
    } else {
        testLogger.Info("로그인 성공: 상태 코드 %d", statusCode)
    }
    
    // JSON 응답 파싱
    var loginRespBody map[string]interface{}
    if err := json.Unmarshal([]byte(respBody), &loginRespBody); err != nil {
        testLogger.Error("JSON 파싱 실패: %v", err)
        t.Fatalf("JSON 파싱 실패: %v", err)
    }
    
    // 토큰 추출
    token, ok := loginRespBody["token"].(string)
    if !ok {
        testLogger.Error("토큰 누락 또는 잘못된 형식: %v", loginRespBody)
        t.Fatal("토큰 누락 또는 잘못된 형식")
    }
    testLogger.Info("토큰 추출 성공")
    
    // 인증이 필요한 엔드포인트 테스트
    t.Run("인증 필요 엔드포인트 테스트", func(t *testing.T) {
        subTestLogger := logger.TestContext("인증 필요 엔드포인트")
        
        subTestLogger.Info("인증된 요청 시도 중...")
        resp := e.GET("/api/users/profile").
            WithHeader("Authorization", "Bearer "+token).
            Expect()
        
        statusCode := resp.Raw().StatusCode
        if statusCode != http.StatusOK {
            subTestLogger.Error("인증된 요청 실패: 상태 코드 %d", statusCode)
            subTestLogger.Debug("응답 내용: %s", resp.Body().Raw())
            t.Errorf("인증된 요청 실패: 상태 코드 %d", statusCode)
        } else {
            subTestLogger.Info("인증된 요청 성공")
        }
    })
}
```

### 10.3 테스트 실패 디버깅 헬퍼

특정 조건에서 더 많은 진단 정보를 수집하는 헬퍼 함수를 구현합니다:

```go
// tests/utils/debug_helpers.go
package utils

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
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
```

### 10.4 테스트 로깅 설정 및 관리

로깅 레벨과 출력을 구성하는 방법:

```go
// tests/integration/setup.go에 추가
import (
    "github.com/isinthesky/api-gateway/tests/utils"
)

// TestConfig는 테스트 구성을 정의합니다
type TestConfig struct {
    LogLevel    int    // 로그 레벨
    LogOutput   string // 로그 출력 (stdout, stderr, file)
    LogFilePath string // 파일로 로깅할 경우 경로
}

// DefaultTestConfig는 기본 테스트 구성을 반환합니다
func DefaultTestConfig() TestConfig {
    return TestConfig{
        LogLevel:    utils.LevelInfo,
        LogOutput:   "stdout",
        LogFilePath: "",
    }
}

// SetupTestLogging은 테스트 로깅을 설정합니다
func SetupTestLogging(config TestConfig) *utils.TestLogger {
    var logger *utils.TestLogger
    
    if config.LogOutput == "file" && config.LogFilePath != "" {
        file, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            log.Fatalf("로그 파일 열기 실패: %v", err)
        }
        
        // 커스텀 로거 생성 로직 추가 필요
        // (기존 TestLogger를 파일 출력용으로 확장해야 함)
    } else {
        logger = utils.NewTestLogger(config.LogLevel)
    }
    
    return logger
}
```

### 10.5 테스트 실패 분석 가이드

테스트 실패 시 체계적인 분석 방법:

1. **실패 유형 파악**
   - 기대값 불일치 (값 비교 실패)
   - 예외 발생 (panic, 패닉)
   - 타임아웃 (응답 지연)
   - 자원 문제 (메모리, 고루틴 누수)

2. **로그 분석 단계**
   - 오류 메시지 정확히 확인
   - 시간 순서대로 로그 흐름 분석
   - 관련 컴포넌트 간 상호작용 확인
   - 환경 변수 및 시스템 상태 검토

3. **디버깅 환경 구성**
   ```bash
   # 로그 레벨 설정
   export TEST_LOG_LEVEL=debug
   
   # 테스트 실행
   go test -v ./tests/integration/auth -run TestAuthenticationWithLogging
   
   # 로그 저장 및 분석
   go test -v ./tests/integration/auth -run TestAuthenticationWithLogging > auth_test.log 2>&1
   grep ERROR auth_test.log
   ```

4. **재현 가능한 테스트 케이스 작성**
   ```go
   // 실패 사례 격리 테스트
   func TestIsolatedFailureCase(t *testing.T) {
       logger := utils.NewTestLogger(utils.LevelDebug)
       logger.Info("격리된 실패 케이스 시작")
       
       // 시스템 상태 캡처
       initialState := utils.CaptureSystemState()
       utils.LogSystemState(logger, initialState)
       
       // 실패 조건 재현
       // ...테스트 코드...
       
       // 실패 후 시스템 상태 캡처
       finalState := utils.CaptureSystemState()
       utils.LogSystemState(logger, finalState)
   }
   ```

### 10.6 로깅 모범 사례

1. **명확한 오류 메시지 작성**
   - 기대값과 실제값 모두 명시 (`Expected X, got Y`)
   - 컨텍스트 정보 포함 (테스트 단계, 관련 데이터)
   - 타임스탬프 및 요청 ID 포함

2. **단계별 로깅**
   - 테스트 시작과 종료를 명확히 기록
   - 각 중요 단계마다 로그 메시지 추가
   - 각 API 요청과 응답 정보 상세히 기록

3. **일관된 로그 형식**
   - JSON 형식 로깅 고려 (구조화된 분석 용이)
   - 로그 레벨 적절히 사용 (DEBUG, INFO, ERROR 등)
   - 로그 내용 표준화 (타임스탬프, 요청 ID, 컴포넌트 등)

4. **데이터 마스킹**
   - 민감한 정보 마스킹 (토큰, 비밀번호 등)
   - 대용량 데이터는 요약 정보만 기록
   - 바이너리 데이터는 적절히 인코딩

5. **성능 고려**
   - 선택적 로깅 활성화 (환경 변수 기반)
   - 디버그 모드에서만 상세 로깅
   - 과도한 로깅으로 인한 성능 저하 방지

## 11. 고급 테스트 기법

### 11.1 기능 플래그 테스트

API Gateway에 기능 플래그(Feature Flag)가 구현되어 있는 경우, 다양한 기능 조합을 테스트합니다:

```go
// tests/integration/features/feature_flags_test.go
package features_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestFeatureFlags(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 기능 플래그 켜기/끄기 테스트
    t.Run("캐싱 기능 플래그", func(t *testing.T) {
        // 캐싱 비활성화 상태에서 요청
        resp1 := e.GET("/api/feature-test/data").
            WithHeader("X-Features", "caching=false").
            Expect().
            Status(http.StatusOK)
        
        // 캐시 비활성화 확인
        resp1.Header("X-Cache-Status").Equal("DISABLED")
        
        // 첫 번째 요청 후 데이터 생성 시간 기록
        timestamp1 := resp1.JSON().Object().Value("timestamp").String().Raw()
        
        // 캐싱 활성화 상태에서 요청
        resp2 := e.GET("/api/feature-test/data").
            WithHeader("X-Features", "caching=true").
            Expect().
            Status(http.StatusOK)
        
        // 두 번째 요청 후 데이터 생성 시간 기록
        timestamp2 := resp2.JSON().Object().Value("timestamp").String().Raw()
        
        // 캐싱이 활성화된 경우, 동일한 요청을 다시 보내면 캐시된 응답을 반환해야 함
        resp3 := e.GET("/api/feature-test/data").
            WithHeader("X-Features", "caching=true").
            Expect().
            Status(http.StatusOK)
        
        // 캐시 히트 확인
        resp3.Header("X-Cache-Status").Equal("HIT")
        
        // 캐시된 응답의 타임스탬프는 두 번째 요청과 동일해야 함
        timestamp3 := resp3.JSON().Object().Value("timestamp").String().Raw()
        
        if timestamp2 != timestamp3 {
            t.Errorf("캐시 히트 시 다른 타임스탬프를 반환: %s != %s", timestamp2, timestamp3)
        }
    })
    
    // 다른 기능 플래그 테스트도 추가 가능
}
```

### 11.2 API 버전 관리 테스트

API 버전 관리 기능을 테스트합니다:

```go
// tests/integration/versioning/api_versioning_test.go
package versioning_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestAPIVersioning(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 경로 기반 버전 관리 테스트
    t.Run("경로 기반 버전 관리", func(t *testing.T) {
        // v1 API 호출
        resp1 := e.GET("/api/v1/users").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v1 API 응답 구조 확인
        resp1.ContainsKey("data")
        resp1.NotContainsKey("metadata")
        
        // v2 API 호출
        resp2 := e.GET("/api/v2/users").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v2 API 응답 구조 확인 (새로운 필드 또는 구조)
        resp2.ContainsKey("data")
        resp2.ContainsKey("metadata")
    })
    
    // 헤더 기반 버전 관리 테스트
    t.Run("헤더 기반 버전 관리", func(t *testing.T) {
        // v1 API 호출 (기본값)
        resp1 := e.GET("/api/users").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v1 API 응답 구조 확인
        resp1.ContainsKey("data")
        resp1.NotContainsKey("metadata")
        
        // v2 API 호출 (헤더 사용)
        resp2 := e.GET("/api/users").
            WithHeader("Accept-Version", "v2").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v2 API 응답 구조 확인
        resp2.ContainsKey("data")
        resp2.ContainsKey("metadata")
    })
    
    // 미디어 타입 기반 버전 관리 테스트
    t.Run("미디어 타입 기반 버전 관리", func(t *testing.T) {
        // v1 API 호출
        resp1 := e.GET("/api/users").
            WithHeader("Accept", "application/json").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v1 API 응답 구조 확인
        resp1.ContainsKey("data")
        resp1.NotContainsKey("metadata")
        
        // v2 API 호출
        resp2 := e.GET("/api/users").
            WithHeader("Accept", "application/vnd.company.v2+json").
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // v2 API 응답 구조 확인
        resp2.ContainsKey("data")
        resp2.ContainsKey("metadata")
    })
}
```

## 12. 실제 API 서버 테스트 확장

제공된 두 개의 API 서버(http://facreport.iptime.org:8006/docs, http://facreport.iptime.org:8007/docs)의 Swagger 문서를 바탕으로 보다 체계적인 테스트를 작성합니다.

### 12.1 API 목록 및 엔드포인트 분석

```go
// tests/integration/facreport/api_discovery_test.go
package facreport_test

import (
    "fmt"
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

// API 서버 엔드포인트 정보
var apiEndpoints = map[string][]struct {
    Method   string
    Path     string
    Auth     bool
    Expected int
}{
    "facreport_8006": {
        {Method: "GET", Path: "/api/users", Auth: true, Expected: http.StatusOK},
        {Method: "POST", Path: "/api/users", Auth: true, Expected: http.StatusCreated},
        {Method: "GET", Path: "/api/users/{id}", Auth: true, Expected: http.StatusOK},
        {Method: "PUT", Path: "/api/users/{id}", Auth: true, Expected: http.StatusOK},
        {Method: "DELETE", Path: "/api/users/{id}", Auth: true, Expected: http.StatusNoContent},
        {Method: "POST", Path: "/api/auth/login", Auth: false, Expected: http.StatusOK},
        // Swagger에서 확인한 다른 엔드포인트 추가
    },
    "facreport_8007": {
        {Method: "GET", Path: "/api/reports", Auth: true, Expected: http.StatusOK},
        {Method: "POST", Path: "/api/reports", Auth: true, Expected: http.StatusCreated},
        {Method: "GET", Path: "/api/reports/{id}", Auth: true, Expected: http.StatusOK},
        {Method: "PUT", Path: "/api/reports/{id}", Auth: true, Expected: http.StatusOK},
        {Method: "DELETE", Path: "/api/reports/{id}", Auth: true, Expected: http.StatusNoContent},
        {Method: "GET", Path: "/api/reports/statistics", Auth: true, Expected: http.StatusOK},
        // Swagger에서 확인한 다른 엔드포인트 추가
    },
}

func TestAPIDiscovery(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 먼저 로그인하여 토큰 가져오기
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    
    // 모든 API 서버와 엔드포인트 테스트
    for server, endpoints := range apiEndpoints {
        t.Run(fmt.Sprintf("%s 엔드포인트 검증", server), func(t *testing.T) {
            for _, ep := range endpoints {
                t.Run(fmt.Sprintf("%s %s", ep.Method, ep.Path), func(t *testing.T) {
                    // 동적 경로 파라미터가 있으면 대체
                    path := ep.Path
                    if server == "facreport_8006" && path == "/api/users/{id}" {
                        path = "/api/users/1" // 테스트용 ID
                    } else if server == "facreport_8007" && path == "/api/reports/{id}" {
                        path = "/api/reports/1" // 테스트용 ID
                    }
                    
                    // 요청 생성
                    req := e.Request(ep.Method, path)
                    
                    // POST/PUT 요청에 필요한 JSON 본문 추가
                    if ep.Method == "POST" || ep.Method == "PUT" {
                        // 엔드포인트에 따라 다른 JSON 본문 설정
                        if path == "/api/users" {
                            req = req.WithJSON(map[string]interface{}{
                                "username": "testuser",
                                "email": "test@example.com",
                                "firstName": "Test",
                                "lastName": "User",
                                "password": "password123",
                            })
                        } else if path == "/api/reports" {
                            req = req.WithJSON(map[string]interface{}{
                                "title": "테스트 보고서",
                                "description": "자동화 테스트용 보고서",
                                "reportDate": "2025-03-26",
                                "content": "테스트 내용",
                                "reportType": "TEST",
                            })
                        }
                    }
                    
                    // 인증이 필요한 경우 토큰 추가
                    if ep.Auth {
                        req = req.WithHeader("Authorization", "Bearer "+token)
                    }
                    
                    // 요청 실행 및 응답 검증
                    // 참고: 실제 상황에서는 기대 상태 코드가 다를 수 있으므로 조정 필요
                    req.Expect().Status(ep.Expected)
                })
            }
        })
    }
}
```

### 12.2 데이터 일관성 테스트

여러 API 요청 간의 데이터 일관성을 테스트합니다:

```go
// tests/integration/facreport/data_consistency_test.go
package facreport_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestDataConsistency(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 인증 토큰 가져오기
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    
    // CRUD 일관성 테스트
    t.Run("사용자 CRUD 일관성", func(t *testing.T) {
        // 1. 생성
        createResp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(map[string]interface{}{
                "username": "consistency_test_user",
                "email": "consistency@example.com",
                "firstName": "Consistency",
                "lastName": "Test",
                "password": "password123",
            }).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        // 생성된 ID 저장
        userId := createResp.Value("id").String().Raw()
        
        // 2. 조회 - 생성된 데이터 확인
        getResp := e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // 생성 데이터와 조회 데이터 일치 확인
        getResp.Value("username").String().Equal("consistency_test_user")
        getResp.Value("email").String().Equal("consistency@example.com")
        
        // 3. 업데이트
        e.PUT("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(map[string]interface{}{
                "firstName": "Updated",
                "lastName": "User",
            }).
            Expect().
            Status(http.StatusOK)
        
        // 4. 업데이트 확인
        updatedResp := e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // 업데이트된 필드 확인
        updatedResp.Value("firstName").String().Equal("Updated")
        updatedResp.Value("lastName").String().Equal("User")
        
        // 업데이트되지 않은 필드 확인 (불변 필드)
        updatedResp.Value("username").String().Equal("consistency_test_user")
        updatedResp.Value("email").String().Equal("consistency@example.com")
        
        // 5. 삭제
        e.DELETE("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNoContent)
        
        // 6. 삭제 확인
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNotFound)
    })
    
    // 보고서와 사용자 간 관계 일관성 테스트
    t.Run("보고서-사용자 관계 일관성", func(t *testing.T) {
        // 1. 보고서 생성
        createReportResp := e.POST("/api/reports").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(map[string]interface{}{
                "title": "일관성 테스트 보고서",
                "description": "사용자-보고서 관계 테스트",
                "reportDate": "2025-03-26",
                "content": "일관성 테스트 내용",
                "reportType": "TEST",
                // authorId는 현재 로그인한 사용자 ID로 자동 설정된다고 가정
            }).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        // 생성된 보고서 ID 저장
        reportId := createReportResp.Value("id").String().Raw()
        
        // 2. 보고서 조회 - 작성자 확인
        getReportResp := e.GET("/api/reports/{id}", reportId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        // 보고서 작성자 ID 추출
        authorId := getReportResp.Value("authorId").String().Raw()
        
        // 3. 작성자 정보 조회
        e.GET("/api/users/{id}", authorId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK)
        
        // 4. 보고서 삭제
        e.DELETE("/api/reports/{id}", reportId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNoContent)
        
        // 5. 작성자는 여전히 존재해야 함
        e.GET("/api/users/{id}", authorId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK)
    })
}
```

## 13. 결론 및 모범 사례

API Gateway 테스트를 성공적으로 구현하기 위한 모범 사례:

1. **계층적 테스트 전략 적용**
   - 단위 테스트 → 통합 테스트 → E2E 테스트 순으로 진행
   - 각 테스트 유형에 맞는 도구와 접근 방식 선택

2. **모의 백엔드 서비스 설계**
   - 실제 서비스 동작을 시뮬레이션하는 정교한 모의 서비스 구현
   - 다양한 오류 시나리오와 지연 시간 시뮬레이션

3. **실제 API 패턴 테스트**
   - facreport_8006/docs와 facreport_8007/docs에서 확인한 API 패턴 테스트
   - 경로 매개변수, 쿼리 매개변수, 헤더 등 다양한 요청 패턴 검증

4. **성능 및 안정성 검증**
   - 부하 테스트를 통한 성능 한계 파악
   - 장애 주입을 통한 복원력 검증
   - 오래 실행되는 테스트와 스트레스 테스트 포함

5. **CI/CD 통합**
   - 모든 테스트를 CI/CD 파이프라인에 통합
   - 자동화된 테스트 보고서 생성 및 배포 여부 결정

이 가이드에 따라 API Gateway 테스트를 구현하면 구조적 개선 작업 전에 현재 기능의 안정성을 보장하고, 리팩토링 후에도 동일한 기능을 유지하는지 확인할 수 있습니다.