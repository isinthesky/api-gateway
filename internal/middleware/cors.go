package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS는 Cross-Origin Resource Sharing 미들웨어를 설정합니다.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Access-Control-Allow-Origin 헤더 설정
		if allowAll {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			// 허용된 오리진 확인
			for _, allowed := range allowedOrigins {
				if allowed == origin {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// OPTIONS 요청 처리 (프리플라이트)
		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24시간
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 기본 CORS 헤더 설정
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, X-Request-ID")

		c.Next()
	}
}

// hasSuffix는 문자열이 지정된 접미사 중 하나로 끝나는지 확인합니다.
func hasSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// CORS2는 도메인 기반 CORS 정책을 설정하는 또 다른 방식의 미들웨어입니다.
func CORS2(allowedDomains []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			// CORS 요청이 아님
			c.Next()
			return
		}

		allowed := false
		
		// 와일드카드 검사
		if len(allowedDomains) == 1 && allowedDomains[0] == "*" {
			allowed = true
		} else {
			// 도메인 패턴 검사
			for _, domain := range allowedDomains {
				// 정확한 일치
				if domain == origin {
					allowed = true
					break
				}
				
				// 와일드카드 서브도메인 (예: *.example.com)
				if strings.HasPrefix(domain, "*.") {
					suffix := domain[1:] // "*.example.com" -> ".example.com"
					if strings.HasSuffix(origin, suffix) {
						allowed = true
						break
					}
				}
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, X-Request-ID")
		}

		// 프리플라이트 요청 처리
		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With, X-Request-ID")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24시간
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
