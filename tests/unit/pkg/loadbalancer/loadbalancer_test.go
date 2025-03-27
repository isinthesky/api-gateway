// +build unit

package loadbalancer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/pkg/loadbalancer"
)

func TestRoundRobinBalancer(t *testing.T) {
	t.Run("SingleTarget", func(t *testing.T) {
		// 단일 대상으로 로드 밸런서 생성
		urls := []string{"http://server1.example.com"}
		lb := loadbalancer.NewRoundRobin(urls)

		// 대상 서버 확인
		target, err := lb.NextTarget()
		assert.NoError(t, err, "단일 대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, "http://server1.example.com", target, "반환된 대상이 정확해야 함")

		// 두 번째 호출도 동일한 대상 반환
		target, err = lb.NextTarget()
		assert.NoError(t, err, "단일 대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, "http://server1.example.com", target, "단일 대상은 항상 동일한 결과를 반환해야 함")
	})

	t.Run("MultipleTargets", func(t *testing.T) {
		// 여러 대상으로 로드 밸런서 생성
		urls := []string{
			"http://server1.example.com",
			"http://server2.example.com",
			"http://server3.example.com",
		}
		lb := loadbalancer.NewRoundRobin(urls)

		// 첫 번째 대상
		target1, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")

		// 두 번째 대상 (다른 대상이어야 함)
		target2, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.NotEqual(t, target1, target2, "다른 대상이 선택되어야 함")

		// 세 번째 대상
		target3, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.NotEqual(t, target2, target3, "다른 대상이 선택되어야 함")
		assert.NotEqual(t, target1, target3, "다른 대상이 선택되어야 함")

		// 네 번째 대상 (첫 번째와 동일해야 함 - 순환)
		target4, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, target1, target4, "네 번째 대상은 첫 번째 대상과 동일해야 함 (순환)")
	})

	t.Run("NoTargets", func(t *testing.T) {
		// 대상 없는 로드 밸런서 생성
		lb := loadbalancer.NewRoundRobin([]string{})

		// 대상 선택 시도
		_, err := lb.NextTarget()
		assert.Error(t, err, "대상이 없을 때 오류가 발생해야 함")
		assert.Equal(t, loadbalancer.ErrNoAvailableTargets, err, "적절한 오류 반환 필요")
	})

	t.Run("HealthCheck", func(t *testing.T) {
		// 여러 대상으로 로드 밸런서 생성
		urls := []string{
			"http://server1.example.com",
			"http://server2.example.com",
			"http://server3.example.com",
		}
		lb := loadbalancer.NewRoundRobin(urls)

		// 첫 번째 서버를 비정상으로 표시
		err := lb.MarkTargetDown("http://server1.example.com")
		assert.NoError(t, err, "대상을 비정상으로 표시할 때 오류가 발생하지 않아야 함")

		// 대상 목록 확인
		targets := lb.GetTargets()
		assert.Equal(t, 3, len(targets), "대상 수는 변하지 않아야 함")

		// 건강한 대상만 선택되어야 함
		for i := 0; i < 10; i++ {
			target, err := lb.NextTarget()
			assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
			assert.NotEqual(t, "http://server1.example.com", target, "비정상 대상은 선택되지 않아야 함")
		}

		// 첫 번째 서버를 정상으로 표시
		err = lb.MarkTargetUp("http://server1.example.com")
		assert.NoError(t, err, "대상을 정상으로 표시할 때 오류가 발생하지 않아야 함")

		// 이제 모든 서버가 선택 가능해야 함
		selected := make(map[string]bool)
		for i := 0; i < 10; i++ {
			target, _ := lb.NextTarget()
			selected[target] = true
		}
		assert.Equal(t, 3, len(selected), "모든 대상이 선택되어야 함")
	})

	t.Run("AddRemoveTarget", func(t *testing.T) {
		// 초기 대상으로 로드 밸런서 생성
		lb := loadbalancer.NewRoundRobin([]string{"http://server1.example.com"})

		// 대상 추가
		err := lb.AddTarget("http://server2.example.com", 1)
		assert.NoError(t, err, "대상 추가에서 오류가 발생하지 않아야 함")

		// 대상 목록 확인
		targets := lb.GetTargets()
		assert.Equal(t, 2, len(targets), "대상이 추가되어야 함")

		// 대상 제거
		err = lb.RemoveTarget("http://server1.example.com")
		assert.NoError(t, err, "대상 제거에서 오류가 발생하지 않아야 함")

		// 대상 목록 확인
		targets = lb.GetTargets()
		assert.Equal(t, 1, len(targets), "대상이 제거되어야 함")
		assert.Equal(t, "http://server2.example.com", targets[0].URL, "남은 대상이 정확해야 함")
	})
}

func TestWeightedRoundRobinBalancer(t *testing.T) {
	t.Run("WeightedDistribution", func(t *testing.T) {
		// 가중치가 있는 대상으로 로드 밸런서 생성
		urlWeights := map[string]int{
			"http://server1.example.com": 1,  // 낮은 가중치
			"http://server2.example.com": 3,  // 높은 가중치
		}
		lb := loadbalancer.NewWeightedRoundRobin(urlWeights)

		// 선택 횟수 계산
		selections := make(map[string]int)
		for i := 0; i < 40; i++ {
			target, err := lb.NextTarget()
			assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
			selections[target]++
		}

		// server2가 server1보다 약 3배 더 자주 선택되어야 함
		assert.True(t, selections["http://server2.example.com"] > selections["http://server1.example.com"],
			"가중치가 높은 서버가 더 자주 선택되어야 함")
		ratio := float64(selections["http://server2.example.com"]) / float64(selections["http://server1.example.com"])
		assert.InDelta(t, 3.0, ratio, 0.5, "선택 비율이 가중치 비율과 근접해야 함")
	})
}

func TestLeastConnectionBalancer(t *testing.T) {
	t.Run("ConnectionBalance", func(t *testing.T) {
		// 여러 대상으로 로드 밸런서 생성
		urls := []string{
			"http://server1.example.com",
			"http://server2.example.com",
			"http://server3.example.com",
		}
		lb := loadbalancer.NewLeastConnection(urls)

		// 첫 번째 선택 - 모든 서버의 연결 수 동일 (0)
		target1, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")

		// 연결 해제 없이 다음 선택 - 첫 번째 서버는 연결 수 증가
		target2, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.NotEqual(t, target1, target2, "다른 대상이 선택되어야 함")

		// 첫 번째 서버의 연결 해제
		loadbalancer.ReleaseConn(lb, target1)

		// 다음 선택 - 첫 번째 서버 선택 (연결 수 가장 적음)
		target3, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, target1, target3, "연결 수가 가장 적은 서버가 선택되어야 함")
	})
}

