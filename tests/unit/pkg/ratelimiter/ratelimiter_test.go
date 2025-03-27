// +build unit

package ratelimiter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/pkg/ratelimiter"
)

func TestTokenBucket(t *testing.T) {
	t.Run("BasicLimit", func(t *testing.T) {
		// 2초 동안 3개 요청 허용하는 레이트 리미터 생성
		limiter := ratelimiter.New(2*time.Second, 3)
		defer limiter.Stop()

		// 같은 키로 요청 실행
		key := "test-client"

		// 처음 3개 요청은 허용되어야 함
		for i := 0; i < 3; i++ {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "처음 %d번째 요청은 허용되어야 함", i+1)
		}

		// 4번째 요청은 거부되어야 함
		allowed := limiter.Allow(key)
		assert.False(t, allowed, "제한을 초과하는 요청은 거부되어야 함")

		// 일정 시간 후 토큰이 보충되어야 함
		time.Sleep(1 * time.Second) // 부분 보충
		allowed = limiter.Allow(key)
		assert.True(t, allowed, "토큰 보충 후 요청은 허용되어야 함")
	})

	t.Run("DifferentKeys", func(t *testing.T) {
		// 1초 동안 2개 요청 허용하는 레이트 리미터 생성
		limiter := ratelimiter.New(1*time.Second, 2)
		defer limiter.Stop()

		// 서로 다른 키로 요청 실행
		key1 := "client-1"
		key2 := "client-2"

		// 각 클라이언트는 독립적으로 제한되어야 함
		for i := 0; i < 2; i++ {
			allowed1 := limiter.Allow(key1)
			allowed2 := limiter.Allow(key2)
			assert.True(t, allowed1, "client-1의 %d번째 요청은 허용되어야 함", i+1)
			assert.True(t, allowed2, "client-2의 %d번째 요청은 허용되어야 함", i+1)
		}

		// 각 클라이언트의 3번째 요청은 거부되어야 함
		allowed1 := limiter.Allow(key1)
		allowed2 := limiter.Allow(key2)
		assert.False(t, allowed1, "client-1의 3번째 요청은 거부되어야 함")
		assert.False(t, allowed2, "client-2의 3번째 요청은 거부되어야 함")
	})

	t.Run("TokenRefill", func(t *testing.T) {
		// 빠른 테스트를 위한 짧은 시간 설정
		// 100ms 동안 5개 요청 허용
		limiter := ratelimiter.New(100*time.Millisecond, 5)
		defer limiter.Stop()

		key := "refill-test"

		// 모든 토큰 소진
		for i := 0; i < 5; i++ {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "처음 %d번째 요청은 허용되어야 함", i+1)
		}

		// 토큰 소진 확인
		allowed := limiter.Allow(key)
		assert.False(t, allowed, "토큰이 소진된 후 요청은 거부되어야 함")

		// 일부 토큰 보충 대기
		time.Sleep(60 * time.Millisecond) // 약 3개 토큰 보충

		// 보충된 토큰만큼 허용
		for i := 0; i < 3; i++ {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "보충 후 %d번째 요청은 허용되어야 함", i+1)
		}

		// 추가 요청은 거부
		allowed = limiter.Allow(key)
		assert.False(t, allowed, "보충된 토큰 이상의 요청은 거부되어야 함")

		// 완전 보충 대기
		time.Sleep(150 * time.Millisecond) // 모든 토큰 보충

		// 다시 모든 토큰 사용 가능
		for i := 0; i < 5; i++ {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "완전 보충 후 %d번째 요청은 허용되어야 함", i+1)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		// 1초 동안 3개 요청 허용
		limiter := ratelimiter.New(1*time.Second, 3)
		defer limiter.Stop()

		key := "reset-test"

		// 일부 토큰 사용
		limiter.Allow(key)
		limiter.Allow(key)

		// 남은 토큰 확인
		count, allowed := limiter.Peek(key)
		assert.Equal(t, 1, count, "남은 토큰 수가 정확해야 함")
		assert.True(t, allowed, "아직 요청 가능해야 함")

		// 키 리셋
		limiter.Reset(key)

		// 리셋 후 토큰 확인
		count, allowed = limiter.Peek(key)
		assert.Equal(t, 3, count, "리셋 후 모든 토큰이 복원되어야 함")
		assert.True(t, allowed, "리셋 후 요청 가능해야 함")
	})

	t.Run("AllowN", func(t *testing.T) {
		// 1초 동안 10개 토큰 허용
		limiter := ratelimiter.New(1*time.Second, 10)
		defer limiter.Stop()

		key := "allow-n-test"

		// 5개 토큰 사용
		allowed := limiter.AllowN(key, 5)
		assert.True(t, allowed, "5개 토큰 요청은 허용되어야 함")

		// 남은 토큰 확인
		count, _ := limiter.Peek(key)
		assert.Equal(t, 5, count, "남은 토큰 수가 정확해야 함")

		// 남은 토큰보다 많은 요청
		allowed = limiter.AllowN(key, 6)
		assert.False(t, allowed, "남은 토큰보다 많은 요청은 거부되어야 함")

		// 남은 토큰 수는 변하지 않아야 함
		count, _ = limiter.Peek(key)
		assert.Equal(t, 5, count, "거부된 요청은 토큰을 소비하지 않아야 함")

		// 남은 토큰으로 요청
		allowed = limiter.AllowN(key, 5)
		assert.True(t, allowed, "남은 토큰 수만큼의 요청은 허용되어야 함")

		// 모든 토큰 소진 확인
		count, allowed = limiter.Peek(key)
		assert.Equal(t, 0, count, "모든 토큰이 소진되어야 함")
		assert.False(t, allowed, "토큰이 없으면 요청 불가능해야 함")
	})
}

