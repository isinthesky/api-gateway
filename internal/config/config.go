package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config는 애플리케이션 설정을 저장하는 구조체입니다.
type Config struct {
	Port                        int           // API Gateway 포트
	DefaultBackend              string        // 기본 백엔드 서비스 URL
	Backends                    []string      // 백엔드 서비스 URL 목록
	JWTSecret                   string        // JWT 토큰 검증용 비밀 키
	JWTIssuer                   string        // JWT 토큰 발행자
	JWTExpirationDelta          time.Duration // JWT 토큰 만료 시간 (초)
	AllowedOrigins              []string      // CORS 허용 오리진 목록
	EnableMetrics               bool          // Prometheus 메트릭 수집 활성화 여부
	LogLevel                    string        // 로그 레벨 (debug, info, warn, error)
	MaxContentSize              int64         // 최대 요청 본문 크기 (바이트)
	ReadTimeout                 time.Duration // 읽기 타임아웃 (초)
	WriteTimeout                time.Duration // 쓰기 타임아웃 (초)
	IdleTimeout                 time.Duration // 유휴 타임아웃 (초)
	RateLimitWindow             time.Duration // 레이트 리밋 윈도우 크기 (초)
	RateLimitMaxReqs            int           // 윈도우 당 최대 요청 수
	RoutesConfigPath            string        // 라우트 설정 파일 경로
	EnableCaching               bool          // 캐싱 활성화 여부
	CacheTTL                    time.Duration // 캐시 항목 기본 수명
	CircuitBreakerErrorThreshold float64       // 서킷 브레이커 오류 임계값
	CircuitBreakerMinRequests    int           // 서킷 브레이커 최소 요청 수
	CircuitBreakerTimeout        time.Duration // 서킷 브레이커 타임아웃
	CircuitBreakerHalfOpenReqs   int           // 서킷 브레이커 반열림 상태 최대 요청 수
	CircuitBreakerSuccessThreshold int          // 서킷 브레이커 성공 임계값
}

// Load는 환경 변수와 구성 파일에서 설정을 로드합니다.
func Load() (*Config, error) {
	// .env 파일 로드 (존재하지 않아도 오류 없음)
	if err := godotenv.Load(); err != nil {
		fmt.Printf("경고: .env 파일을 찾을 수 없습니다. 환경 변수를 직접 사용합니다.\n")
	}

	cfg := &Config{
		Port:                      getEnvInt("PORT", 8080),
		DefaultBackend:            getEnv("BACKEND_URL", "http://localhost:8081"),
		JWTSecret:                 getEnv("JWT_SECRET", "your-secret-key"),
		JWTIssuer:                 getEnv("JWT_ISSUER", "api-gateway"),
		JWTExpirationDelta:        time.Duration(getEnvInt("JWT_EXPIRATION", 3600)) * time.Second,
		AllowedOrigins:            getEnvArray("ALLOWED_ORIGINS", []string{"*"}),
		EnableMetrics:             getEnvBool("ENABLE_METRICS", true),
		LogLevel:                  getEnv("LOG_LEVEL", "info"),
		MaxContentSize:            int64(getEnvInt("MAX_CONTENT_SIZE", 10*1024*1024)), // 10MB
		ReadTimeout:               time.Duration(getEnvInt("READ_TIMEOUT", 20)) * time.Second,
		WriteTimeout:              time.Duration(getEnvInt("WRITE_TIMEOUT", 20)) * time.Second,
		IdleTimeout:               time.Duration(getEnvInt("IDLE_TIMEOUT", 120)) * time.Second,
		RateLimitWindow:           time.Duration(getEnvInt("RATE_LIMIT_WINDOW", 60)) * time.Second,
		RateLimitMaxReqs:          getEnvInt("RATE_LIMIT_MAX_REQUESTS", 200),
		RoutesConfigPath:          getEnv("ROUTES_CONFIG_PATH", "configs/routes.json"),
		EnableCaching:             getEnvBool("ENABLE_CACHING", true),
		CacheTTL:                  time.Duration(getEnvInt("CACHE_TTL", 300)) * time.Second, // 기본 5분
		CircuitBreakerErrorThreshold: getEnvFloat("CIRCUIT_BREAKER_ERROR_THRESHOLD", 0.5),
		CircuitBreakerMinRequests:   getEnvInt("CIRCUIT_BREAKER_MIN_REQUESTS", 10),
		CircuitBreakerTimeout:       time.Duration(getEnvInt("CIRCUIT_BREAKER_TIMEOUT", 60)) * time.Second,
		CircuitBreakerHalfOpenReqs:  getEnvInt("CIRCUIT_BREAKER_HALF_OPEN_REQS", 5),
		CircuitBreakerSuccessThreshold: getEnvInt("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
	}

	// 백엔드 URL 목록
	backendsStr := getEnv("BACKEND_URLS", "")
	if backendsStr != "" {
		cfg.Backends = strings.Split(backendsStr, ",")
		for i, url := range cfg.Backends {
			cfg.Backends[i] = strings.TrimSpace(url)
		}
	} else {
		// 기본 백엔드만 사용
		cfg.Backends = []string{cfg.DefaultBackend}
	}

	// 라우트 구성 파일 존재 여부 확인
	if _, err := os.Stat(cfg.RoutesConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("라우트 구성 파일이 존재하지 않습니다: %s", cfg.RoutesConfigPath)
	}

	return cfg, nil
}

// LoadRoutes는 라우트 구성 파일을 로드합니다.
func (c *Config) LoadRoutes() ([]Route, error) {
	data, err := os.ReadFile(c.RoutesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("라우트 구성 파일 읽기 실패: %v", err)
	}

	var routesConfig RoutesConfig
	if err := json.Unmarshal(data, &routesConfig); err != nil {
		return nil, fmt.Errorf("라우트 구성 파싱 실패: %v", err)
	}

	return routesConfig.Routes, nil
}

// RoutesConfig는 routes.json 파일의 구조입니다.
type RoutesConfig struct {
	Routes []Route `json:"routes"`
}

// Route는 단일 라우트 구성입니다.
type Route struct {
	Path        string   `json:"path"`
	TargetURL   string   `json:"targetURL"`
	Methods     []string `json:"methods"`
	StripPrefix string   `json:"stripPrefix"`
	RequireAuth bool     `json:"requireAuth"`
	Cacheable   bool     `json:"cacheable"`
	Timeout     int      `json:"timeout"` // 초 단위
}

// 환경 변수 유틸리티 함수
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvArray(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.Split(value, ",")
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