func TestSingleTarget(t *testing.T) {
	t.Run("BasicOperation", func(t *testing.T) {
		// 단일 대상 로드 밸런서 생성
		lb := loadbalancer.NewSingle("http://server.example.com")

		// 대상 선택
		target, err := lb.NextTarget()
		assert.NoError(t, err, "대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, "http://server.example.com", target, "단일 대상이 정확히 반환되어야 함")

		// 대상 목록 확인
		targets := lb.GetTargets()
		assert.Equal(t, 1, len(targets), "대상이 하나여야 함")
		assert.Equal(t, "http://server.example.com", targets[0].URL, "대상 URL이 정확해야 함")

		// 대상 상태 변경
		err = lb.MarkTargetDown("http://server.example.com")
		assert.NoError(t, err, "대상 상태 변경에서 오류가 발생하지 않아야 함")

		// 건강하지 않은 상태에서 선택 시도
		_, err = lb.NextTarget()
		assert.Equal(t, loadbalancer.ErrNoAvailableTargets, err, "건강하지 않은 대상은 선택되지 않아야 함")

		// 대상 상태 복구
		err = lb.MarkTargetUp("http://server.example.com")
		assert.NoError(t, err, "대상 상태 복구에서 오류가 발생하지 않아야 함")

		// 복구 후 선택 시도
		target, err = lb.NextTarget()
		assert.NoError(t, err, "복구 후 대상 선택에서 오류가 발생하지 않아야 함")
		assert.Equal(t, "http://server.example.com", target, "단일 대상이 정확히 반환되어야 함")
	})

	t.Run("UnsupportedOperations", func(t *testing.T) {
		// 단일 대상 로드 밸런서 생성
		lb := loadbalancer.NewSingle("http://server.example.com")

		// 대상 추가 시도 (지원되지 않음)
		err := lb.AddTarget("http://another.example.com", 1)
		assert.Error(t, err, "지원되지 않는 작업에서 오류가 발생해야 함")

		// 대상 제거 시도 (지원되지 않음)
		err = lb.RemoveTarget("http://server.example.com")
		assert.Error(t, err, "지원되지 않는 작업에서 오류가 발생해야 함")
	})
}
