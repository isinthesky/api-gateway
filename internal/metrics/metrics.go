package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector는 API Gateway 메트릭 수집기입니다.
type Collector struct {
	requestTotal      *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	requestSize       *prometheus.SummaryVec
	responseSize      *prometheus.SummaryVec
	circuitBreakerTotal *prometheus.CounterVec
	cacheHitTotal     *prometheus.CounterVec
	ratelimitTotal    *prometheus.CounterVec
	errorTotal        *prometheus.CounterVec
	inFlightRequests  *prometheus.GaugeVec
}

// NewCollector는 새로운 메트릭 수집기를 생성합니다.
func NewCollector() *Collector {
	return &Collector{
		requestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_total",
				Help: "API Gateway 총 요청 수",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_request_duration_seconds",
				Help:    "API Gateway 요청 처리 시간 (초)",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		requestSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "api_gateway_request_size_bytes",
				Help:       "API Gateway 요청 크기 (바이트)",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"method", "path"},
		),
		responseSize: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "api_gateway_response_size_bytes",
				Help:       "API Gateway 응답 크기 (바이트)",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"method", "path", "status"},
		),
		circuitBreakerTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_circuit_breaker_triggers_total",
				Help: "API Gateway 서킷 브레이커 트리거 횟수",
			},
			[]string{"path", "status"},
		),
		cacheHitTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_cache_hits_total",
				Help: "API Gateway 캐시 히트 횟수",
			},
			[]string{"path"},
		),
		ratelimitTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_total",
				Help: "API Gateway 속도 제한 적용 횟수",
			},
			[]string{"path", "client_ip"},
		),
		errorTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_errors_total",
				Help: "API Gateway 오류 발생 횟수",
			},
			[]string{"method", "path", "error_type"},
		),
		inFlightRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_in_flight_requests",
				Help: "API Gateway 현재 처리 중인 요청 수",
			},
			[]string{"method", "path"},
		),
	}
}

// ObserveRequest는 요청 메트릭을 기록합니다.
func (c *Collector) ObserveRequest(r *http.Request, size int64) {
	if r == nil {
		return
	}

	method := r.Method
	path := r.URL.Path

	c.requestSize.WithLabelValues(method, path).Observe(float64(size))
}

// ObserveResponse는 응답 메트릭을 기록합니다.
func (c *Collector) ObserveResponse(r *http.Request, status int, size int, duration time.Duration) {
	if r == nil {
		return
	}

	method := r.Method
	path := r.URL.Path
	statusStr := strconv.Itoa(status)

	c.requestTotal.WithLabelValues(method, path, statusStr).Inc()
	c.requestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
	c.responseSize.WithLabelValues(method, path, statusStr).Observe(float64(size))
}

// ObserveCacheHit는 캐시 히트를 기록합니다.
func (c *Collector) ObserveCacheHit(path string) {
	c.cacheHitTotal.WithLabelValues(path).Inc()
}

// ObserveCircuitBreaker는 서킷 브레이커 트리거를 기록합니다.
func (c *Collector) ObserveCircuitBreaker(path string, status string) {
	c.circuitBreakerTotal.WithLabelValues(path, status).Inc()
}

// ObserveRateLimit는 속도 제한 적용을 기록합니다.
func (c *Collector) ObserveRateLimit(path string, clientIP string) {
	c.ratelimitTotal.WithLabelValues(path, clientIP).Inc()
}

// ObserveError는 오류 발생을 기록합니다.
func (c *Collector) ObserveError(r *http.Request, errorType string) {
	if r == nil {
		return
	}

	method := r.Method
	path := r.URL.Path

	c.errorTotal.WithLabelValues(method, path, errorType).Inc()
}

// IncInFlightRequests는 현재 처리 중인 요청 수를 증가시킵니다.
func (c *Collector) IncInFlightRequests(r *http.Request) {
	if r == nil {
		return
	}

	c.inFlightRequests.WithLabelValues(r.Method, r.URL.Path).Inc()
}

// DecInFlightRequests는 현재 처리 중인 요청 수를 감소시킵니다.
func (c *Collector) DecInFlightRequests(r *http.Request) {
	if r == nil {
		return
	}

	c.inFlightRequests.WithLabelValues(r.Method, r.URL.Path).Dec()
}
