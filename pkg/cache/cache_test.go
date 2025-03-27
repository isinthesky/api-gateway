//go:build unit
// +build unit

package cache

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		// 캐시 초기화
		cache := New(1 * time.Second)
		defer cache.Close()

		// 테스트 데이터
		key := "test-key"
		headers := make(http.Header)
		headers.Add("Content-Type", "application/json")
		cachedResp := &CachedResponse{
			StatusCode: 200,
			Headers:    headers,
			Body:       []byte(`{"message":"Hello, World!"}`),
		}

		// 캐시에 아이템 저장
		cache.Set(key, cachedResp, 0) // 기본 TTL 사용

		// 캐시에서 아이템 조회
		retrieved, found := cache.Get(key)
		assert.True(t, found, "캐시에서 아이템을 찾을 수 있어야 함")
		assert.Equal(t, cachedResp.StatusCode, retrieved.StatusCode, "상태 코드 일치해야 함")
		assert.Equal(t, cachedResp.Headers.Get("Content-Type"), retrieved.Headers.Get("Content-Type"), "헤더 일치해야 함")
		assert.Equal(t, cachedResp.Body, retrieved.Body, "본문 일치해야 함")

		// 캐시에서 아이템 삭제
		cache.Delete(key)
		_, found = cache.Get(key)
		assert.False(t, found, "삭제 후 아이템을 찾을 수 없어야 함")
	})

	t.Run("Expiry", func(t *testing.T) {
		// 짧은 TTL 설정
		cache := New(100 * time.Millisecond)
		defer cache.Close()

		// 테스트 데이터
		key := "expiry-test"
		cachedResp := &CachedResponse{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte("Expiry Test"),
		}

		// 캐시에 아이템 저장
		cache.Set(key, cachedResp, 0) // 기본 TTL 사용 (100ms)

		// 즉시 조회 - 찾아야 함
		_, found := cache.Get(key)
		assert.True(t, found, "캐시에 저장 후 바로 아이템을 찾을 수 있어야 함")

		// TTL 경과 후 조회 - 만료되어 찾을 수 없어야 함
		time.Sleep(200 * time.Millisecond)
		_, found = cache.Get(key)
		assert.False(t, found, "만료 후 아이템을 찾을 수 없어야 함")
	})

	t.Run("CustomTTL", func(t *testing.T) {
		// 기본 TTL 설정
		cache := New(1 * time.Second)
		defer cache.Close()

		// 테스트 데이터
		key1 := "short-ttl"
		key2 := "long-ttl"
		cachedResp := &CachedResponse{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte("TTL Test"),
		}

		start := time.Now()
		t.Logf("테스트 시작: %v", start)

		// 다른 TTL로 저장
		shortTTL := 50 * time.Millisecond
		longTTL := 300 * time.Millisecond
		cache.Set(key1, cachedResp, shortTTL)  // 짧은 TTL
		cache.Set(key2, cachedResp, longTTL) // 긴 TTL
		t.Logf("아이템 저장 완료: %v", time.Since(start))

		// 짧은 TTL 경과 전 확인
		_, found1 := cache.Get(key1)
		_, found2 := cache.Get(key2)
		assert.True(t, found1, "짧은 TTL 아이템이 즉시 조회되어야 함")
		assert.True(t, found2, "긴 TTL 아이템이 즉시 조회되어야 함")
		t.Logf("초기 확인 완료: %v", time.Since(start))

		// 짧은 TTL 경과 후 조회
		sleepTime := 300 * time.Millisecond // 50ms TTL의 2배로 설정
		t.Logf("대기 시작 (%v): %v", sleepTime, time.Since(start))
		time.Sleep(sleepTime)
		t.Logf("대기 완료: %v", time.Since(start))

		_, found1 = cache.Get(key1)
		_, found2 = cache.Get(key2)
		t.Logf("짧은 TTL 확인 시점: %v", time.Since(start))
		assert.False(t, found1, "짧은 TTL 아이템은 만료되어야 함 (TTL: %v, 경과 시간: %v)", shortTTL, time.Since(start))
		assert.True(t, found2, "긴 TTL 아이템은 아직 유효해야 함")

		// 긴 TTL 경과 후 조회
		sleepTime = 300 * time.Millisecond // 300ms TTL의 1배로 설정
		t.Logf("추가 대기 시작 (%v): %v", sleepTime, time.Since(start))
		time.Sleep(sleepTime)
		t.Logf("추가 대기 완료: %v", time.Since(start))

		_, found2 = cache.Get(key2)
		t.Logf("긴 TTL 확인 시점: %v", time.Since(start))
		assert.False(t, found2, "긴 TTL 아이템도 만료되어야 함 (TTL: %v, 경과 시간: %v)", longTTL, time.Since(start))
	})

	t.Run("Clear", func(t *testing.T) {
		// 캐시 초기화
		cache := New(1 * time.Second)
		defer cache.Close()

		// 여러 아이템 저장
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("key-%d", i)
			cachedResp := &CachedResponse{
				StatusCode: 200,
				Headers:    http.Header{},
				Body:       []byte(fmt.Sprintf("Item %d", i)),
			}
			cache.Set(key, cachedResp, 0)
		}

		// 모든 아이템 삭제
		cache.Clear()

		// 모든 아이템이 삭제되었는지 확인
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("key-%d", i)
			_, found := cache.Get(key)
			assert.False(t, found, "Clear 후 아이템을 찾을 수 없어야 함")
		}
	})

	t.Run("ExpiryDebug", func(t *testing.T) {
		// 짧은 TTL 설정
		cache := New(100 * time.Millisecond)
		defer cache.Close()

		// 테스트 데이터
		key := "expiry-debug"
		cachedResp := &CachedResponse{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte("Expiry Debug Test"),
		}

		// 캐시에 아이템 저장
		cache.Set(key, cachedResp, 0)

		// 저장 직후 만료 시간 확인
		item, found := cache.Get(key)
		assert.True(t, found, "캐시에 저장 후 바로 아이템을 찾을 수 있어야 함")
		t.Logf("저장 직후 만료 시간: %v (현재 시간: %v)", item.Expiry, time.Now())
		assert.True(t, time.Now().Before(item.Expiry), "만료 시간이 현재 시간보다 미래여야 함")

		// TTL 경과 후 만료 시간 확인
		time.Sleep(150 * time.Millisecond)
		item, found = cache.Get(key)
		if found {
			t.Logf("TTL 경과 후 만료 시간: %v (현재 시간: %v)", item.Expiry, time.Now())
			assert.False(t, time.Now().Before(item.Expiry), "만료 시간이 현재 시간보다 과거여야 함")
		} else {
			t.Log("TTL 경과 후 아이템이 삭제됨")
		}
	})
}
