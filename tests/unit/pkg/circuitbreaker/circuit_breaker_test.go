// +build unit

package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/pkg/circuitbreaker"
)

// 테스트용 오류
var errTest = errors.New("test error")

// 항상 성공하는 함수
func successFunc() (interface{}, error) {
	return "success", nil
}

// 항상 실패하는 함수
func failureFunc() (interface{}, error) {
	return nil, errTest
}

func TestCircuitBreakerBasic(t *testing.T) {
	t.Run("InitialState", func(t *testing.T) {
		// 기본 설정으로 서킷 브레이커 생성
		cb := circuitbreaker.New(circuitbreaker.Config{
			ErrorThreshold:   0.5,  // 50% 오류율
			MinRequests:      5,    // 최소 5개 요청
			TimeoutDuration:  1 * time.Second,
			HalfOpenMaxReqs:  2,
			SuccessThreshold: 2,
		})

		// 초기 상태는 닫힘 상태
		assert.Equal(t, "closed", cb.GetState(), "초기 상태는 닫힘이어야 함")

		// 닫힘 상태에서는 요청이 정상 처리되어야 함
		result, err := cb.Execute(successFunc)
		assert.NoError(t, err, "성공 함수는 오류를 반환하지 않아야 함")
		assert.Equal(t, "success", result, "함수는 예상된 결과를 반환해야 함")
	})

	t.Run("OpenState", func(t *testing.T) {
		// 짧은 타임아웃을 가진 서킷 브레이커 생성
		cb := circuitbreaker.New(circuitbreaker.Config{
			ErrorThreshold:   0.5,  // 50% 오류율
			MinRequests:      2,    // 최소 2개 요청
			TimeoutDuration:  500 * time.Millisecond,
			HalfOpenMaxReqs:  2,
			SuccessThreshold: 2,
		})

		// 오류 발생시키기
		for i := 0; i < 3; i++ {
			_, _ = cb.Execute(failureFunc)
		}

		// 서킷 브레이커 상태 확인
		assert.Equal(t, "open", cb.GetState(), "충분한 오류 후에는 열림 상태가 되어야 함")

		// 열림 상태에서는 즉시 실패해야 함
		_, err := cb.Execute(successFunc)
		assert.Equal(t, circuitbreaker.ErrCircuitOpen, err, "열림 상태에서는 회로 열림 오류를 반환해야 함")

		// 타임아웃 대기
		time.Sleep(600 * time.Millisecond)

		// 타임아웃 후에는 반열림 상태로 전환되어 요청이 허용되어야 함
		result, err := cb.Execute(successFunc)
		assert.NoError(t, err, "타임아웃 후에는 요청이 허용되어야 함")
		assert.Equal(t, "success", result, "함수는 예상된 결과를 반환해야 함")
	})

	t.Run("HalfOpenState", func(t *testing.T) {
		// 서킷 브레이커 생성
		cb := circuitbreaker.New(circuitbreaker.Config{
			ErrorThreshold:   0.5,  // 50% 오류율
			MinRequests:      2,    // 최소 2개 요청
			TimeoutDuration:  500 * time.Millisecond,
			HalfOpenMaxReqs:  3,    // 반열림 상태에서 최대 3개 요청
			SuccessThreshold: 2,    // 2번 연속 성공 시 닫힘 상태로 전환
		})

		// 오류 발생시켜 열림 상태로 전환
		for i := 0; i < 3; i++ {
			_, _ = cb.Execute(failureFunc)
		}

		// 타임아웃 대기
		time.Sleep(600 * time.Millisecond)

		// 반열림 상태에서 성공 요청
		_, _ = cb.Execute(successFunc)
		_, _ = cb.Execute(successFunc)

		// 연속 성공으로 닫힘 상태로 전환되어야 함
		assert.Equal(t, "closed", cb.GetState(), "연속 성공 후에는 닫힘 상태로 전환되어야 함")

		// 반열림 상태에서 실패하면 다시 열림 상태로 전환
		cb.Reset()
		// 오류 발생시켜 열림 상태로 전환
		for i := 0; i < 3; i++ {
			_, _ = cb.Execute(failureFunc)
		}

		// 타임아웃 대기
		time.Sleep(600 * time.Millisecond)

		// 반열림 상태에서 실패 요청
		_, _ = cb.Execute(failureFunc)

		// 실패로 다시 열림 상태로 전환되어야 함
		assert.Equal(t, "open", cb.GetState(), "실패 후에는 다시 열림 상태로 전환되어야 함")
	})
}

