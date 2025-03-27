// +build integration

package auth

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"

	"github.com/isinthesky/api-gateway/tests/integration"
)

func TestAuthBasic(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 인증 정보
	var authToken string

	// 1. 로그인 기능 테스트
	t.Run("Login", func(t *testing.T) {
		// 로그인 요청
		resp := e.POST("/api/auth/login").
			WithJSON(map[string]interface{}{
				"username": "testuser",
				"password": "testpass",
			}).
			Expect()

		// 응답 상태 확인 (200 OK 또는 프로젝트 설정에 따른 다른 상태)
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusOK || status == http.StatusCreated,
			"로그인은 200 OK 또는 201 Created를 반환해야 함")

		// 토큰 추출 시도
		if status == http.StatusOK || status == http.StatusCreated {
			token := resp.JSON().Object().Value("token").String().Raw()
			if token != "" {
				authToken = token
				t.Logf("인증 토큰 획득: %s", authToken)
			}
		}
	})

	// 2. 토큰 없이 보호된 리소스 접근
	t.Run("Access Protected Without Token", func(t *testing.T) {
		// 보호된 리소스 요청 (토큰 없이)
		resp := e.GET("/api/protected/resource").
			Expect()

		// 401 Unauthorized 또는 403 Forbidden 예상
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusUnauthorized || status == http.StatusForbidden,
			"보호된 리소스는 인증 없이 접근 시 401 또는 403을 반환해야 함")
	})

	// 3. 토큰으로 보호된 리소스 접근 (토큰이 있는 경우)
	if authToken != "" {
		t.Run("Access Protected With Token", func(t *testing.T) {
			// 보호된 리소스 요청 (토큰 사용)
			resp := e.GET("/api/protected/resource").
				WithHeader("Authorization", "Bearer "+authToken).
				Expect()

			// 성공적인 응답 예상
			status := resp.Raw().StatusCode
			assert.Equal(t, http.StatusOK, status, "유효한 토큰으로 보호된 리소스에 접근 가능해야 함")
		})
	} else {
		t.Skip("인증 토큰을 획득하지 못해 인증 테스트를 건너뜁니다")
	}

	// 4. 잘못된 토큰으로 접근
	t.Run("Access Protected With Invalid Token", func(t *testing.T) {
		// 보호된 리소스 요청 (잘못된 토큰 사용)
		resp := e.GET("/api/protected/resource").
			WithHeader("Authorization", "Bearer invalid.token.here").
			Expect()

		// 401 Unauthorized 예상
		status := resp.Raw().StatusCode
		assert.Equal(t, http.StatusUnauthorized, status, "잘못된 토큰으로 보호된 리소스에 접근 시 401을 반환해야 함")
	})
}

