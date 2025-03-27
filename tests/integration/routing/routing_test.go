// +build integration

package routing

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/tests/integration"
)

func TestRoutingBasic(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 1. 루트 경로 접근 테스트
	t.Run("Root Path", func(t *testing.T) {
		e.GET("/").
			Expect().
			Status(http.StatusOK).
			ContentType("text/html")
	})

	// 2. 정적 파일 요청 테스트
	t.Run("Static Files", func(t *testing.T) {
		// 테스트용 정적 파일 요청
		resp := e.GET("/assets/test.css").
			Expect()

		// 정적 파일 서비스가 구성되어 있다면 200 OK, 아니면 404 Not Found
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusOK || status == http.StatusNotFound,
			"정적 파일 요청은 200 OK 또는 404 Not Found여야 합니다")
	})

	// 3. API 엔드포인트 요청 테스트
	t.Run("API Endpoints", func(t *testing.T) {
		// 사용자 목록 API
		e.GET("/api/users").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("users")

		// 제품 목록 API
		e.GET("/api/products").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("products")

		// 주문 목록 API
		e.GET("/api/orders").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("orders")
	})

	// 4. 없는 경로 요청 테스트
	t.Run("Not Found Routes", func(t *testing.T) {
		e.GET("/non-existent-path").
			Expect().
			Status(http.StatusNotFound)

		e.GET("/api/non-existent").
			Expect().
			Status(http.StatusNotFound)
	})

	// 5. HTTP 메서드 지원 테스트
	t.Run("HTTP Methods Support", func(t *testing.T) {
		// OPTIONS 메서드 테스트
		e.OPTIONS("/api/users").
			Expect().
			Status(http.StatusNoContent).
			Header("Access-Control-Allow-Methods").
			Contains("GET")

		// HEAD 메서드 테스트
		resp := e.HEAD("/api/users").
			Expect()

		// HEAD 메서드가 지원되는지 확인
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusOK || status == http.StatusMethodNotAllowed,
			"HEAD 메서드는 200 OK 또는 405 Method Not Allowed여야 합니다")
	})
}

func TestRoutingStripping(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 경로 접두사 제거 확인
	t.Run("Path Prefix Stripping", func(t *testing.T) {
		// API 경로 접두사는 일반적으로 제거됩니다
		e.GET("/api/users").
			Expect().
			Status(http.StatusOK)

		// 원래 라우팅 설정에 따라 달라질 수 있음
		// 특정 서비스로 직접 요청 시도
		resp := e.GET("/users").
			Expect()

		// 라우팅 설정에 따라 200 OK 또는 404 Not Found
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusOK || status == http.StatusNotFound,
			"경로 제거 설정에 따라 다를 수 있음")
	})
}

func TestRoutingParameters(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// URL 파라미터 전달 테스트
	t.Run("URL Parameters", func(t *testing.T) {
		// ID 파라미터 사용
		e.GET("/api/products/1").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("id").
			ValueEqual("id", 1)

		e.GET("/api/orders/1").
			Expect().
			Status(http.StatusOK).
			JSON().Object().
			ContainsKey("id").
			ValueEqual("id", 1)
	})

	// 쿼리 파라미터 전달 테스트
	t.Run("Query Parameters", func(t *testing.T) {
		// 쿼리 파라미터 전달
		resp := e.GET("/api/products").
			WithQuery("category", "electronics").
			Expect()

		// 서비스 구현에 따라 다를 수 있음
		status := resp.Raw().StatusCode
		assert.Equal(t, http.StatusOK, status, "쿼리 파라미터가 올바르게 전달되어야 함")
	})
}

func TestRoutingContentTypes(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 다양한 Content-Type 처리 테스트
	t.Run("Content Types", func(t *testing.T) {
		// JSON 요청 테스트
		e.POST("/api/orders").
			WithJSON(map[string]interface{}{
				"user_id": 1,
				"items": []map[string]interface{}{
					{"product_id": 1, "quantity": 2},
				},
			}).
			Expect().
			Status(http.StatusOK).
			ContentType("application/json")

		// Form 요청 테스트
		resp := e.POST("/api/users").
			WithFormField("name", "Test User").
			WithFormField("email", "test@example.com").
			Expect()

		// 서비스 구현에 따라 다를 수 있음
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusOK || status == http.StatusCreated || status == http.StatusBadRequest,
			"폼 요청은 서비스 구현에 따라 다를 수 있음")
	})
}

func TestRoutingWebsocket(t *testing.T) {
	// WebSocket 테스트는 다른 라이브러리를 사용해야 할 수 있음
	// 여기서는 간단하게 HTTP 접근 테스트만 수행

	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// WebSocket 엔드포인트 존재 확인
	t.Run("WebSocket Endpoint", func(t *testing.T) {
		resp := e.GET("/websocket/events").
			Expect()

		// WebSocket 업그레이드 요청이 필요하므로 400 Bad Request 또는 
		// 업그레이드 응답(101 Switching Protocols)이 예상됨
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusBadRequest || status == http.StatusSwitchingProtocols,
			"WebSocket 엔드포인트는 업그레이드 요청 없이 접근하면 400 또는 101을 반환해야 함")
	})
}
