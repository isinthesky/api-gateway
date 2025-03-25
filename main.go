package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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

	// 정렬된 라우트 로깅
	log.Println("=== 라우트 순서 확인 ===")
	for i, route := range routesJSON.Routes {
		log.Printf("%d: %s -> %s", i+1, route.Path, route.TargetURL)
	}
	log.Println("========================")

	// 라우트를 그룹화
	var rootRoutes []RouteConfig        // 루트 경로 라우트 ("/")
	var apiRoutes []RouteConfig         // API 관련 라우트 ("/api/*")
	var specificRoutes []RouteConfig    // 특정 경로 라우트 (예: "/login")
	var rootCatchAllRoute *RouteConfig  // 루트 캐치올 라우트 ("/*proxyPath")

	for _, route := range routesJSON.Routes {
		if route.Path == "/" {
			rootRoutes = append(rootRoutes, route)
		} else if strings.HasPrefix(route.Path, "/api") {
			apiRoutes = append(apiRoutes, route)
		} else if route.Path == "/*proxyPath" {
			routeCopy := route
			rootCatchAllRoute = &routeCopy
		} else {
			specificRoutes = append(specificRoutes, route)
		}
	}

	// JWT 인증 미들웨어 설정 (인증 스킵을 위해 주석 처리)
	// jwtConfig := middleware.JWTConfig{
	// 	SecretKey:       cfg.JWTSecret,
	// 	Issuer:          cfg.JWTIssuer,
	// 	ExpirationDelta: cfg.JWTExpirationDelta,
	// }
	// jwtAuthMiddleware := middleware.JWTAuthMiddleware(jwtConfig)

	// 1. 루트 라우트 등록
	for _, route := range rootRoutes {
		registerRoute(router, httpProxy, route)
	}

	// 2. 특정 경로 라우트 등록
	for _, route := range specificRoutes {
		registerRoute(router, httpProxy, route)
	}

	// 3. API 라우트 등록 (그룹 사용)
	apiGroup := router.Group("/api")
	for _, route := range apiRoutes {
		// "/api" 접두사 제거
		subPath := strings.TrimPrefix(route.Path, "/api")
		log.Printf("API 그룹 라우트 등록: %s -> %s", subPath, route.TargetURL)
		
		handlers := []gin.HandlerFunc{}
		
		// JWT 인증이 필요한 경우
		// if route.RequireAuth {
		// 	handlers = append(handlers, jwtAuthMiddleware)
		// }
		
		// 프록시 핸들러 추가
		stripPath := route.StripPrefix != ""
		handlers = append(handlers, proxy.HTTPProxyHandler(httpProxy, route.TargetURL, stripPath))
		
		// HTTP 메서드에 따라 라우트 등록
		for _, method := range route.Methods {
			switch method {
			case "GET":
				apiGroup.GET(subPath, handlers...)
			case "POST":
				apiGroup.POST(subPath, handlers...)
			case "PUT":
				apiGroup.PUT(subPath, handlers...)
			case "DELETE":
				apiGroup.DELETE(subPath, handlers...)
			case "PATCH":
				apiGroup.PATCH(subPath, handlers...)
			case "HEAD":
				apiGroup.HEAD(subPath, handlers...)
			case "OPTIONS":
				apiGroup.OPTIONS(subPath, handlers...)
			}
		}
	}

	// 4. 루트 캐치올 라우트 등록 (있는 경우)
	if rootCatchAllRoute != nil {
		log.Println("루트 캐치올 라우트 등록:", rootCatchAllRoute.Path, "->", rootCatchAllRoute.TargetURL)
		
		// 캐치올 라우트를 API 경로와 충돌하지 않도록 명시적으로 등록
		wildcardHandlers := []gin.HandlerFunc{
			proxy.HTTPProxyHandler(httpProxy, rootCatchAllRoute.TargetURL, false),
		}
		
		// 특정 경로를 제외한 나머지 경로에 대해 캐치올 처리
		for _, method := range rootCatchAllRoute.Methods {
			log.Printf("캐치올 핸들러 등록: %s /web/* -> %s", method, rootCatchAllRoute.TargetURL)
			router.Handle(method, "/web/*path", wildcardHandlers...)
			
			log.Printf("캐치올 핸들러 등록: %s /assets/* -> %s", method, rootCatchAllRoute.TargetURL)
			router.Handle(method, "/assets/*path", wildcardHandlers...)
			
			log.Printf("캐치올 핸들러 등록: %s /static/* -> %s", method, rootCatchAllRoute.TargetURL)
			router.Handle(method, "/static/*path", wildcardHandlers...)
			
			// 필요한 경우 다른 경로도 추가
		}
		
		// NoRoute 핸들러 등록 (매칭되지 않는 모든 경로)
		router.NoRoute(func(c *gin.Context) {
			log.Printf("NoRoute 핸들러: %s %s -> %s", c.Request.Method, c.Request.URL.Path, rootCatchAllRoute.TargetURL)
			
			// 프록시 핸들러 실행
			proxy.HTTPProxyHandler(httpProxy, rootCatchAllRoute.TargetURL, false)(c)
		})
	}

	// WebSocket 프록시 설정
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

// registerRoute는 기본 라우터에 라우트를 등록하는 헬퍼 함수입니다.
func registerRoute(router *gin.Engine, httpProxy *proxy.HTTPProxy, route RouteConfig) {
	handlers := []gin.HandlerFunc{}

	// JWT 인증이 필요한 경우
	// if route.RequireAuth {
	// 	handlers = append(handlers, jwtAuthMiddleware)
	// }

	// 프록시 핸들러 추가
	stripPath := route.StripPrefix != ""
	handlers = append(handlers, proxy.HTTPProxyHandler(httpProxy, route.TargetURL, stripPath))

	// 지원하는 HTTP 메서드에 따라 라우트 등록
	for _, method := range route.Methods {
		log.Printf("라우트 등록: %s %s -> %s", method, route.Path, route.TargetURL)
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

