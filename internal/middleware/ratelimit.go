package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter는 요청 제한기능을 구현한 구조체입니다.
type RateLimiter struct {
	mutex      sync.Mutex
	window     time.Duration
	limit      int
	clients    map[string][]time.Time
	cleanerQuit chan struct{}
}

// NewRateLimiter는 새로운 RateLimiter를 생성합니다.
func NewRateLimiter(window time.Duration, limit int) *RateLimiter {
	return &RateLimiter{
		window:     window,
		limit:      limit,
		clients:    make(map[string][]time.Time),
		cleanerQuit: make(chan struct{}),
	}
}

// StartCleaner는 오래된 데이터를 정리하는 고루틴을 시작합니다.
func (rl *RateLimiter) StartCleaner(cleanInterval time.Duration) {
	ticker := time.NewTicker(cleanInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				rl.cleanExpired()
			case <-rl.cleanerQuit:
				ticker.Stop()
				return
			}
		}
	}()
}

// StopCleaner는 클리너 고루틴을 중지합니다.
func (rl *RateLimiter) StopCleaner() {
	close(rl.cleanerQuit)
}

// cleanExpired는 만료된 요청 기록을 제거합니다.
func (rl *RateLimiter) cleanExpired() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for ip, timestamps := range rl.clients {
		var validTimestamps []time.Time
		for _, ts := range timestamps {
			if now.Sub(ts) < rl.window {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		
		if len(validTimestamps) > 0 {
			rl.clients[ip] = validTimestamps
		} else {
			delete(rl.clients, ip)
		}
	}
}

// Allow는 주어진 클라이언트의 요청을 허용할지 결정합니다.
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	
	// 현재 윈도우 내의 유효한 요청만 유지
	var validTimestamps []time.Time
	for _, ts := range rl.clients[clientIP] {
		if now.Sub(ts) < rl.window {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// 요청 제한 확인
	if len(validTimestamps) >= rl.limit {
		rl.clients[clientIP] = validTimestamps
		return false
	}

	// 현재 요청 추가
	rl.clients[clientIP] = append(validTimestamps, now)
	return true
}

// RateLimitMiddleware는 요청 속도를 제한하는 미들웨어입니다.
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// 요청 제한 확인
		if !rl.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "요청 제한 초과",
				"message": "잠시 후 다시 시도해주세요",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