func TestCircuitBreakerMetrics(t *testing.T) {
	// 서킷 브레이커 생성
	cb := circuitbreaker.New(circuitbreaker.Config{
		ErrorThreshold:   0.5,  // 50% 오류율
		MinRequests:      5,    // 최소 5개 요청
		TimeoutDuration:  1 * time.Second,
		HalfOpenMaxReqs:  2,
		SuccessThreshold: 2,
	})

	// 메트릭 초기 상태 확인
	metrics := cb.GetMetrics()
	assert.Equal(t, int64(0), metrics["success_count"], "초기 성공 횟수는 0이어야 함")
	assert.Equal(t, int64(0), metrics["failure_count"], "초기 실패 횟수는 0이어야 함")
	assert.Equal(t, float64(0), metrics["error_rate"], "초기 오류율은 0이어야 함")

	// 요청 실행
	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(successFunc)
	}
	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(failureFunc)
	}

	// 메트릭 업데이트 확인
	metrics = cb.GetMetrics()
	assert.Equal(t, int64(3), metrics["success_count"], "성공 횟수는 3이어야 함")
	assert.Equal(t, int64(2), metrics["failure_count"], "실패 횟수는 2이어야 함")
	assert.Equal(t, float64(2)/float64(5), metrics["error_rate"], "오류율은 2/5여야 함")
}

func TestCircuitBreakerReset(t *testing.T) {
	// 서킷 브레이커 생성
	cb := circuitbreaker.New(circuitbreaker.Config{
		ErrorThreshold:   0.5,  // 50% 오류율
		MinRequests:      2,    // 최소 2개 요청
		TimeoutDuration:  1 * time.Second,
		HalfOpenMaxReqs:  2,
		SuccessThreshold: 2,
	})

	// 오류 발생시켜 열림 상태로 전환
	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(failureFunc)
	}

	// 열림 상태 확인
	assert.Equal(t, "open", cb.GetState(), "오류 후에는 열림 상태가 되어야 함")

	// 메트릭 확인
	metrics := cb.GetMetrics()
	assert.True(t, metrics["failure_count"].(int64) > 0, "실패 횟수가 0보다 커야 함")

	// 리셋
	cb.Reset()

	// 리셋 후 상태 확인
	assert.Equal(t, "closed", cb.GetState(), "리셋 후에는 닫힘 상태가 되어야 함")

	// 리셋 후 메트릭 확인
	metrics = cb.GetMetrics()
	assert.Equal(t, int64(0), metrics["success_count"], "리셋 후 성공 횟수는 0이어야 함")
	assert.Equal(t, int64(0), metrics["failure_count"], "리셋 후 실패 횟수는 0이어야 함")
	assert.Equal(t, float64(0), metrics["error_rate"], "리셋 후 오류율은 0이어야 함")
}

func TestCircuitBreakerCustomConfig(t *testing.T) {
	// 사용자 정의 설정으로 서킷 브레이커 생성
	cb := circuitbreaker.New(circuitbreaker.Config{
		ErrorThreshold:   0.3,             // 30% 오류율
		MinRequests:      10,              // 최소 10개 요청
		TimeoutDuration:  2 * time.Second, // 2초 타임아웃
		HalfOpenMaxReqs:  5,               // 반열림 상태에서 최대 5개 요청
		SuccessThreshold: 3,               // 3번 연속 성공 시 닫힘 상태로 전환
	})

	// 초기 설정 확인
	metrics := cb.GetMetrics()
	assert.Equal(t, float64(0.3), cb.Config().ErrorThreshold, "오류 임계값이 정확해야 함")
	assert.Equal(t, 10, cb.Config().MinRequests, "최소 요청 수가 정확해야 함")
	assert.Equal(t, 2*time.Second, cb.Config().TimeoutDuration, "타임아웃이 정확해야 함")
	assert.Equal(t, 5, cb.Config().HalfOpenMaxReqs, "반열림 최대 요청 수가 정확해야 함")
	assert.Equal(t, 3, cb.Config().SuccessThreshold, "성공 임계값이 정확해야 함")

	// 기능 테스트
	// 9개의 실패 요청 (최소 요청 수 미달)
	for i := 0; i < 9; i++ {
		_, _ = cb.Execute(failureFunc)
	}

	// 최소 요청 수 미달로 닫힘 상태 유지
	assert.Equal(t, "closed", cb.GetState(), "최소 요청 수 미달로 닫힘 상태가 유지되어야 함")

	// 10번째 실패 요청 (최소 요청 수 충족, 오류율 100% > 30%)
	_, _ = cb.Execute(failureFunc)

	// 열림 상태로 전환
	assert.Equal(t, "open", cb.GetState(), "오류율 초과로 열림 상태가 되어야 함")

	// 문자열 포맷팅 확인
	stateStr := cb.FormatState()
	assert.Contains(t, stateStr, "state: open", "상태 문자열에 'open' 포함되어야 함")
	assert.Contains(t, stateStr, "error rate:", "상태 문자열에 'error rate:' 포함되어야 함")
}
