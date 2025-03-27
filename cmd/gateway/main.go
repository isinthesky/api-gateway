package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/isinthesky/api-gateway/internal/config"
	"github.com/isinthesky/api-gateway/internal/handler"
	"github.com/isinthesky/api-gateway/internal/metrics"
	"github.com/isinthesky/api-gateway/internal/middleware"
	"github.com/isinthesky/api-gateway/pkg/cache"
	"github.com/isinthesky/api-gateway/pkg/circuitbreaker"
	"github.com/isinthesky/api-gateway/pkg/loadbalancer"
	"github.com/isinthesky/api-gateway/pkg/ratelimiter"
)

func main() {
	// 로깅 설정
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("API Gateway 시작 중...")

	// 설정 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("설정 로드 실패: %v", err)
	}

	// Gin 모드 설정
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 라우터 생성
	router := gin.New()

	// 기본 미들웨어 등록
	router.Use(gin.Recovery())
	router.Use(middleware.StructuredLogger())

	// CORS 미들웨어 설정
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	// 레이트 리미터 설정
	rateLimiter := ratelimiter.New(cfg.RateLimitWindow, cfg.RateLimitMaxReqs)
	router.Use(middleware.RateLimit(rateLimiter))
	
	// 요청 크기 제한 설정
	router.Use(middleware.SizeLimitMiddleware(cfg))
	
	// 메트릭 설정
	if cfg.EnableMetrics {
		metricsCollector := metrics.NewCollector()
		router.Use(middleware.Metrics(metricsCollector))
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// 캐시 초기화
	cacheProvider := cache.New(cfg.CacheTTL)

	// 부하 분산기 초기화
	var lb loadbalancer.LoadBalancer
	if len(cfg.Backends) > 1 {
		lb = loadbalancer.NewRoundRobin(cfg.Backends)
	} else {
		lb = loadbalancer.NewSingle(cfg.DefaultBackend)
	}

	// 서킷 브레이커 초기화
	cb := circuitbreaker.New(circuitbreaker.Config{
		ErrorThreshold:   cfg.CircuitBreakerErrorThreshold,
		MinRequests:      cfg.CircuitBreakerMinRequests,
		TimeoutDuration:  cfg.CircuitBreakerTimeout,
		HalfOpenMaxReqs:  cfg.CircuitBreakerHalfOpenReqs,
		SuccessThreshold: cfg.CircuitBreakerSuccessThreshold,
	})

	// 핸들러 초기화
	routeHandler := handler.NewRouteHandler(lb, cb, cacheProvider, cfg)
	
	// 라우트 설정
	if err := routeHandler.RegisterRoutes(router); err != nil {
		log.Fatalf("라우트 등록 실패: %v", err)
	}

	// 서버 초기화
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// 서버를 고루틴에서 실행
	go func() {
		log.Printf("API Gateway 서버 실행 중 - 포트: %d\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("서버 실행 오류: %v", err)
		}
	}()

	// 종료 신호 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("서버 종료 중...")

	// 서버 종료 컨텍스트 (10초 타임아웃)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 정상 종료 시도
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("서버 강제 종료: %v", err)
	}

	// 리소스 정리
	rateLimiter.Stop()
	cacheProvider.Close()

	log.Println("서버가 정상적으로 종료되었습니다")
}
