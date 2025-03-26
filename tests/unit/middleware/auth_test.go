package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
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
    
    // 유효한 JWT 토큰 생성
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": "1234567890",
        "name": "Test User",
        "iss": "test-issuer",
        "iat": time.Now().Unix(),
        "exp": time.Now().Add(time.Hour).Unix(),
    })
    
    validToken, err := token.SignedString([]byte("test-secret-key"))
    if err != nil {
        t.Fatalf("토큰 생성 오류: %v", err)
    }
    
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
        {
            name:       "유효한 토큰",
            authHeader: "Bearer " + validToken,
            wantStatus: http.StatusOK,
        },
        {
            name:       "잘못된 헤더 형식",
            authHeader: "Token " + validToken,
            wantStatus: http.StatusUnauthorized,
        },
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

func TestAuthMiddlewareWithExpiredToken(t *testing.T) {
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
    
    // 만료된 JWT 토큰 생성
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub": "1234567890",
        "name": "Test User",
        "iss": "test-issuer",
        "iat": time.Now().Add(-2 * time.Hour).Unix(),
        "exp": time.Now().Add(-1 * time.Hour).Unix(), // 1시간 전에 만료
    })
    
    expiredToken, err := token.SignedString([]byte("test-secret-key"))
    if err != nil {
        t.Fatalf("토큰 생성 오류: %v", err)
    }
    
    // 만료된 토큰으로 요청
    req, _ := http.NewRequest("GET", "/protected", nil)
    req.Header.Set("Authorization", "Bearer "+expiredToken)
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusUnauthorized, w.Code)
    assert.Contains(t, w.Body.String(), "토큰이 만료되었습니다")
}