func TestSlidingWindow(t *testing.T) {
	t.Run("BasicLimit", func(t *testing.T) {
		// 200ms 동안 3개 요청 허용하는 슬라이딩 윈도우 생성
		limiter := ratelimiter.NewSlidingWindow(200*time.Millisecond, 3)
		defer limiter.Stop()

		key := "sliding-test"

		// 처음 3개 요청은 허용되어야 함
		for i := 0; i < 3; i++ {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "처음 %d번째 요청은 허용되어야 함", i+1)
		}

		// 4번째 요청은 거부되어야 함
		allowed := limiter.Allow(key)
		assert.False(t, allowed, "제한을 초과하는 요청은 거부되어야 함")

		// 윈도우 이동 대기
		time.Sleep(150 * time.Millisecond)

		// 일부 요청이 윈도우에서 나가 새 요청이 허용될 수 있음
		allowed = limiter.Allow(key)
		assert.True(t, allowed, "윈도우 이동 후 요청은 허용되어야 함")
	})

	t.Run("ExpiryManagement", func(t *testing.T) {
		// 매우 짧은 윈도우 설정
		limiter := ratelimiter.NewSlidingWindow(100*time.Millisecond, 2)
		defer limiter.Stop()

		key := "expiry-test"

		// 초기 요청 2개
		limiter.Allow(key)
		limiter.Allow(key)

		// 3번째 요청은 거부
		allowed := limiter.Allow(key)
		assert.False(t, allowed, "윈도우 내 요청 초과 시 거부되어야 함")

		// 윈도우 완전히 지난 후 (모든 타임스탬프 만료)
		time.Sleep(150 * time.Millisecond)

		// 새 요청 허용 확인
		allowed = limiter.Allow(key)
		assert.True(t, allowed, "윈도우 경과 후 요청은 허용되어야 함")

		// 두 번째 새 요청 허용 확인
		allowed = limiter.Allow(key)
		assert.True(t, allowed, "윈도우 경과 후 두 번째 요청도 허용되어야 함")

		// 세 번째 새 요청 거부 확인
		allowed = limiter.Allow(key)
		assert.False(t, allowed, "윈도우 내 새 요청 초과 시 거부되어야 함")
	})

	// 이 테스트는 레이트 리미터의 내부 클리너 동작을 검증
	t.Run("AutoCleaning", func(t *testing.T) {
		// 짧은 윈도우 설정 (클리너 동작 확인용)
		limiter := ratelimiter.NewSlidingWindow(50*time.Millisecond, 1)
		defer limiter.Stop()

		// 여러 키로 요청
		keys := []string{"key1", "key2", "key3", "key4", "key5"}
		for _, key := range keys {
			limiter.Allow(key)
		}

		// 클리너가 동작할 시간 기다림 (내부적으로 윈도우의 절반마다 실행)
		time.Sleep(200 * time.Millisecond)

		// 모든 키가 정리되어야 함 (내부 상태 직접 검증은 어렵지만, 새 요청 가능 여부로 검증)
		for _, key := range keys {
			allowed := limiter.Allow(key)
			assert.True(t, allowed, "클리너 동작 후 키 %s의 요청은 허용되어야 함", key)
		}
	})
}
