package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims는 JWT 토큰에 포함되는 클레임(claim) 정보를 담는 구조체입니다.
type Claims struct {
	Roles []string `json:"roles"`
	jwt.RegisteredClaims
}

// Authenticator는 인증 관련 기능을 제공하는 인터페이스입니다.
type Authenticator interface {
	GenerateToken(userID string, roles []string) (string, error)
	VerifyToken(tokenString string) (*Claims, error)
}

// JWTAuthenticator는 JWT 기반 인증을 구현하는 구조체입니다.
type JWTAuthenticator struct {
	secretKey       string
	issuer          string
	expirationDelta time.Duration
}

// New는 새로운 Authenticator를 생성합니다.
func New(secretKey, issuer string, expirationDelta time.Duration) Authenticator {
	return &JWTAuthenticator{
		secretKey:       secretKey,
		issuer:          issuer,
		expirationDelta: expirationDelta,
	}
}

// GenerateToken은 사용자 ID와 역할을 기반으로 JWT 토큰을 생성합니다.
func (a *JWTAuthenticator) GenerateToken(userID string, roles []string) (string, error) {
	if userID == "" {
		return "", errors.New("유효한 사용자 ID가 필요합니다")
	}

	now := time.Now()
	expiresAt := now.Add(a.expirationDelta)

	claims := &Claims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    a.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(a.secretKey))
	if err != nil {
		return "", fmt.Errorf("토큰 서명 실패: %v", err)
	}

	return signedToken, nil
}

// VerifyToken은 JWT 토큰의 유효성을 검증하고 클레임을 반환합니다.
func (a *JWTAuthenticator) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 서명 방식 확인
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("서명 방식이 일치하지 않습니다: %v", token.Header["alg"])
		}
		return []byte(a.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("토큰 파싱 실패: %v", err)
	}

	if !token.Valid {
		return nil, errors.New("유효하지 않은 토큰입니다")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("유효한 클레임을 추출할 수 없습니다")
	}

	// 발급자 확인
	if claims.Issuer != a.issuer {
		return nil, fmt.Errorf("발급자가 일치하지 않습니다: %s", claims.Issuer)
	}

	// 만료 시간 확인
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("만료된 토큰입니다")
	}

	return claims, nil
}
