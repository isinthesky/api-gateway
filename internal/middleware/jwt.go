package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig는 JWT 인증에 필요한 설정을 담는 구조체입니다.
type JWTConfig struct {
	SecretKey       string
	Issuer          string
	ExpirationDelta time.Duration
}

// Claims는 JWT 토큰에 포함되는 클레임(claim) 정보를 담는 구조체입니다.
type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTAuthMiddleware는 JWT 토큰을 검증하는 미들웨어 함수를 반환합니다.
func JWTAuthMiddleware(config JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Authorization 헤더에서 JWT 추출
		tokenString, err := extractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "인증 토큰이 필요합니다"})
			c.Abort()
			return
		}

		// JWT 토큰 파싱 및 검증
		token, err := validateToken(tokenString, config.SecretKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "유효하지 않은 토큰입니다", "details": err.Error()})
			c.Abort()
			return
		}

		// 검증된 토큰의 클레임 가져오기
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "유효하지 않은 토큰입니다"})
			c.Abort()
			return
		}

		// 토큰 만료 확인
		if time.Now().Unix() > claims.ExpiresAt.Unix() {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "만료된 토큰입니다"})
			c.Abort()
			return
		}

		// 클레임 정보를 컨텍스트에 저장
		c.Set("userId", claims.Subject)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// extractToken은 HTTP 요청에서 JWT 토큰을 추출합니다.
func extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	
	if authHeader == "" {
		return "", errors.New("Authorization 헤더가 없습니다")
	}
	
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("Authorization 헤더 형식이 잘못되었습니다")
	}
	
	return parts[1], nil
}

// validateToken은 JWT 토큰의 유효성을 검증합니다.
func validateToken(tokenString, secretKey string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// HMAC 알고리즘 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("서명 방식이 일치하지 않습니다")
		}
		return []byte(secretKey), nil
	})
	
	return token, err
}
