package cache

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// CachedResponse는 캐시된 HTTP 응답을 나타냅니다.
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Expiry     time.Time
}

// CacheProvider는 캐싱 기능을 제공하는 인터페이스입니다.
type CacheProvider interface {
	Get(key string) (*CachedResponse, bool)
	Set(key string, response *CachedResponse, ttl time.Duration)
	Delete(key string)
	Clear()
	Close()
}

// MemoryCache는 메모리 기반 캐시 구현체입니다.
type MemoryCache struct {
	items      map[string]*CachedResponse
	mu         sync.RWMutex
	defaultTTL time.Duration
	quit       chan struct{}
}

// New는 새로운 메모리 캐시를 생성합니다.
func New(defaultTTL time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items:      make(map[string]*CachedResponse),
		defaultTTL: defaultTTL,
		quit:       make(chan struct{}),
	}

	// 캐시 만료 클리너 시작
	go cache.startCleaner()

	return cache
}

// Get은 캐시에서 키에 해당하는 응답을 가져옵니다.
func (c *MemoryCache) Get(key string) (*CachedResponse, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return nil, false
	}

	now := time.Now()
	// 만료 시간과 현재 시간의 차이가 2ms 이상일 때만 만료로 처리
	if now.Sub(item.Expiry) > 2*time.Millisecond {
		// 디버깅을 위한 로그 추가
		log.Printf("캐시 만료: key=%s, 만료시간=%v, 현재시간=%v, 차이=%v", key, item.Expiry, now, now.Sub(item.Expiry))
		c.Delete(key)
		return nil, false
	}

	return item, true
}

// Set은 응답을 캐시에 저장합니다.
func (c *MemoryCache) Set(key string, response *CachedResponse, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 응답 객체 복사
	newResponse := &CachedResponse{
		StatusCode: response.StatusCode,
		Headers:    response.Headers.Clone(), // HTTP 헤더 복사
		Body:       make([]byte, len(response.Body)),
	}
	copy(newResponse.Body, response.Body)

	// TTL이 지정되지 않은 경우 기본값 사용
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	// 만료 시간 설정
	now := time.Now()
	newResponse.Expiry = now.Add(ttl)
	
	// 디버깅을 위한 로그 추가
	log.Printf("캐시 저장: key=%s, TTL=%v, 만료시간=%v, 현재시간=%v", key, ttl, newResponse.Expiry, now)

	// 캐시에 저장
	c.items[key] = newResponse
}

// Delete는 캐시에서 키에 해당하는 항목을 삭제합니다.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear는 캐시의 모든 항목을 삭제합니다.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CachedResponse)
}

// Close는 캐시 리소스를 정리합니다.
func (c *MemoryCache) Close() {
	close(c.quit)
}

// startCleaner는 만료된 캐시 항목을 정기적으로 제거하는 백그라운드 작업을 시작합니다.
func (c *MemoryCache) startCleaner() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanExpired()
		case <-c.quit:
			return
		}
	}
}

// cleanExpired는 만료된 캐시 항목을 제거합니다.
func (c *MemoryCache) cleanExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if now.After(item.Expiry) {
			delete(c.items, key)
		}
	}
}
