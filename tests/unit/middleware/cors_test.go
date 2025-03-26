package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/internal/middleware"
)

func TestCORSMiddleware(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 기본 CORS 미들웨어 등록 (모든 출처 허용)
    router.Use(middleware.CORSMiddleware())
    
    // 테스트 엔드포인트 설정
    router.GET("/api/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })
    
    // 테스트 케이스
    testCases := []struct {
        name           string
        origin         string
        method         string
        expectedOrigin string
        expectedStatus int
    }{
        {
            name:           "일반 GET 요청",
            origin:         "http://example.com",
            method:         "GET",
            expectedOrigin: "*", // 모든 출처 허용
            expectedStatus: http.StatusOK,
        },
        {
            name:           "다른 출처에서의 요청",
            origin:         "https://different-example.com",
            method:         "GET",
            expectedOrigin: "*", // 모든 출처 허용
            expectedStatus: http.StatusOK,
        },
        {
            name:           "프리플라이트 요청 (OPTIONS)",
            origin:         "http://example.com",
            method:         "OPTIONS",
            expectedOrigin: "*",
            expectedStatus: http.StatusNoContent, // 204 No Content
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            req, _ := http.NewRequest(tc.method, "/api/test", nil)
            req.Header.Set("Origin", tc.origin)
            
            // CORS 프리플라이트 요청 설정
            if tc.method == "OPTIONS" {
                req.Header.Set("Access-Control-Request-Method", "GET")
                req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
            }
            
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            
            // 상태 코드 확인
            assert.Equal(t, tc.expectedStatus, w.Code)
            
            // CORS 헤더 확인
            assert.Equal(t, tc.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
            assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
            
            // OPTIONS 요청인 경우 추가 헤더 확인
            if tc.method == "OPTIONS" {
                assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
                assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
                assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
                assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
            }
        })
    }
}

func TestCustomCORSMiddleware(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 사용자 정의 출처 목록으로 CORS 미들웨어 등록
    allowedOrigins := []string{
        "http://example.com",
        "https://test.example.com",
    }
    router.Use(middleware.CustomCORSMiddleware(allowedOrigins))
    
    // 테스트 엔드포인트 설정
    router.GET("/api/test", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "success"})
    })
    
    // 테스트 케이스
    testCases := []struct {
        name                string
        origin              string
        expectedOrigin      string
        expectedAllowOrigin bool
        expectedStatus      int
    }{
        {
            name:                "허용된 출처에서의 요청",
            origin:              "http://example.com",
            expectedOrigin:      "http://example.com",
            expectedAllowOrigin: true,
            expectedStatus:      http.StatusOK,
        },
        {
            name:                "다른 허용된 출처에서의 요청",
            origin:              "https://test.example.com",
            expectedOrigin:      "https://test.example.com",
            expectedAllowOrigin: true,
            expectedStatus:      http.StatusOK,
        },
        {
            name:                "허용되지 않은 출처에서의 요청",
            origin:              "https://unknown.com",
            expectedOrigin:      "",
            expectedAllowOrigin: false,
            expectedStatus:      http.StatusOK, // 요청은 처리되지만 CORS 헤더는 설정되지 않음
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            req, _ := http.NewRequest("GET", "/api/test", nil)
            req.Header.Set("Origin", tc.origin)
            
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            
            // 상태 코드 확인
            assert.Equal(t, tc.expectedStatus, w.Code)
            
            // CORS 헤더 확인
            if tc.expectedAllowOrigin {
                assert.Equal(t, tc.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
                assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
            } else {
                // 허용되지 않은 출처의 경우 CORS 헤더가 없어야 함
                assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
            }
        })
    }
}
