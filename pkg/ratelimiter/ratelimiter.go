package ratelimiter

import (
	"errors"
	"sync"
	"time"
)

// 오류 정의
var (
	ErrRateLimited = errors.New("rate limit exceeded")
)

// RateLimiter는 요청 속도 제한 인터페이스입니다.
type RateLimiter interface {
	Allow(key string) bool
	AllowN(key string, n int) bool
	Peek(key string) (int, bool)
	Reset(key string)
	Stop()
}

// TokenBucket은 토큰 버킷 알고리즘 기반 속도 제한기입니다.
type TokenBucket struct {
	tokens     map[string]*bucket
	mu         sync.RWMutex
	rate       float64      // 초당 토큰 보충 속도
	capacity   int          // 버킷 최대 용량
	window     time.Duration // 속도 측정 시간 단위
	quit       chan struct{}
}

// bucket은 토큰 버킷 상태를 나타냅니다.
type bucket struct {
	tokens    float64     // 현재 토큰 수
	lastCheck time.Time   // 마지막 확인 시간
}

// New는 새로운 속도 제한기를 생성합니다.
func New(window time.Duration, maxRequests int) *TokenBucket {
	// 시간당 토큰 보충 속도 계산
	rate := float64(maxRequests) / window.Seconds()

	rl := &TokenBucket{
		tokens:   make(map[string]*bucket),
		rate:     rate,
		capacity: maxRequests,
		window:   window,
		quit:     make(chan struct{}),
	}

	// 청소 고루틴 시작
	go rl.startCleaner()

	return rl
}

// Allow는 주어진 키에 대한 요청을 허용할지 결정합니다 (1개의 토큰 사용).
func (rl *TokenBucket) Allow(key string) bool {
	return rl.AllowN(key, 1)
}

// AllowN은 주어진 키에 대해 n개의 토큰을 사용할 수 있는지 확인합니다.
func (rl *TokenBucket) AllowN(key string, n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	
	// 버킷이 존재하지 않으면 생성
	b, exists := rl.tokens[key]
	if !exists {
		b = &bucket{
			tokens:    float64(rl.capacity),
			lastCheck: now,
		}
		rl.tokens[key] = b
	}

	// 토큰 보충 계산
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens = min(float64(rl.capacity), b.tokens+(elapsed*rl.rate))
	b.lastCheck = now

	// 요청에 충분한 토큰이 있는지 확인
	if b.tokens < float64(n) {
		return false
	}

	// 토큰 사용
	b.tokens -= float64(n)
	return true
}

// Peek는 키에 대한 현재 토큰 상태를 반환합니다 (토큰 사용 없음).
func (rl *TokenBucket) Peek(key string) (int, bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	b, exists := rl.tokens[key]
	if !exists {
		return rl.capacity, true
	}

	// 현재 사용 가능한 토큰 수 계산
	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	tokens := min(float64(rl.capacity), b.tokens+(elapsed*rl.rate))

	return int(tokens), tokens >= 1.0
}

// Reset은 키에 대한 버킷을 초기화합니다.
func (rl *TokenBucket) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.tokens, key)
}

// Stop은 청소 고루틴을 중지합니다.
func (rl *TokenBucket) Stop() {
	close(rl.quit)
}

// startCleaner는 오래된 버킷을 제거하는 백그라운드 작업을 시작합니다.
func (rl *TokenBucket) startCleaner() {
	ticker := time.NewTicker(rl.window * 2) // 윈도우의 2배 주기로 정리
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanExpired()
		case <-rl.quit:
			return
		}
	}
}

// cleanExpired는 토큰이 최대 용량인 버킷을 제거합니다.
func (rl *TokenBucket) cleanExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, b := range rl.tokens {
		// 마지막 확인 후 경과 시간
		elapsed := now.Sub(b.lastCheck).Seconds()
		
		// 현재 토큰 수 계산
		currentTokens := min(float64(rl.capacity), b.tokens+(elapsed*rl.rate))
		
		// 토큰이 최대 용량에 도달한 버킷은 삭제 (일정 시간 동안 사용되지 않음)
		if currentTokens >= float64(rl.capacity) && now.Sub(b.lastCheck) > rl.window*2 {
			delete(rl.tokens, key)
		}
	}
}

// min은 두 float64 값 중 작은 값을 반환합니다.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// SlidingWindow는 슬라이딩 윈도우 알고리즘 기반 속도 제한기입니다.
type SlidingWindow struct {
	clients    map[string][]time.Time
	mu         sync.RWMutex
	window     time.Duration
	maxRequests int
	quit       chan struct{}
}

// NewSlidingWindow는 새로운 슬라이딩 윈도우 속도 제한기를 생성합니다.
func NewSlidingWindow(window time.Duration, maxRequests int) *SlidingWindow {
	rl := &SlidingWindow{
		clients:     make(map[string][]time.Time),
		window:      window,
		maxRequests: maxRequests,
		quit:        make(chan struct{}),
	}

	// 청소 고루틴 시작
	go rl.startCleaner()

	return rl
}

// Allow는 주어진 키에 대한 요청을 허용할지 결정합니다.
func (rl *SlidingWindow) Allow(key string) bool {
	return rl.AllowN(key, 1)
}

// AllowN은 주어진 키에 대해 n개의 요청을 허용할지 결정합니다.
func (rl *SlidingWindow) AllowN(key string, n int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)
	
	// 윈도우 내의 유효한 타임스탬프만 유지
	var validTimes []time.Time
	for _, ts := range rl.clients[key] {
		if ts.After(windowStart) {
			validTimes = append(validTimes, ts)
		}
	}

	// 요청 허용 여부 확인
	if len(validTimes) + n > rl.maxRequests {
		rl.clients[key] = validTimes
		return false
	}

	// 새로운 타임스탬프 추가
	for i := 0; i < n; i++ {
		validTimes = append(validTimes, now)
	}
	rl.clients[key] = validTimes
	
	return true
}

// Peek는 키에 대한 현재 요청 수를 반환합니다.
func (rl *SlidingWindow) Peek(key string) (int, bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)
	
	// 윈도우 내의 유효한 요청 수 계산
	var count int
	for _, ts := range rl.clients[key] {
		if ts.After(windowStart) {
			count++
		}
	}

	// 요청 가능 여부 확인
	return count, count < rl.maxRequests
}

// Reset은 키에 대한 요청 기록을 초기화합니다.
func (rl *SlidingWindow) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.clients, key)
}

// Stop은 청소 고루틴을 중지합니다.
func (rl *SlidingWindow) Stop() {
	close(rl.quit)
}

// startCleaner는 오래된 요청 기록을 제거하는 백그라운드 작업을 시작합니다.
func (rl *SlidingWindow) startCleaner() {
	ticker := time.NewTicker(rl.window / 2) // 윈도우의 절반 주기로 정리
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanExpired()
		case <-rl.quit:
			return
		}
	}
}

// cleanExpired는 만료된 요청 기록을 제거합니다.
func (rl *SlidingWindow) cleanExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)
	
	for key, times := range rl.clients {
		var validTimes []time.Time
		for _, ts := range times {
			if ts.After(windowStart) {
				validTimes = append(validTimes, ts)
			}
		}
		
		if len(validTimes) > 0 {
			rl.clients[key] = validTimes
		} else {
			delete(rl.clients, key)
		}
	}
}
