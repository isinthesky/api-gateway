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
        // 전달된 헤더 확인
        forwardedFor := r.Header.Get("X-Forwarded-For")
        assert.NotEmpty(t, forwardedFor, "X-Forwarded-For 헤더가 설정되어야 함")
        
        // 사용자 정의 헤더 확인
        customHeader := r.Header.Get("X-Custom-Header")
        assert.Equal(t, "test-value", customHeader, "사용자 정의 헤더가 전달되어야 함")
        
        // 경로 정보 확인
        assert.Equal(t, "/api", r.URL.Path, "요청 경로가 올바르게 전달되어야 함")
        
        // 백엔드 응답 설정
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
    assert.Equal(t, http.StatusOK, w.Code, "상태 코드가 일치해야 함")
    assert.Contains(t, w.Body.String(), "백엔드 응답", "응답 본문이 일치해야 함")
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "컨텐츠 타입 헤더가 일치해야 함")
}

func TestHTTPProxyHandlerWithStripPrefix(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 모의 백엔드 서버 설정
    backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 경로가 /api이 아닌 / 여야 함 (prefix가 제거되었으므로)
        assert.Equal(t, "/", r.URL.Path, "접두사가 제거된 경로여야 함")
        
        // 백엔드 응답 설정
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message":"접두사 제거 테스트 응답"}`))
    }))
    defer backendServer.Close()
    
    // 프록시 설정
    httpProxy := proxy.NewHTTPProxy(backendServer.URL)
    
    // stripPrefix=true로 라우트 구성
    router.GET("/api/*path", proxy.HTTPProxyHandler(httpProxy, backendServer.URL, true))
    
    // 테스트 실행
    req, _ := http.NewRequest("GET", "/api/", nil)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 응답 검증
    assert.Equal(t, http.StatusOK, w.Code, "상태 코드가 일치해야 함")
    assert.Contains(t, w.Body.String(), "접두사 제거 테스트 응답", "응답 본문이 일치해야 함")
}

func TestHTTPProxyHandlerWithQueryParams(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 모의 백엔드 서버 설정
    backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 쿼리 파라미터 확인
        queryParams := r.URL.Query()
        assert.Equal(t, "10", queryParams.Get("limit"), "limit 쿼리 파라미터가 전달되어야 함")
        assert.Equal(t, "0", queryParams.Get("offset"), "offset 쿼리 파라미터가 전달되어야 함")
        
        // 백엔드 응답 설정
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"message":"쿼리 파라미터 테스트 응답","limit":10,"offset":0}`))
    }))
    defer backendServer.Close()
    
    // 프록시 설정
    httpProxy := proxy.NewHTTPProxy(backendServer.URL)
    
    // 테스트 라우트 구성
    router.GET("/api/data", proxy.HTTPProxyHandler(httpProxy, backendServer.URL, false))
    
    // 쿼리 파라미터가 있는 요청
    req, _ := http.NewRequest("GET", "/api/data?limit=10&offset=0", nil)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 응답 검증
    assert.Equal(t, http.StatusOK, w.Code, "상태 코드가 일치해야 함")
    assert.Contains(t, w.Body.String(), "쿼리 파라미터 테스트 응답", "응답 본문이 일치해야 함")
    assert.Contains(t, w.Body.String(), `"limit":10`, "limit 값이 응답에 포함되어야 함")
}

func TestHTTPProxyHandlerWithBackendError(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 오류를 반환하는 모의 백엔드 서버 설정
    backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 500 Internal Server Error 반환
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(`{"error":"백엔드 서버 오류"}`))
    }))
    defer backendServer.Close()
    
    // 프록시 설정
    httpProxy := proxy.NewHTTPProxy(backendServer.URL)
    
    // 테스트 라우트 구성
    router.GET("/api/error", proxy.HTTPProxyHandler(httpProxy, backendServer.URL, false))
    
    // 테스트 실행
    req, _ := http.NewRequest("GET", "/api/error", nil)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 응답 검증 - 백엔드 오류가 클라이언트에게 그대로 전달되어야 함
    assert.Equal(t, http.StatusInternalServerError, w.Code, "백엔드 오류가 전달되어야 함")
    assert.Contains(t, w.Body.String(), "백엔드 서버 오류", "오류 메시지가 포함되어야 함")
}

func TestHTTPProxyHandlerWithNonexistentBackend(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 존재하지 않는 백엔드 URL
    nonexistentURL := "http://non-existent-backend:12345"
    
    // 프록시 설정
    httpProxy := proxy.NewHTTPProxy(nonexistentURL)
    
    // 테스트 라우트 구성
    router.GET("/api/non-existent", proxy.HTTPProxyHandler(httpProxy, nonexistentURL, false))
    
    // 테스트 실행
    req, _ := http.NewRequest("GET", "/api/non-existent", nil)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // 응답 검증 - 백엔드 연결 실패 시 502 Bad Gateway 예상
    assert.Equal(t, http.StatusBadGateway, w.Code, "연결 실패 시 502 상태 코드여야 함")
}
