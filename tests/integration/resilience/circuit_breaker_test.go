// +build integration

package resilience

import (
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/tests/integration"
	"github.com/isinthesky/api-gateway/tests/mocks"
)

func TestCircuitBreaker(t *testing.T) {
	// 테스트 서버 설정 
	mockServer := mocks.NewMockServer(t, mocks.WithServerOptions{
		EnableFaultInjection: true,
	})
	defer mockServer.Close()

	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 서킷 브레이커 테스트
	t.Run("CircuitBreakerTripping", func(t *testing.T) {
		// 오류 주입 활성화
		mockServer.EnableFault(500, 0)

		// 연속해서 여러 번 요청하면 서킷 브레이커가 열려야 함
		var lastStatus int
		tripped := false
		for i := 0; i < 10; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			lastStatus = resp.Raw().StatusCode
			if lastStatus == http.StatusServiceUnavailable {
				tripped = true
				break
			}

			// 약간의 지연 추가 (서킷 브레이커 상태 업데이트 시간 허용)
			time.Sleep(100 * time.Millisecond)
		}

		assert.True(t, tripped, "연속된 오류 후 서킷 브레이커가 열려야 함")

		// 오류 주입 비활성화
		mockServer.DisableFault()

		// 서킷 브레이커가 열린 상태에서는 요청이 바로 실패해야 함
		resp := e.GET("/api/fault-test").
			Expect()

		assert.Equal(t, http.StatusServiceUnavailable, resp.Raw().StatusCode,
			"서킷 브레이커가 열린 상태에서는 요청이 즉시 실패해야 함")

		// 타임아웃 대기 (서킷 브레이커 타임아웃 동안)
		time.Sleep(5 * time.Second)

		// 타임아웃 후에는 반열림 상태가 되어 일부 요청이 허용되어야 함
		resp = e.GET("/api/fault-test").
			Expect()

		assert.Equal(t, http.StatusOK, resp.Raw().StatusCode,
			"타임아웃 후에는 요청이 성공해야 함 (반열림 상태)")
	})

	t.Run("CircuitBreakerRecovery", func(t *testing.T) {
		// 오류 주입 활성화
		mockServer.EnableFault(500, 0)

		// 서킷 브레이커가 열리도록 여러 번 요청
		for i := 0; i < 10; i++ {
			_ = e.GET("/api/fault-test").Expect()
			time.Sleep(100 * time.Millisecond)
		}

		// 서킷 브레이커가 열렸는지 확인
		resp := e.GET("/api/fault-test").
			Expect()
		assert.Equal(t, http.StatusServiceUnavailable, resp.Raw().StatusCode,
			"서킷 브레이커가 열려야 함")

		// 오류 주입 비활성화 (정상 작동으로 복원)
		mockServer.DisableFault()

		// 타임아웃 대기
		time.Sleep(5 * time.Second)

		// 서비스가 정상화된 상태에서 연속 성공 요청
		for i := 0; i < 5; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			assert.Equal(t, http.StatusOK, resp.Raw().StatusCode,
				"%d번째 회복 요청이 성공해야 함", i+1)
			time.Sleep(100 * time.Millisecond) // 약간의 지연
		}

		// 서킷 브레이커가 정상 상태로 복구되었는지 확인
		for i := 0; i < 10; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			assert.Equal(t, http.StatusOK, resp.Raw().StatusCode,
				"서킷 브레이커 복구 후에는 모든 요청이 성공해야 함")
		}
	})

	t.Run("CircuitBreakerPartialFailure", func(t *testing.T) {
		// 부분 오류 주입 (50% 확률로 오류)
		mockServer.EnableFault(500, 0.5)

		// 부분 오류 상황에서 여러 번 요청
		successCount := 0
		failureCount := 0

		for i := 0; i < 20; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			if resp.Raw().StatusCode == http.StatusOK {
				successCount++
			} else {
				failureCount++
			}

			time.Sleep(100 * time.Millisecond)
		}

		// 부분 성공 및 실패 확인
		assert.True(t, successCount > 0, "일부 요청은 성공해야 함")
		assert.True(t, failureCount > 0, "일부 요청은 실패해야 함")

		// 오류 주입 비활성화
		mockServer.DisableFault()

		// 정상화 대기
		time.Sleep(5 * time.Second)

		// 복구 확인
		for i := 0; i < 5; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			assert.Equal(t, http.StatusOK, resp.Raw().StatusCode,
				"정상화 후에는 모든 요청이 성공해야 함")
		}
	})
}
