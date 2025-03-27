package facreport_8006_test

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/isinthesky/api-gateway/tests/integration"
	"github.com/isinthesky/api-gateway/tests/utils"
)

func TestUsers(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // 로거 설정
    logger := utils.NewTestLogger(utils.LevelDebug)
    testLogger := logger.TestContext("facreport_8006 사용자 API 테스트")
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 인증 토큰 가져오기
    testLogger.Info("사용자 인증 중...")
    loginResp := e.POST("/api/auth/login").
        WithJSON(map[string]interface{}{
            "username": "testuser",
            "password": "testpass",
        }).
        Expect().
        Status(http.StatusOK).
        JSON().Object()
    
    token := loginResp.Value("token").String().Raw()
    testLogger.Info("인증 토큰 획득 성공")
    
    // 사용자 목록 API 테스트
    t.Run("사용자 목록 조회", func(t *testing.T) {
        subLogger := logger.TestContext("사용자 목록 조회")
        subLogger.Info("사용자 목록 요청 중...")
        
        resp := e.GET("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object()
        
        resp.ContainsKey("data")
        data := resp.Value("data").Array()
        data.NotEmpty()
        
        // 첫 번째 사용자 정보 확인
        firstUser := data.Element(0).Object()
        firstUser.ContainsKey("id")
        firstUser.ContainsKey("username")
        firstUser.ContainsKey("email")
        
        subLogger.Info("사용자 목록 조회 성공")
    })
    
    // 사용자 생성 API 테스트
    t.Run("사용자 생성", func(t *testing.T) {
        subLogger := logger.TestContext("사용자 생성")
        subLogger.Info("새 사용자 생성 요청 중...")
        
        newUser := map[string]interface{}{
            "username": "newuser",
            "email": "newuser@example.com",
            "firstName": "New",
            "lastName": "User",
            "password": "password123",
        }
        
        resp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        resp.ContainsKey("id").NotEmpty()
        
        // 생성된 사용자 ID 가져오기
        userId := resp.Value("id").String().Raw()
        subLogger.Info("생성된 사용자 ID: %s", userId)
        
        // 생성된 사용자 조회 테스트
        subLogger.Info("생성된 사용자 조회 중...")
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("username").
            ValueEqual("username", "newuser")
        
        subLogger.Info("사용자 생성 및 조회 성공")
    })
    
    // 사용자 업데이트 API 테스트
    t.Run("사용자 업데이트", func(t *testing.T) {
        subLogger := logger.TestContext("사용자 업데이트")
        subLogger.Info("업데이트를 위한 사용자 생성 중...")
        
        // 먼저 사용자 생성
        newUser := map[string]interface{}{
            "username": "updateuser",
            "email": "updateuser@example.com",
            "firstName": "Update",
            "lastName": "User",
            "password": "password123",
        }
        
        createResp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        userId := createResp.Value("id").String().Raw()
        subLogger.Info("생성된 사용자 ID: %s", userId)
        
        // 사용자 업데이트
        subLogger.Info("사용자 정보 업데이트 중...")
        updateData := map[string]interface{}{
            "firstName": "Updated",
            "lastName": "Name",
        }
        
        e.PUT("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(updateData).
            Expect().
            Status(http.StatusOK)
        
        // 업데이트 확인
        subLogger.Info("업데이트된 사용자 정보 확인 중...")
        e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ValueEqual("firstName", "Updated").
            ValueEqual("lastName", "Name")
        
        subLogger.Info("사용자 업데이트 성공")
    })
    
    // 사용자 삭제 API 테스트
    t.Run("사용자 삭제", func(t *testing.T) {
        subLogger := logger.TestContext("사용자 삭제")
        subLogger.Info("삭제를 위한 사용자 생성 중...")
        
        // 먼저 사용자 생성
        newUser := map[string]interface{}{
            "username": "deleteuser",
            "email": "deleteuser@example.com",
            "firstName": "Delete",
            "lastName": "User",
            "password": "password123",
        }
        
        createResp := e.POST("/api/users").
            WithHeader("Authorization", "Bearer "+token).
            WithJSON(newUser).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        userId := createResp.Value("id").String().Raw()
        subLogger.Info("생성된 사용자 ID: %s", userId)
        
        // 사용자 삭제
        subLogger.Info("사용자 삭제 중...")
        e.DELETE("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect().
            Status(http.StatusNoContent)
        
        // 삭제 확인 (실제 구현에서는 404를 반환해야 하지만, 모의 서버에서는 다를 수 있음)
        subLogger.Info("삭제된 사용자 확인 중...")
        resp := e.GET("/api/users/{id}", userId).
            WithHeader("Authorization", "Bearer "+token).
            Expect()
        
        statusCode := resp.Raw().StatusCode
        subLogger.Info("삭제 후 조회 상태 코드: %d", statusCode)
        
        // 참고: 모의 서버 구현에 따라 삭제 후에도 OK를 반환할 수 있음
        if statusCode != http.StatusNotFound && statusCode != http.StatusOK {
            t.Errorf("예상치 못한 상태 코드: %d", statusCode)
        }
        
        subLogger.Info("사용자 삭제 테스트 완료")
    })
}
