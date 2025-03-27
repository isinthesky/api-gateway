package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/isinthesky/api-gateway/internal/config"
)

// SizeLimitMiddleware는 요청 본문 크기를 제한하는 미들웨어입니다.
func SizeLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// POST, PUT, PATCH 메서드에만 적용
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			// Content-Length 헤더 확인
			if c.Request.ContentLength > cfg.MaxContentSize {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": fmt.Sprintf("요청 본문 크기가 허용된 최대값(%d 바이트)을 초과합니다", cfg.MaxContentSize),
				})
				return
			}

			// Content-Length가 설정되지 않은 경우도 처리 (MaxBytesReader 사용)
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxContentSize)

			// 다음 핸들러 실행
			c.Next()

			// MaxBytesReader에서 발생한 오류 확인
			if c.Errors.Last() != nil && c.Errors.Last().Err.Error() == "http: request body too large" {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": fmt.Sprintf("요청 본문 크기가 허용된 최대값(%d 바이트)을 초과합니다", cfg.MaxContentSize),
				})
				return
			}
		} else {
			// 다른 메서드는 그대로 통과
			c.Next()
		}
	}
}
