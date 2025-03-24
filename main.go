package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/isinthesky/api-gateway/config"
	"github.com/isinthesky/api-gateway/internal/middleware"
	"github.com/isinthesky/api-gateway/proxy"
)

// RouteConfig는 라우팅 구성을 정의합니다.
type RouteConfig struct {
	Path        string   `json:"path"`
	TargetURL   string   `json:"targetURL"`
	Methods     []string `json:"methods"`
	StripPrefix string   `json:"stripPrefix"`
	RequireAuth bool     `json:"requireAuth"`
}

// RoutesJSON은 routes.json 파일의 구조입니다.
type RoutesJSON struct {
	Routes []RouteConfig `json:"routes"`
}

func main() {
	// 설정을 로드합니다.
	cfg := config.LoadConfig()

	// Gin 모드 설정
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 미들웨어 등록
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware())
	
	// CORS 미들웨어 설정
	if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
		router.Use(middleware.CORSMiddleware())
	} else {
		router.Use(middleware.CustomCORSMiddleware(cfg.AllowedOrigins))
	}
	
	// 레이트 리미터 설정
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitWindow, cfg.RateLimitMaxReqs)
	rateLimiter.StartCleaner(5 * time.Minute) // 5분마다 정리
	router.Use(middleware.RateLimitMiddleware(rateLimiter))
	
	// 요청 크기 제한 설정
	router.Use(middleware.SizeLimitMiddleware(cfg))
	
	// 쿠키→헤더 변환 미들웨어
	router.Use(middleware.CookieToHeaderMiddleware)

	// 메트릭 엔드포인트 설정
	if cfg.EnableMetrics {
		router.Use(middleware.MetricsMiddleware())
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// 상태 확인 엔드포인트
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 라우트 구성 파일 로드
	routesBytes, err := ioutil.ReadFile(cfg.RoutesConfigPath)
	if err != nil {
		log.Fatalf("라우트 설정 파일을 열 수 없습니다: %v", err)
	}

	var routesJSON RoutesJSON
	if err := json.Unmarshal(routesBytes, &routesJSON); err != nil {
		log.Fatalf("라우트 설정을 파싱할 수 없습니다: %v", err)
	}

	// HTTP 프록시 설정
	httpProxy := proxy.NewHTTPProxy(cfg.BackendBaseURL)

	// JWT 인증 미들웨어 설정
	jwtConfig := middleware.JWTConfig{
		SecretKey:       cfg.JWTSecret,
		Issuer:          cfg.JWTIssuer,
		ExpirationDelta: cfg.JWTExpirationDelta,
	}
	jwtAuthMiddleware := middleware.JWTAuthMiddleware(jwtConfig)

	// 각 라우트 설정
	for _, route := range routesJSON.Routes {
		handlers := []gin.HandlerFunc{}

		// JWT 인증이 필요한 경우
		if route.RequireAuth {
			handlers = append(handlers, jwtAuthMiddleware)
		}

		// 프록시 핸들러 추가
		stripPath := route.StripPrefix != ""
		handlers = append(handlers, proxy.HTTPProxyHandler(httpProxy, route.TargetURL, stripPath))

		// 지원하는 HTTP 메서드에 따라 라우트 등록
		for _, method := range route.Methods {
			switch method {
			case "GET":
				router.GET(route.Path, handlers...)
			case "POST":
				router.POST(route.Path, handlers...)
			case "PUT":
				router.PUT(route.Path, handlers...)
			case "DELETE":
				router.DELETE(route.Path, handlers...)
			case "PATCH":
				router.PATCH(route.Path, handlers...)
			case "HEAD":
				router.HEAD(route.Path, handlers...)
			case "OPTIONS":
				router.OPTIONS(route.Path, handlers...)
			}
		}
	}

	// WebSocket 프록시 설정 (routes.json과 충돌하지 않는 경로 사용)
	wsUpgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	wsProxy := proxy.NewWebSocketProxy(cfg.BackendBaseURL, wsUpgrader)
	router.GET("/websocket/*proxyPath", proxy.WebSocketProxyHandler(wsProxy))

	// 서버 시작
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	log.Printf("API Gateway 서버 시작 - 포트: %d\n", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("서버 시작 오류: %v", err)
	}
}
