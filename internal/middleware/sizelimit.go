package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/isinthesky/api-gateway/config"
)

// ErrRequestEntityTooLarge은 요청 크기가 너무 큰 경우 반환됩니다.
var ErrRequestEntityTooLarge = errors.New("요청 엔티티가 너무 큽니다")

// SizeLimitMiddleware는 요청 본문 크기를 제한하는 미들웨어입니다.
func SizeLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxContentSize)
		c.Next()
	}
} 