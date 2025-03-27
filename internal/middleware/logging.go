package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// 요청 처리 시간을 측정하는 히스토그램
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "API Gateway를 통한 HTTP 요청 처리 시간 (초)",
			Buckets: prometheus.DefBuckets, // 기본 버킷 사용 (0.005, 0.01, 0.025, ... 등 초)
		},
		[]string{"path", "method", "status"}, // 라벨: 경로, HTTP 메서드, 상태 코드별 측정
	)

	// 요청 횟수를 카운트하는 카운터
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "API Gateway를 통해 처리된 총 요청 수",
		},
		[]string{"path", "method", "status"}, // 라벨: 경로, HTTP 메서드, 상태 코드별 측정
	)
)

func init() {
	// Prometheus 기본 레지스트리에 메트릭 등록
	prometheus.MustRegister(reqDuration)
	prometheus.MustRegister(reqCounter)
}

// LoggingMiddleware는 요청 메서드, 경로, 상태 코드, 소요 시간을 로깅하는 미들웨어입니다.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 다음 핸들러 실행
		c.Next()

		// 요청 처리 후 로깅
		status := c.Writer.Status()
		latency := time.Since(start)
		log.Printf("%s %s -> %d (%v)", method, path, status, latency)
	}
}

// BasicMetricsMiddleware는 Prometheus 메트릭을 수집하는 미들웨어입니다.
func BasicMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		// 다음 핸들러 실행
		c.Next()

		// 요청 처리 완료 후 메트릭 수집
		status := c.Writer.Status()
		duration := time.Since(start).Seconds()

		statusLabel := string(rune(status))

		// 히스토그램에 관측값 기록
		reqDuration.WithLabelValues(path, method, statusLabel).Observe(duration)
		// 카운터 증가
		reqCounter.WithLabelValues(path, method, statusLabel).Inc()
	}
}
