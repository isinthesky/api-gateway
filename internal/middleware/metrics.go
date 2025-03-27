package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isinthesky/api-gateway/internal/metrics"
)

// metricsResponseWriter는 응답 크기를 추적하는 ResponseWriter 래퍼입니다.
type metricsResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// newMetricsResponseWriter는 새로운 metricsResponseWriter를 생성합니다.
func newMetricsResponseWriter(writer gin.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: writer,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
	}
}

// Write는 응답 본문을 캡처합니다.
func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader는 응답 상태 코드를 캡처합니다.
func (w *metricsResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Status는 응답 상태 코드를 반환합니다.
func (w *metricsResponseWriter) Status() int {
	return w.status
}

// Size는 응답 본문 크기를 반환합니다.
func (w *metricsResponseWriter) Size() int {
	return w.body.Len()
}

// Metrics는 메트릭을 수집하는 미들웨어입니다.
func Metrics(collector *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 시작 시간
		start := time.Now()

		// 요청 본문 크기 캡처 (필요한 경우)
		var requestSize int64
		if c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		// 요청 메트릭 기록
		collector.ObserveRequest(c.Request, requestSize)

		// 처리 중인 요청 수 증가
		collector.IncInFlightRequests(c.Request)
		defer collector.DecInFlightRequests(c.Request)

		// 응답 본문 캡처를 위한 래퍼 설정
		resWriter := newMetricsResponseWriter(c.Writer)
		c.Writer = resWriter

		// 다음 핸들러 실행
		c.Next()

		// 요청 처리 시간
		duration := time.Since(start)

		// 응답 메트릭 기록
		collector.ObserveResponse(c.Request, resWriter.Status(), resWriter.Size(), duration)

		// 오류 발생 시 기록
		if len(c.Errors) > 0 {
			collector.ObserveError(c.Request, "server_error")
		}

		// 상태 코드가 오류인 경우
		if resWriter.Status() >= 400 {
			var errorType string
			switch {
			case resWriter.Status() >= 500:
				errorType = "server_error"
			case resWriter.Status() >= 400:
				errorType = "client_error"
			}
			collector.ObserveError(c.Request, errorType)
		}
	}
}

// CircuitBreakerMetrics는 서킷 브레이커 이벤트를 기록하는 미들웨어입니다.
func CircuitBreakerMetrics(collector *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 서킷 브레이커 상태는 핸들러에서 오류 응답에 포함되어 있음
		c.Next()

		// 상태 코드가 서비스 사용 불가인 경우 서킷 브레이커로 간주
		if c.Writer.Status() == http.StatusServiceUnavailable {
			// 오류 메시지에서 서킷 브레이커 관련 정보 추출
			var status string
			if c.Errors.Last() != nil {
				// Meta가 map[string]interface{} 타입인지 확인
				if meta, ok := c.Errors.Last().Meta.(map[string]interface{}); ok {
					if s, ok := meta["circuit_breaker_status"].(string); ok {
						status = s
					} else {
						status = "open" // 기본값
					}
				} else {
					status = "open" // 기본값
				}
			} else {
				status = "open" // 기본값
			}

			collector.ObserveCircuitBreaker(c.Request.URL.Path, status)
		}
	}
}

// CacheMetrics는 캐시 히트를 기록하는 미들웨어입니다.
func CacheMetrics(collector *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// X-Cache 헤더가 HIT인 경우 캐시 히트로 간주
		if c.Writer.Header().Get("X-Cache") == "HIT" {
			collector.ObserveCacheHit(c.Request.URL.Path)
		}
	}
}

// RateLimitMetrics는 속도 제한 적용을 기록하는 미들웨어입니다.
func RateLimitMetrics(collector *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 상태 코드가 너무 많은 요청인 경우 속도 제한으로 간주
		if c.Writer.Status() == http.StatusTooManyRequests {
			collector.ObserveRateLimit(c.Request.URL.Path, c.ClientIP())
		}
	}
}

// DetailedMetricsMiddleware는 기본적인 Prometheus 메트릭을 수집하는 미들웨어입니다. (구버전 호환용)
func DetailedMetricsMiddleware() gin.HandlerFunc {
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

		// 여기서 필요한 메트릭 수집 로직 구현
		// (Prometheus 메트릭 기록 등)
		_ = status
		_ = duration
		_ = method
		_ = path
	}
}
