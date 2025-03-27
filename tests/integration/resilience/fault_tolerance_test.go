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

func TestFaultTolerance(t *testing.T) {
	// 테스트 서버 설정
	mockServer := mocks.NewMockServer(t, mocks.WithServerOptions{
		EnableFaultInjection: true,
	})
	defer mockServer.Close()

	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 타임아웃 테스트
	t.Run("Timeouts", func(t *testing.T) {
		// 지연 주입 (5초 지연)
		mockServer.EnableDelay(5 * time.Second)

		// 요청 실행 (게이트웨이의 타임아웃이 5초 미만이어야 함)
		resp := e.GET("/api/delay-test").
			Expect()

		// 게이트웨이 타임아웃 확인
		assert.Equal(t, http.StatusGatewayTimeout, resp.Raw().StatusCode,
			"긴 지연 시간에는 게이트웨이 타임아웃이 발생해야 함")

		// 지연 비활성화
		mockServer.DisableDelay()

		// 정상 요청 확인
		resp = e.GET("/api/delay-test").
			Expect()

		assert.Equal(t, http.StatusOK, resp.Raw().StatusCode,
			"지연 없이는 요청이 성공해야 함")
	})

	// 부하 분산 테스트
	t.Run("LoadBalancing", func(t *testing.T) {
		// 여러 백엔드 서버 시뮬레이션
		server1 := mocks.NewMockServer(t, mocks.WithServerOptions{
			ServerID: "server1",
		})
		defer server1.Close()

		server2 := mocks.NewMockServer(t, mocks.WithServerOptions{
			ServerID: "server2",
		})
		defer server2.Close()

		// 여러 요청을 실행하여 부하 분산 확인
		serverResponses := make(map[string]int)
		
		for i := 0; i < 10; i++ {
			resp := e.GET("/api/server-id").
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			serverID := resp.Value("server_id").String().Raw()
			serverResponses[serverID]++
		}

		// 여러 서버가 사용되었는지 확인
		assert.True(t, len(serverResponses) > 1, "요청이 여러 서버로 분산되어야 함")
	})

	// 실패한 서버 처리 테스트
	t.Run("FailedServerHandling", func(t *testing.T) {
		// 첫 번째 서버에 오류 주입
		mockServer.EnableFault(500, 1.0)

		// 여러 요청 실행
		successCount := 0
		for i := 0; i < 10; i++ {
			resp := e.GET("/api/fault-test").
				Expect()

			if resp.Raw().StatusCode == http.StatusOK {
				successCount++
			}

			time.Sleep(100 * time.Millisecond)
		}

		// 일부 요청은 성공해야 함 (다른 서버로 라우팅된 경우)
		assert.True(t, successCount > 0, "일부 요청은 성공해야 함 (부하 분산)")

		// 오류 주입 비활성화
		mockServer.DisableFault()
	})

	// 네트워크 파티션 시뮬레이션
	t.Run("NetworkPartition", func(t *testing.T) {
		// 서버 중단 시뮬레이션
		mockServer.Shutdown()

		// 요청 실행 (서버가 중단되었으므로 실패 예상)
		resp := e.GET("/api/test").
			Expect()

		// 서버 중단 시 적절한 오류 코드 확인
		assert.True(t, resp.Raw().StatusCode >= 500, "서버 중단 시 5xx 오류가 발생해야 함")

		// 새 서버 시작
		newServer := mocks.NewMockServer(t, mocks.WithServerOptions{})
		defer newServer.Close()

		// 잠시 대기 (서버 감지 시간)
		time.Sleep(2 * time.Second)

		// 복구 후 요청 확인
		resp = e.GET("/api/test").
			Expect()

		assert.Equal(t, http.StatusOK, resp.Raw().StatusCode, "새 서버 시작 후 요청이 성공해야 함")
	})

	// 리트라이 매커니즘 테스트
	t.Run("RetryMechanism", func(t *testing.T) {
		// 일시적 오류 주입 (처음 2번 실패, 그 다음 성공)
		mockServer.EnableTemporaryFault(500, 2)

		// 요청 실행 (리트라이 매커니즘이 있는 경우 성공해야 함)
		start := time.Now()
		resp := e.GET("/api/retry-test").
			Expect()

		duration := time.Since(start)

		// 리트라이 시간 확인 (일정 시간 이상 소요되어야 함)
		assert.True(t, duration >= 300*time.Millisecond, "리트라이 시간이 충분해야 함")

		// 최종 요청 상태 확인
		assert.Equal(t, http.StatusOK, resp.Raw().StatusCode, "리트라이 후 요청이 성공해야 함")

		// 일시적 오류 비활성화
		mockServer.DisableTemporaryFault()
	})
}
