package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/isinthesky/api-gateway/pkg/ratelimiter"
)

// RateLimit은 요청 속도를 제한하는 미들웨어입니다.
func RateLimit(limiter ratelimiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 클라이언트 식별자 (IP 주소 사용)
		clientID := c.ClientIP()
		
		// IP 기반 키 생성
		key := clientID + ":" + c.FullPath()
		
		// 속도 제한 확인
		if !limiter.Allow(key) {
			// 요청 거부
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "요청 속도 제한 초과",
				"message": "잠시 후 다시 시도해주세요",
				"retry_after": 1, // 초 단위
			})
			
			// 클라이언트에게 재시도 시간 알림
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", "1") // 실제로는 설정값 사용
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", time.Now().Add(time.Second).Format(time.RFC1123))
			
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// DynamicRateLimit은 경로/클라이언트에 따라 다른 제한을 적용하는 미들웨어입니다.
func DynamicRateLimit(configs map[string]ratelimiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		clientID := c.ClientIP()
		
		// 경로에 맞는 리미터 선택
		var limiter ratelimiter.RateLimiter
		var found bool
		
		if limiter, found = configs[path]; !found {
			// 경로별 설정이 없으면 기본 리미터 사용
			limiter = configs["default"]
		}
		
		// 속도 제한 확인
		key := clientID + ":" + path
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "요청 속도 제한 초과",
				"message": "잠시 후 다시 시도해주세요",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// IPBasedRateLimit은 IP 주소 기반 속도 제한 미들웨어입니다.
func IPBasedRateLimit(limiter ratelimiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// IP 기반 제한 확인
		if !limiter.Allow(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "IP 기반 요청 속도 제한 초과",
				"message": "잠시 후 다시 시도해주세요",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// TokenBucketRateLimit은 토큰 버킷 알고리즘 기반 속도 제한 미들웨어입니다.
func TokenBucketRateLimit(ratePerSecond float64, bucketSize int) gin.HandlerFunc {
	// 토큰 버킷 초기화
	buckets := make(map[string]*Bucket)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// 버킷이 없는 경우 생성
		if _, exists := buckets[clientIP]; !exists {
			buckets[clientIP] = &Bucket{
				tokens:    float64(bucketSize),
				capacity:  float64(bucketSize),
				rate:      ratePerSecond,
				lastCheck: time.Now(),
			}
		}
		
		bucket := buckets[clientIP]
		
		// 현재 토큰 계산
		now := time.Now()
		elapsedTime := now.Sub(bucket.lastCheck).Seconds()
		bucket.tokens += elapsedTime * bucket.rate
		
		// 최대 용량 제한
		if bucket.tokens > bucket.capacity {
			bucket.tokens = bucket.capacity
		}
		
		bucket.lastCheck = now
		
		// 토큰 사용 가능 여부 확인
		if bucket.tokens < 1 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "토큰 버킷 속도 제한 초과",
				"message": "잠시 후 다시 시도해주세요",
			})
			c.Abort()
			return
		}
		
		// 토큰 사용
		bucket.tokens -= 1
		
		c.Next()
	}
}

// Bucket은 토큰 버킷 알고리즘 상태를 저장합니다.
type Bucket struct {
	tokens    float64   // 현재 토큰 수
	capacity  float64   // 최대 토큰 수
	rate      float64   // 초당 토큰 보충률
	lastCheck time.Time // 마지막 업데이트 시간
}
