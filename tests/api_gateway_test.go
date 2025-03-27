package tests

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
)

const (
	gatewayBaseURL = "http://localhost:8080"
	waitTimeout    = 30 * time.Second
)

func TestAPIGateway(t *testing.T) {
	// 환경 확인 및 설정
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = gatewayBaseURL
	}

	// API Gateway가 준비될 때까지 대기
	err := waitForService(gatewayURL, waitTimeout)
	if err != nil {
		t.Fatalf("API Gateway를 사용할 수 없음: %v", err)
	}

	// httpexpect 인스턴스 생성
	e := httpexpect.New(t, gatewayURL)

	// 헬스 체크 테스트
	t.Run("Health Check", func(t *testing.T) {
		e.GET("/health").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("status").
			ValueEqual("status", "ok")
	})

	// 라우팅 테스트 - 루트 경로
	t.Run("Root Path", func(t *testing.T) {
		e.GET("/").
			Expect().
			Status(http.StatusOK).
			ContentType("text/html")
	})

	// 라우팅 테스트 - API 경로
	t.Run("API Routes", func(t *testing.T) {
		// 사용자 API
		t.Run("Users API", func(t *testing.T) {
			e.GET("/api/users").
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				ContainsKey("users").
				Value("users").Array().Length().Gt(0)
		})

		// 제품 API
		t.Run("Products API", func(t *testing.T) {
			e.GET("/api/products").
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				ContainsKey("products").
				Value("products").Array().Length().Gt(0)

			// 특정 제품 조회
			e.GET("/api/products/1").
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				ContainsKey("id").
				ContainsKey("name").
				ContainsKey("price")
		})

		// 주문 API
		t.Run("Orders API", func(t *testing.T) {
			e.GET("/api/orders").
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				ContainsKey("orders").
				Value("orders").Array().Length().Gt(0)

			// 특정 주문 조회
			e.GET("/api/orders/1").
				Expect().
				Status(http.StatusOK).
				JSON().Object().
				ContainsKey("id").
				ContainsKey("user_id").
				ContainsKey("total").
				ContainsKey("status")
		})
	})

	// 속도 제한 테스트
	t.Run("Rate Limiting", func(t *testing.T) {
		// 연속 요청으로 속도 제한 테스트
		lastStatus := 0
		tooManyRequestsOccurred := false

		for i := 0; i < 300; i++ {
			resp := e.GET("/api/users").
				Expect().
				Raw()

			lastStatus = resp.StatusCode
			if lastStatus == http.StatusTooManyRequests {
				tooManyRequestsOccurred = true
				break
			}
		}

		assert.True(t, tooManyRequestsOccurred, "속도 제한이 활성화되지 않음")
	})

	// 존재하지 않는 경로 테스트
	t.Run("Not Found Routes", func(t *testing.T) {
		e.GET("/api/non-existent").
			Expect().
			Status(http.StatusNotFound)
	})
}

// waitForService는 서비스가 준비될 때까지 대기합니다.
func waitForService(url string, timeout time.Duration) error {
	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}

	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("서비스 대기 시간 초과 (%s)", url)
		}

		resp, err := client.Get(url + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		// 1초 대기 후 재시도
		time.Sleep(1 * time.Second)
	}
}
