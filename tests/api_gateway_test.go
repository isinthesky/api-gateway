package tests

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isinthesky/api-gateway/internal/middleware"
	"github.com/stretchr/testify/assert"
)

// 테스트용 간단한 JWT 인증 미들웨어
func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 테스트용으로 항상 통과
		c.Set("userId", "test-user-id")
		c.Set("roles", []string{"user"})
		c.Next()
	}
}

// TestCookieToHeaderMiddleware는 쿠키를 헤더로 변환하는 미들웨어를 테스트합니다.
func TestCookieToHeaderMiddleware(t *testing.T) {
	// Gin 모드 설정
	gin.SetMode(gin.TestMode)

	// 라우터 설정
	router := gin.New()
	router.Use(middleware.CookieToHeaderMiddleware)

	// 테스트 핸들러
	router.GET("/test", func(c *gin.Context) {
		// 헤더 값 확인
		auth := c.Request.Header.Get("Authorization")
		c.String(http.StatusOK, auth)
	})

	// 테스트 서버 생성
	server := httptest.NewServer(router)
	defer server.Close()

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 테스트 케이스 1: 쿠키 없는 경우
	t.Run("No Cookie", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/test", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "", string(body), "쿠키가 없을 때는 빈 문자열이어야 합니다")
	})

	// 테스트 케이스 2: 쿠키 있는 경우
	t.Run("With Cookie", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/test", nil)
		assert.NoError(t, err)

		// 토큰 쿠키 설정
		req.AddCookie(&http.Cookie{Name: "token", Value: "test-jwt-token"})

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "Bearer test-jwt-token", string(body), "쿠키에서 헤더로 변환이 올바르게 되어야 합니다")
	})
}

// TestRateLimiter는 레이트 리미터 기능을 테스트합니다.
func TestRateLimiter(t *testing.T) {
	// Gin 모드 설정
	gin.SetMode(gin.TestMode)

	// 라우터 설정
	router := gin.New()

	// 레이트 리미터 설정 (3초 동안 최대 2번의 요청)
	rateLimiter := middleware.NewRateLimiter(3*time.Second, 2)
	router.Use(middleware.RateLimitMiddleware(rateLimiter))

	// 테스트 핸들러
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// 테스트 서버 생성
	server := httptest.NewServer(router)
	defer server.Close()

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 요청 1: 성공해야 함
	t.Run("First Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1") // IP 설정
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "첫 번째 요청은 성공해야 합니다")
	})

	// 요청 2: 아직 제한에 걸리지 않아야 함
	t.Run("Second Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1") // 같은 IP
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "두 번째 요청도 성공해야 합니다")
	})

	// 요청 3: 이제 제한에 걸려야 함
	t.Run("Third Request (Limited)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1") // 같은 IP
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "세 번째 요청은 제한에 걸려야 합니다")
	})

	// 대기 후 다시 요청 (제한이 풀려야 함)
	t.Run("After Wait (Allowed Again)", func(t *testing.T) {
		// 제한 시간보다 조금 더 기다림
		time.Sleep(4 * time.Second)
		
		req, _ := http.NewRequest("GET", server.URL+"/test", nil)
		req.Header.Set("X-Forwarded-For", "127.0.0.1") // 같은 IP
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "대기 후에는 다시 요청이 성공해야 합니다")
	})
}

// TestCORSMiddleware는 CORS 미들웨어를 테스트합니다.
func TestCORSMiddleware(t *testing.T) {
	// Gin 모드 설정
	gin.SetMode(gin.TestMode)

	// 라우터 설정
	router := gin.New()
	router.Use(middleware.CORSMiddleware())

	// 테스트 핸들러
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// 테스트 서버 생성
	server := httptest.NewServer(router)
	defer server.Close()

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 테스트 케이스: CORS 헤더 확인
	t.Run("CORS Headers", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/test", nil)
		assert.NoError(t, err)
		
		// Origin 헤더 설정
		req.Header.Set("Origin", "http://example.com")
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		// CORS 헤더 확인
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"), "Access-Control-Allow-Origin 헤더가 설정되어야 합니다")
		assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"), "Access-Control-Allow-Credentials 헤더가 설정되어야 합니다")
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"), "Access-Control-Allow-Headers 헤더가 설정되어야 합니다")
		assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"), "Access-Control-Allow-Methods 헤더가 설정되어야 합니다")
	})

	// OPTIONS 메서드 테스트
	t.Run("OPTIONS Request", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", server.URL+"/test", nil)
		assert.NoError(t, err)
		
		// Origin 헤더 설정
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		// 상태 코드와 헤더 확인
		assert.Equal(t, 204, resp.StatusCode, "OPTIONS 요청은 204 상태코드를 반환해야 합니다")
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"), "Access-Control-Allow-Origin 헤더가 설정되어야 합니다")
	})
}

// TestContentSizeLimit는 요청 크기 제한 미들웨어를 테스트합니다.
func TestContentSizeLimit(t *testing.T) {
	// Gin 모드 설정
	gin.SetMode(gin.TestMode)

	// 최대 크기 설정 (100 바이트)
	maxSize := int64(100)

	// 라우터 설정
	router := gin.New()
	router.Use(middleware.ContentSizeLimit(maxSize))

	// 테스트 핸들러
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	// 테스트 서버 생성
	server := httptest.NewServer(router)
	defer server.Close()

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 테스트 케이스 1: 허용된 크기 요청
	t.Run("Allowed Size", func(t *testing.T) {
		// 허용된 크기의 데이터 생성 (50 바이트)
		data := make([]byte, 50)
		for i := range data {
			data[i] = 'A'
		}
		
		req, err := http.NewRequest("POST", server.URL+"/test", bytes.NewBuffer(data))
		assert.NoError(t, err)
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode, "허용된 크기의 요청은 성공해야 합니다")
	})

	// 테스트 케이스 2: 제한 초과 요청
	t.Run("Exceeded Size", func(t *testing.T) {
		// 제한을 초과하는 데이터 생성 (200 바이트)
		data := make([]byte, 200)
		for i := range data {
			data[i] = 'A'
		}
		
		req, err := http.NewRequest("POST", server.URL+"/test", bytes.NewBuffer(data))
		assert.NoError(t, err)
		
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "크기 제한을 초과한 요청은 413 상태코드를 반환해야 합니다")
	})
} 