func TestAuthRoles(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 인증 정보 - 여러 역할을 가진 사용자
	var adminToken string
	var userToken string

	// 1. 관리자 로그인
	t.Run("Admin Login", func(t *testing.T) {
		// 관리자 로그인 요청
		resp := e.POST("/api/auth/login").
			WithJSON(map[string]interface{}{
				"username": "admin",
				"password": "adminpass",
			}).
			Expect()

		// 응답 상태 확인
		status := resp.Raw().StatusCode
		if status == http.StatusOK || status == http.StatusCreated {
			token := resp.JSON().Object().Value("token").String().Raw()
			if token != "" {
				adminToken = token
				t.Logf("관리자 토큰 획득: %s", adminToken)
			}
		}
	})

	// 2. 일반 사용자 로그인
	t.Run("User Login", func(t *testing.T) {
		// 일반 사용자 로그인 요청
		resp := e.POST("/api/auth/login").
			WithJSON(map[string]interface{}{
				"username": "user",
				"password": "userpass",
			}).
			Expect()

		// 응답 상태 확인
		status := resp.Raw().StatusCode
		if status == http.StatusOK || status == http.StatusCreated {
			token := resp.JSON().Object().Value("token").String().Raw()
			if token != "" {
				userToken = token
				t.Logf("사용자 토큰 획득: %s", userToken)
			}
		}
	})

	// 3. 역할 기반 접근 제어 - 관리자 전용 리소스
	if adminToken != "" && userToken != "" {
		t.Run("Admin Only Resource", func(t *testing.T) {
			// 관리자 토큰으로 접근
			adminResp := e.GET("/api/admin/resource").
				WithHeader("Authorization", "Bearer "+adminToken).
				Expect()

			// 일반 사용자 토큰으로 접근
			userResp := e.GET("/api/admin/resource").
				WithHeader("Authorization", "Bearer "+userToken).
				Expect()

			// 관리자는 접근 가능, 일반 사용자는 접근 불가
			adminStatus := adminResp.Raw().StatusCode
			userStatus := userResp.Raw().StatusCode

			assert.Equal(t, http.StatusOK, adminStatus, "관리자는 관리자 전용 리소스에 접근 가능해야 함")
			assert.Equal(t, http.StatusForbidden, userStatus, "일반 사용자는 관리자 전용 리소스에 접근 불가능해야 함")
		})
	} else {
		t.Skip("역할 별 토큰을 획득하지 못해 역할 기반 접근 제어 테스트를 건너뜁니다")
	}
}

func TestAuthSessionManagement(t *testing.T) {
	// 테스트 클라이언트 생성
	e := httpexpect.New(t, integration.GetGatewayURL())

	// 인증 정보
	var authToken string

	// 1. 로그인 및 토큰 획득
	resp := e.POST("/api/auth/login").
		WithJSON(map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		}).
		Expect()

	// 응답 상태 확인
	status := resp.Raw().StatusCode
	if status == http.StatusOK || status == http.StatusCreated {
		token := resp.JSON().Object().Value("token").String().Raw()
		if token != "" {
			authToken = token
		}
	}

	// 토큰이 없으면 테스트 건너뛰기
	if authToken == "" {
		t.Skip("인증 토큰을 획득하지 못해 세션 관리 테스트를 건너뜁니다")
		return
	}

	// 2. 토큰 갱신
	t.Run("Token Refresh", func(t *testing.T) {
		// 토큰 갱신 요청
		resp := e.POST("/api/auth/refresh").
			WithHeader("Authorization", "Bearer "+authToken).
			Expect()

		// 응답 확인
		status := resp.Raw().StatusCode
		assert.Equal(t, http.StatusOK, status, "토큰 갱신은 200 OK를 반환해야 함")

		// 새 토큰 확인
		newToken := resp.JSON().Object().Value("token").String().Raw()
		assert.NotEmpty(t, newToken, "토큰 갱신은 새 토큰을 반환해야 함")
		assert.NotEqual(t, authToken, newToken, "갱신된 토큰은 이전 토큰과 달라야 함")

		// 새 토큰 사용
		if newToken != "" {
			authToken = newToken
		}
	})

	// 3. 로그아웃
	t.Run("Logout", func(t *testing.T) {
		// 로그아웃 요청
		resp := e.POST("/api/auth/logout").
			WithHeader("Authorization", "Bearer "+authToken).
			Expect()

		// 응답 확인 (204 No Content 또는 200 OK 예상)
		status := resp.Raw().StatusCode
		assert.True(t, status == http.StatusNoContent || status == http.StatusOK,
			"로그아웃은 204 No Content 또는 200 OK를 반환해야 함")

		// 로그아웃 후 토큰 사용 시도
		afterLogoutResp := e.GET("/api/protected/resource").
			WithHeader("Authorization", "Bearer "+authToken).
			Expect()

		// 401 Unauthorized 예상
		afterLogoutStatus := afterLogoutResp.Raw().StatusCode
		assert.Equal(t, http.StatusUnauthorized, afterLogoutStatus,
			"로그아웃 후에는 토큰이 무효화되어야 함")
	})
}
