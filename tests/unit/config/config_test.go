// +build unit

package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/isinthesky/api-gateway/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	// 환경 변수 초기화
	os.Clearenv()

	t.Run("Default Values", func(t *testing.T) {
		// 기본값 테스트
		cfg, err := config.Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 기본값 확인
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, "http://localhost:8081", cfg.DefaultBackend)
		assert.Equal(t, "your-secret-key", cfg.JWTSecret)
		assert.Equal(t, "api-gateway", cfg.JWTIssuer)
		assert.Equal(t, 3600*time.Second, cfg.JWTExpirationDelta)
		assert.Equal(t, []string{"*"}, cfg.AllowedOrigins)
		assert.Equal(t, true, cfg.EnableMetrics)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, int64(10*1024*1024), cfg.MaxContentSize) // 10MB
		assert.Equal(t, 20*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 20*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 120*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 60*time.Second, cfg.RateLimitWindow)
		assert.Equal(t, 200, cfg.RateLimitMaxReqs)
		assert.Equal(t, "configs/routes.json", cfg.RoutesConfigPath)
	})

	t.Run("Custom Values", func(t *testing.T) {
		// 환경 변수 설정
		os.Setenv("PORT", "9090")
		os.Setenv("BACKEND_URL", "http://custom-backend:8000")
		os.Setenv("JWT_SECRET", "custom-secret-key")
		os.Setenv("JWT_ISSUER", "custom-issuer")
		os.Setenv("JWT_EXPIRATION", "7200")
		os.Setenv("ALLOWED_ORIGINS", "http://example.com,http://localhost:3000")
		os.Setenv("ENABLE_METRICS", "false")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("MAX_CONTENT_SIZE", "5242880") // 5MB
		os.Setenv("READ_TIMEOUT", "30")
		os.Setenv("WRITE_TIMEOUT", "30")
		os.Setenv("IDLE_TIMEOUT", "180")
		os.Setenv("RATE_LIMIT_WINDOW", "120")
		os.Setenv("RATE_LIMIT_MAX_REQUESTS", "100")
		os.Setenv("ROUTES_CONFIG_PATH", "custom/routes.json")
		os.Setenv("ENABLE_CACHING", "false")
		os.Setenv("BACKEND_URLS", "http://backend1:8001,http://backend2:8002")

		// 사용자 지정 값 테스트
		cfg, err := config.Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 사용자 지정 값 확인
		assert.Equal(t, 9090, cfg.Port)
		assert.Equal(t, "http://custom-backend:8000", cfg.DefaultBackend)
		assert.Equal(t, "custom-secret-key", cfg.JWTSecret)
		assert.Equal(t, "custom-issuer", cfg.JWTIssuer)
		assert.Equal(t, 7200*time.Second, cfg.JWTExpirationDelta)
		assert.ElementsMatch(t, []string{"http://example.com", "http://localhost:3000"}, cfg.AllowedOrigins)
		assert.Equal(t, false, cfg.EnableMetrics)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, int64(5242880), cfg.MaxContentSize) // 5MB
		assert.Equal(t, 30*time.Second, cfg.ReadTimeout)
		assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
		assert.Equal(t, 180*time.Second, cfg.IdleTimeout)
		assert.Equal(t, 120*time.Second, cfg.RateLimitWindow)
		assert.Equal(t, 100, cfg.RateLimitMaxReqs)
		assert.Equal(t, "custom/routes.json", cfg.RoutesConfigPath)
		assert.Equal(t, false, cfg.EnableCaching)
		assert.ElementsMatch(t, []string{"http://backend1:8001", "http://backend2:8002"}, cfg.Backends)
	})

	t.Run("Invalid Values", func(t *testing.T) {
		// 잘못된 환경 변수 설정
		os.Setenv("PORT", "invalid")
		os.Setenv("JWT_EXPIRATION", "invalid")
		os.Setenv("ENABLE_METRICS", "invalid")
		os.Setenv("MAX_CONTENT_SIZE", "invalid")
		os.Setenv("READ_TIMEOUT", "invalid")

		// 잘못된 값 테스트 (기본값으로 대체)
		cfg, err := config.Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 기본값으로 대체 확인
		assert.Equal(t, 8080, cfg.Port)
		assert.Equal(t, 3600*time.Second, cfg.JWTExpirationDelta)
		assert.Equal(t, true, cfg.EnableMetrics)
		assert.Equal(t, int64(10*1024*1024), cfg.MaxContentSize)
		assert.Equal(t, 20*time.Second, cfg.ReadTimeout)
	})

	t.Run("LoadRoutes", func(t *testing.T) {
		// 테스트 라우트 구성 파일 생성
		testRoutesJSON := `{
			"routes": [
				{
					"path": "/",
					"targetURL": "http://test:8081",
					"methods": ["GET"],
					"stripPrefix": "",
					"requireAuth": false,
					"cacheable": true,
					"timeout": 30
				},
				{
					"path": "/api/test",
					"targetURL": "http://test:8082/test",
					"methods": ["GET", "POST"],
					"stripPrefix": "/api",
					"requireAuth": true,
					"cacheable": false,
					"timeout": 20
				}
			]
		}`

		// 임시 라우트 파일 생성
		tempFile, err := os.CreateTemp("", "test-routes-*.json")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(testRoutesJSON)
		assert.NoError(t, err)
		tempFile.Close()

		// 환경 변수 설정
		os.Setenv("ROUTES_CONFIG_PATH", tempFile.Name())

		// 설정 로드
		cfg, err := config.Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 라우트 로드
		routes, err := cfg.LoadRoutes()
		assert.NoError(t, err)
		assert.NotNil(t, routes)
		assert.Equal(t, 2, len(routes))

		// 첫 번째 라우트 확인
		assert.Equal(t, "/", routes[0].Path)
		assert.Equal(t, "http://test:8081", routes[0].TargetURL)
		assert.ElementsMatch(t, []string{"GET"}, routes[0].Methods)
		assert.Equal(t, "", routes[0].StripPrefix)
		assert.Equal(t, false, routes[0].RequireAuth)
		assert.Equal(t, true, routes[0].Cacheable)
		assert.Equal(t, 30, routes[0].Timeout)

		// 두 번째 라우트 확인
		assert.Equal(t, "/api/test", routes[1].Path)
		assert.Equal(t, "http://test:8082/test", routes[1].TargetURL)
		assert.ElementsMatch(t, []string{"GET", "POST"}, routes[1].Methods)
		assert.Equal(t, "/api", routes[1].StripPrefix)
		assert.Equal(t, true, routes[1].RequireAuth)
		assert.Equal(t, false, routes[1].Cacheable)
		assert.Equal(t, 20, routes[1].Timeout)
	})
}
