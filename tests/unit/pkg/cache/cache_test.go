//go:build unit
// +build unit

package cache_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/pkg/cache"
)

func TestMemoryCache(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		// 캐시 초기화
		cache := cache.New(1 * time.Second)
		defer cache.Close()

		// 테스트 데이터
		key := "test-key"
		headers := make(http.Header)
		headers.Add("Content-Type", "application/json")
		cachedResp := &cache.CachedResponse{
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
		cache := cache.New(100 * time.Millisecond)
		defer cache.Close()

		// 테스트 데이터
		key := "expiry-test"
		cachedResp := &cache.CachedResponse{
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
		cache := cache.New(100 * time.Millisecond)
		defer cache.Close()

		// 테스트 데이터
		key1 := "short-ttl"
		key2 := "long-ttl"
		cachedResp := &cache.CachedResponse{
			StatusCode: 200,
			Headers:    http.Header{},
			Body:       []byte("TTL Test"),
		}

		// 다른 TTL로 저장
		cache.Set(key1, cachedResp, 50*time.Millisecond)  // 짧은 TTL
		cache.Set(key2, cachedResp, 300*time.Millisecond) // 긴 TTL

		// 짧은 TTL 경과 후 조회
		time.Sleep(100 * time.Millisecond)
		_, found1 := cache.Get(key1)
		_, found2 := cache.Get(key2)
		assert.False(t, found1, "짧은 TTL 아이템은 만료되어야 함")
		assert.True(t, found2, "긴 TTL 아이템은 아직 유효해야 함")

		// 긴 TTL 경과 후 조회
		time.Sleep(250 * time.Millisecond)
		_, found2 = cache.Get(key2)
		assert.False(t, found2, "긴 TTL 아이템도 만료되어야 함")
	})

	t.Run("Clear", func(t *testing.T) {
		// 캐시 초기화
		cache := cache.New(1 * time.Second)
		defer cache.Close()

		// 여러 아이템 저장
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("key-%d", i)
			cachedResp := &cache.CachedResponse{
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
}
