package config_test

import (
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/config"
)

func TestLoadConfig(t *testing.T) {
    // 원래 환경 변수 저장
    originalPort := os.Getenv("PORT")
    originalBackendURL := os.Getenv("BACKEND_BASE_URL")
    originalLogLevel := os.Getenv("LOG_LEVEL")
    originalRoutesPath := os.Getenv("ROUTES_CONFIG_PATH")
    
    // 테스트 종료 후 원래 환경 변수 복원
    defer func() {
        os.Setenv("PORT", originalPort)
        os.Setenv("BACKEND_BASE_URL", originalBackendURL)
        os.Setenv("LOG_LEVEL", originalLogLevel)
        os.Setenv("ROUTES_CONFIG_PATH", originalRoutesPath)
    }()
    
    // 테스트를 위한 환경 변수 설정
    os.Setenv("PORT", "9090")
    os.Setenv("BACKEND_BASE_URL", "http://test-backend:8000")
    os.Setenv("LOG_LEVEL", "debug")
    os.Setenv("ROUTES_CONFIG_PATH", "./test-routes.json")
    os.Setenv("JWT_SECRET", "test-jwt-secret")
    os.Setenv("JWT_ISSUER", "test-issuer")
    os.Setenv("JWT_EXPIRATION_DELTA", "7200")
    os.Setenv("ALLOWED_ORIGINS", "http://example.com,https://test.com")
    os.Setenv("RATE_LIMIT_WINDOW", "60")
    os.Setenv("RATE_LIMIT_MAX_REQUESTS", "100")
    os.Setenv("READ_TIMEOUT", "30")
    os.Setenv("WRITE_TIMEOUT", "30")
    os.Setenv("IDLE_TIMEOUT", "60")
    os.Setenv("MAX_REQUEST_SIZE", "10")
    os.Setenv("ENABLE_METRICS", "true")
    
    // 설정 로드
    cfg := config.LoadConfig()
    
    // 설정값 검증
    assert.Equal(t, 9090, cfg.Port, "포트가 올바르게 로드되어야 함")
    assert.Equal(t, "http://test-backend:8000", cfg.BackendBaseURL, "백엔드 URL이 올바르게 로드되어야 함")
    assert.Equal(t, "debug", cfg.LogLevel, "로그 레벨이 올바르게 로드되어야 함")
    assert.Equal(t, "./test-routes.json", cfg.RoutesConfigPath, "라우트 설정 경로가 올바르게 로드되어야 함")
    assert.Equal(t, "test-jwt-secret", cfg.JWTSecret, "JWT 시크릿이 올바르게 로드되어야 함")
    assert.Equal(t, "test-issuer", cfg.JWTIssuer, "JWT 발급자가 올바르게 로드되어야 함")
    assert.Equal(t, 7200, cfg.JWTExpirationDelta, "JWT 만료 시간이 올바르게 로드되어야 함")
    assert.Equal(t, []string{"http://example.com", "https://test.com"}, cfg.AllowedOrigins, "허용된 출처가 올바르게 로드되어야 함")
    assert.Equal(t, 60*time.Second, cfg.RateLimitWindow, "비율 제한 윈도우가 올바르게 로드되어야 함")
    assert.Equal(t, 100, cfg.RateLimitMaxReqs, "최대 요청 수가 올바르게 로드되어야 함")
    assert.Equal(t, 30*time.Second, cfg.ReadTimeout, "읽기 타임아웃이 올바르게 로드되어야 함")
    assert.Equal(t, 30*time.Second, cfg.WriteTimeout, "쓰기 타임아웃이 올바르게 로드되어야 함")
    assert.Equal(t, 60*time.Second, cfg.IdleTimeout, "유휴 타임아웃이 올바르게 로드되어야 함")
    assert.Equal(t, int64(10*1024*1024), cfg.MaxRequestSize, "최대 요청 크기가 올바르게 로드되어야 함")
    assert.Equal(t, true, cfg.EnableMetrics, "메트릭 활성화 여부가 올바르게 로드되어야 함")
}

func TestDefaultConfig(t *testing.T) {
    // 모든 환경 변수 저장
    originalEnv := make(map[string]string)
    for _, key := range []string{
        "PORT", "BACKEND_BASE_URL", "LOG_LEVEL", "ROUTES_CONFIG_PATH",
        "JWT_SECRET", "JWT_ISSUER", "JWT_EXPIRATION_DELTA",
        "ALLOWED_ORIGINS", "RATE_LIMIT_WINDOW", "RATE_LIMIT_MAX_REQUESTS",
        "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
        "MAX_REQUEST_SIZE", "ENABLE_METRICS",
    } {
        originalEnv[key] = os.Getenv(key)
        os.Unsetenv(key)
    }
    
    // 테스트 종료 후 환경 변수 복원
    defer func() {
        for key, value := range originalEnv {
            if value != "" {
                os.Setenv(key, value)
            }
        }
    }()
    
    // 기본 설정 로드
    cfg := config.LoadConfig()
    
    // 기본값 검증
    assert.Equal(t, 8080, cfg.Port, "기본 포트는 8080이어야 함")
    assert.Equal(t, "http://localhost:8000", cfg.BackendBaseURL, "기본 백엔드 URL이 올바라야 함")
    assert.Equal(t, "info", cfg.LogLevel, "기본 로그 레벨은 info여야 함")
    assert.Equal(t, "./routes.json", cfg.RoutesConfigPath, "기본 라우트 설정 경로가 올바라야 함")
    assert.Equal(t, "your-secret-key", cfg.JWTSecret, "기본 JWT 시크릿이 올바라야 함")
    assert.Equal(t, "api-gateway", cfg.JWTIssuer, "기본 JWT 발급자가 올바라야 함")
    assert.Equal(t, 3600, cfg.JWTExpirationDelta, "기본 JWT 만료 시간이 올바라야 함")
    assert.Equal(t, []string{"*"}, cfg.AllowedOrigins, "기본적으로 모든 출처를 허용해야 함")
    assert.Equal(t, 60*time.Second, cfg.RateLimitWindow, "기본 비율 제한 윈도우가 올바라야 함")
    assert.Equal(t, 100, cfg.RateLimitMaxReqs, "기본 최대 요청 수가 올바라야 함")
    assert.Equal(t, 10*time.Second, cfg.ReadTimeout, "기본 읽기 타임아웃이 올바라야 함")
    assert.Equal(t, 10*time.Second, cfg.WriteTimeout, "기본 쓰기 타임아웃이 올바라야 함")
    assert.Equal(t, 30*time.Second, cfg.IdleTimeout, "기본 유휴 타임아웃이 올바라야 함")
    assert.Equal(t, int64(5*1024*1024), cfg.MaxRequestSize, "기본 최대 요청 크기가 올바라야 함")
    assert.Equal(t, false, cfg.EnableMetrics, "기본적으로 메트릭은 비활성화되어야 함")
}
