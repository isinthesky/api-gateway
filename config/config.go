package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config는 애플리케이션 설정을 저장하는 구조체입니다.
type Config struct {
	Port               int           // API Gateway 포트
	BackendBaseURL     string        // 기본 백엔드 서비스 URL
	JWTSecret          string        // JWT 토큰 검증용 비밀 키
	JWTIssuer          string        // JWT 토큰 발행자
	JWTExpirationDelta time.Duration // JWT 토큰 만료 시간 (초)
	AllowedOrigins     []string      // CORS 허용 오리진 목록
	EnableMetrics      bool          // Prometheus 메트릭 수집 활성화 여부
	LogLevel           string        // 로그 레벨 (debug, info, warn, error)
	MaxContentSize     int64         // 최대 요청 본문 크기 (바이트)
	ReadTimeout        time.Duration // 읽기 타임아웃 (초)
	WriteTimeout       time.Duration // 쓰기 타임아웃 (초)
	IdleTimeout        time.Duration // 유휴 타임아웃 (초)
	RateLimitWindow    time.Duration // 레이트 리밋 윈도우 크기 (초)
	RateLimitMaxReqs   int           // 윈도우 당 최대 요청 수
	RoutesConfigPath   string        // 라우트 설정 파일 경로
}

// LoadConfig는 환경 변수나 설정 파일에서 설정을 로드합니다.
func LoadConfig() *Config {
	// .env 파일 로드 (존재하지 않아도 오류 없음)
	if err := godotenv.Load(); err != nil {
		log.Println("경고: .env 파일을 찾을 수 없습니다. 환경 변수를 직접 사용합니다.")
	}

	// 포트
	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		port = 8080
	}

	// JWT 만료 시간
	jwtExpiration, err := strconv.Atoi(getEnv("JWT_EXPIRATION", "3600"))
	if err != nil {
		jwtExpiration = 3600
	}

	// 메트릭 활성화 여부
	enableMetrics, err := strconv.ParseBool(getEnv("ENABLE_METRICS", "true"))
	if err != nil {
		enableMetrics = true
	}

	// 최대 컨텐츠 크기
	maxContentSize, err := strconv.ParseInt(getEnv("MAX_CONTENT_SIZE", "10485760"), 10, 64)
	if err != nil {
		maxContentSize = 10 * 1024 * 1024 // 10MB
	}

	// 타임아웃 설정
	readTimeout, err := strconv.Atoi(getEnv("READ_TIMEOUT", "30"))
	if err != nil {
		readTimeout = 30
	}

	writeTimeout, err := strconv.Atoi(getEnv("WRITE_TIMEOUT", "30"))
	if err != nil {
		writeTimeout = 30
	}

	idleTimeout, err := strconv.Atoi(getEnv("IDLE_TIMEOUT", "120"))
	if err != nil {
		idleTimeout = 120
	}

	// 레이트 리밋 설정
	rateLimitWindow, err := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW", "60"))
	if err != nil {
		rateLimitWindow = 60
	}

	rateLimitMaxReqs, err := strconv.Atoi(getEnv("RATE_LIMIT_MAX_REQUESTS", "100"))
	if err != nil {
		rateLimitMaxReqs = 100
	}

	// CORS 허용 오리진 목록 (쉼표로 구분된 문자열)
	originsStr := getEnv("ALLOWED_ORIGINS", "*")
	var origins []string
	if originsStr == "*" {
		origins = []string{"*"}
	} else {
		origins = strings.Split(originsStr, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
	}

	return &Config{
		Port:               port,
		BackendBaseURL:     getEnv("BACKEND_URL", "http://localhost:8081"),
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
		JWTIssuer:          getEnv("JWT_ISSUER", "api-gateway"),
		JWTExpirationDelta: time.Duration(jwtExpiration) * time.Second,
		AllowedOrigins:     origins,
		EnableMetrics:      enableMetrics,
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		MaxContentSize:     maxContentSize,
		ReadTimeout:        time.Duration(readTimeout) * time.Second,
		WriteTimeout:       time.Duration(writeTimeout) * time.Second,
		IdleTimeout:        time.Duration(idleTimeout) * time.Second,
		RateLimitWindow:    time.Duration(rateLimitWindow) * time.Second,
		RateLimitMaxReqs:   rateLimitMaxReqs,
		RoutesConfigPath:   getEnv("ROUTES_CONFIG_PATH", "config/routes.json"),
	}
}

// getEnv는 환경 변수를 가져오고, 없으면 기본값을 반환합니다.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
