// +build unit

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/internal/middleware"
	"github.com/isinthesky/api-gateway/pkg/ratelimiter"
)

func TestRateLimit(t *testing.T) {
	// Gin 테스트 모드 설정
	gin.SetMode(gin.TestMode)

	t.Run("BasicRateLimit", func(t *testing.T) {
		// 레이트 리미터 설정 (2초 동안 3개 요청 허용)
		limiter := ratelimiter.New(2*time.Second, 3)
		
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.RateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// 여러 번 요청 실행
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234" // 같은 IP 주소 사용
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 처음 3번은 성공
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 제한 초과 요청
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234" // 같은 IP 주소 사용
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 제한을 초과하면 429 Too Many Requests
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "요청 속도 제한 초과")
		assert.NotEmpty(t, w.Header().Get("Retry-After"))
	})

	t.Run("DifferentIPAddresses", func(t *testing.T) {
		// 레이트 리미터 설정 (5초 동안 2개 요청 허용)
		limiter := ratelimiter.New(5*time.Second, 2)
		
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.RateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// IP 주소 1
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 각 IP 주소별로 2번까지 성공
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// IP 주소 1 - 제한 초과
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// 제한 초과
		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		// IP 주소 2 - 다른 IP 주소는 독립적으로 제한
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.2:1234" // 다른 IP 주소
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// 다른 IP 주소는 독립적으로 제한되므로 성공
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("DynamicRateLimit", func(t *testing.T) {
		// 경로별 레이트 리미터 설정
		configs := make(map[string]ratelimiter.RateLimiter)
		configs["/high-limit"] = ratelimiter.New(5*time.Second, 5)       // 5초 동안 5개 요청
		configs["/low-limit"] = ratelimiter.New(5*time.Second, 2)        // 5초 동안 2개 요청
		configs["default"] = ratelimiter.New(5*time.Second, 3)           // 기본 - 5초 동안 3개 요청
		
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.DynamicRateLimit(configs))
		
		router.GET("/high-limit", func(c *gin.Context) {
			c.String(http.StatusOK, "high limit")
		})
		
		router.GET("/low-limit", func(c *gin.Context) {
			c.String(http.StatusOK, "low limit")
		})
		
		router.GET("/default-limit", func(c *gin.Context) {
			c.String(http.StatusOK, "default limit")
		})

		// 높은 제한 경로 테스트
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/high-limit", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 낮은 제한 경로 테스트
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/low-limit", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 낮은 제한 초과
		req := httptest.NewRequest(http.MethodGet, "/low-limit", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("IPBasedRateLimit", func(t *testing.T) {
		// IP 기반 레이트 리미터 설정
		limiter := ratelimiter.New(5*time.Second, 2)
		
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.IPBasedRateLimit(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// IP 주소 기반 제한 테스트
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// 제한 초과
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "IP 기반 요청 속도 제한 초과")
	})
}
