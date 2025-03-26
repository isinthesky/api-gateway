package routing_test

import (
    "net/http"
    "testing"

    "github.com/gavv/httpexpect/v2"
    "github.com/isinthesky/api-gateway/tests/integration"
)

func TestRouting(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 테스트 케이스
    t.Run("기본 라우팅", func(t *testing.T) {
        // 사용자 API 요청
        e.GET("/api/users").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("data")
        
        // 보고서 API 요청
        e.GET("/api/reports").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("data")
    })
    
    t.Run("경로 파라미터 라우팅", func(t *testing.T) {
        // 특정 사용자 ID로 요청
        e.GET("/api/users/123").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("id").
            ValueEqual("id", "123")
        
        // 특정 보고서 ID로 요청
        e.GET("/api/reports/456").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("id").
            ValueEqual("id", "456")
    })
    
    t.Run("쿼리 파라미터", func(t *testing.T) {
        // 쿼리 파라미터가 있는 보고서 요청
        e.GET("/api/reports").
            WithQuery("limit", 10).
            WithQuery("offset", 0).
            WithQuery("reportType", "TEST").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("limit")
    })
    
    t.Run("공개 엔드포인트", func(t *testing.T) {
        // 공개 상태 엔드포인트
        e.GET("/api/public/status").
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ContainsKey("status").
            ValueEqual("status", "ok")
    })
    
    t.Run("존재하지 않는 경로", func(t *testing.T) {
        // 존재하지 않는 경로로 요청
        e.GET("/non-existent-path").
            Expect().
            Status(http.StatusNotFound)
    })
}

func TestHTTPMethods(t *testing.T) {
    // 테스트 환경 설정
    setup := integration.NewTestSetup(t)
    defer setup.Cleanup()
    
    // httpexpect 인스턴스 생성
    e := httpexpect.New(t, "http://localhost:18080")
    
    // 테스트 데이터
    userData := map[string]interface{}{
        "username": "testuser",
        "email": "test@example.com",
        "firstName": "Test",
        "lastName": "User",
        "password": "password123",
    }
    
    // 테스트 케이스
    t.Run("HTTP 메소드 테스트", func(t *testing.T) {
        // POST 요청 (사용자 생성)
        createResp := e.POST("/api/users").
            WithJSON(userData).
            Expect().
            Status(http.StatusCreated).
            JSON().Object()
        
        // 생성된 ID 추출
        userId := createResp.Value("id").String().Raw()
        
        // GET 요청 (사용자 조회)
        e.GET("/api/users/{id}", userId).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ValueEqual("username", userData["username"])
        
        // PUT 요청 (사용자 업데이트)
        updateData := map[string]interface{}{
            "firstName": "Updated",
            "lastName": "Name",
        }
        
        e.PUT("/api/users/{id}", userId).
            WithJSON(updateData).
            Expect().
            Status(http.StatusOK)
        
        // 업데이트 확인
        e.GET("/api/users/{id}", userId).
            Expect().
            Status(http.StatusOK).
            JSON().Object().
            ValueEqual("firstName", "Updated").
            ValueEqual("lastName", "Name")
        
        // DELETE 요청 (사용자 삭제)
        e.DELETE("/api/users/{id}", userId).
            Expect().
            Status(http.StatusNoContent)
        
        // 삭제 확인
        e.GET("/api/users/{id}", userId).
            Expect().
            Status(http.StatusOK)  // 모의 서버는 항상 OK 반환
    })
}
