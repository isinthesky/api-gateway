package middleware

import (
	"github.com/gin-gonic/gin"
)

// CookieToHeaderMiddleware는 "token" 쿠키를 읽어서 Authorization 헤더로 설정하는 미들웨어입니다.
func CookieToHeaderMiddleware(c *gin.Context) {
	cookie, err := c.Cookie("token")
	if err == nil {
		// "token" 쿠키가 존재하면 Authorization 헤더로 설정
		authHeader := "Bearer " + cookie
		c.Request.Header.Set("Authorization", authHeader)
	}
	c.Next() // 다음 핸들러로 계속 진행
} 