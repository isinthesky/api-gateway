package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CookieToHeader는 쿠키를 헤더로 변환하는 미들웨어입니다.
func CookieToHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization 헤더가 없는 경우에만 쿠키 확인
		if c.GetHeader("Authorization") == "" {
			// 인증 토큰 쿠키 확인
			tokenCookie, err := c.Cookie("access_token")
			if err == nil && tokenCookie != "" {
				// 쿠키 값을 Authorization 헤더로 추가
				c.Request.Header.Set("Authorization", "Bearer "+tokenCookie)
			}
		}
		c.Next()
	}
}

// Timeout은 요청 타임아웃을 설정하는 미들웨어입니다.
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 타임아웃 채널 생성
		doneCh := make(chan struct{})
		
		// 타이머 시작
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		
		// 백그라운드 고루틴에서 핸들러 실행
		go func() {
			c.Next()
			close(doneCh) // 핸들러 완료 시 채널 닫기
		}()
		
		// 타임아웃 또는 완료 대기
		select {
		case <-timer.C: // 타임아웃 발생
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": "요청 처리 시간이 초과되었습니다",
			})
			return
		case <-doneCh: // 정상 완료
			return
		}
	}
}

// SecureHeaders는 보안 관련 HTTP 헤더를 추가하는 미들웨어입니다.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 보안 관련 헤더 설정
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// HTTP/2 서버 푸시가 활성화된 경우 CSRF 토큰 미리 로드
		// c.Header("Link", "</assets/csrf-token.js>; rel=preload; as=script")
		
		c.Next()
	}
}

// RequestID는 요청 ID를 생성하고 설정하는 미들웨어입니다.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 이미 헤더에 요청 ID가 있는지 확인
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 새 요청 ID 생성 (UUID 권장)
			requestID = GenerateUUID()
			c.Request.Header.Set("X-Request-ID", requestID)
		}
		
		// 응답 헤더에도 요청 ID 설정
		c.Header("X-Request-ID", requestID)
		
		// 로깅 및 추적을 위해 컨텍스트에 요청 ID 저장
		c.Set("RequestID", requestID)
		
		c.Next()
	}
}

// RecoveryWithJSON은 패닉 복구 및 JSON 응답을 제공하는 미들웨어입니다.
func RecoveryWithJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 패닉 상세 정보 로깅
				// logger.Error("Panic recovered", zap.Any("error", err), zap.String("request_id", c.GetString("RequestID")))
				
				// JSON 오류 응답
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":  "서버 내부 오류가 발생했습니다",
					"detail": "관리자에게 문의하세요",
					"code":   "INTERNAL_SERVER_ERROR",
				})
			}
		}()
		
		c.Next()
	}
}

// GenerateUUID는 임의의 UUID를 생성합니다 (간단한 구현).
func GenerateUUID() string {
	// 실제 구현에서는 github.com/google/uuid 같은 라이브러리 사용 권장
	now := time.Now().UnixNano()
	return strconv.FormatInt(now, 10)
}
