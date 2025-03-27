// +build unit

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/internal/middleware"
)

func TestCORS(t *testing.T) {
	// Gin 테스트 모드 설정
	gin.SetMode(gin.TestMode)

	t.Run("AllowAll", func(t *testing.T) {
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.CORS([]string{"*"}))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// 테스트 요청 생성
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		// 요청 실행
		router.ServeHTTP(w, req)

		// 응답 확인
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("AllowSpecificOrigins", func(t *testing.T) {
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.CORS([]string{"http://example.com", "http://localhost:3000"}))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// 허용된 오리진 테스트
		t.Run("AllowedOrigin", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://example.com")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		})

		// 허용되지 않은 오리진 테스트
		t.Run("DisallowedOrigin", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://evil.com")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		})
	})

	t.Run("PreflightRequest", func(t *testing.T) {
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.CORS([]string{"*"}))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// 프리플라이트 요청 생성
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
		w := httptest.NewRecorder()

		// 요청 실행
		router.ServeHTTP(w, req)

		// 프리플라이트 응답 확인
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("CORS2Middleware", func(t *testing.T) {
		// 라우터 설정
		router := gin.New()
		router.Use(middleware.CORS2([]string{"http://example.com", "*.example.org"}))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "test")
		})

		// 허용된 도메인 테스트
		t.Run("AllowedDomain", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://example.com")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		})

		// 허용된 와일드카드 도메인 테스트
		t.Run("AllowedWildcardDomain", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://sub.example.org")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "http://sub.example.org", w.Header().Get("Access-Control-Allow-Origin"))
		})

		// 허용되지 않은 도메인 테스트
		t.Run("DisallowedDomain", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "http://evil.com")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
		})
	})
}
