package proxy_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "github.com/isinthesky/api-gateway/proxy"
)

func TestWebSocketProxyHandler(t *testing.T) {
    // 테스트 설정
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // 웹소켓 업그레이더 설정
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }
    
    // 모의 웹소켓 서버
    wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 웹소켓 연결 업그레이드
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            t.Fatalf("웹소켓 업그레이드 실패: %v", err)
            return
        }
        defer conn.Close()
        
        // 에코 서버 구현: 수신한 메시지를 그대로 돌려보냄
        for {
            messageType, message, err := conn.ReadMessage()
            if err != nil {
                // 연결 종료 시 정상 종료
                if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
                    return
                }
                t.Logf("메시지 읽기 오류: %v", err)
                return
            }
            
            // 수신한 메시지에 "에코: " 접두사 추가하여 응답
            responseMessage := "에코: " + string(message)
            if err := conn.WriteMessage(messageType, []byte(responseMessage)); err != nil {
                t.Logf("메시지 쓰기 오류: %v", err)
                return
            }
        }
    }))
    defer wsServer.Close()
    
    // WebSocket URL 변환 (HTTP → WebSocket)
    wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
    
    // 웹소켓 프록시 설정
    wsProxy := proxy.NewWebSocketProxy(wsURL, upgrader)
    
    // 테스트 라우트 구성
    router.GET("/websocket/*proxyPath", proxy.WebSocketProxyHandler(wsProxy))
    
    // WebSocket 클라이언트는 실제 연결이 필요하므로 직접 테스트하기는 어려움
    // 여기서는 핸들러가 등록되었는지만 확인
    
    // 엔드포인트 설정 확인
    routes := router.Routes()
    found := false
    for _, route := range routes {
        if route.Path == "/websocket/*proxyPath" && route.Method == "GET" {
            found = true
            break
        }
    }
    
    assert.True(t, found, "WebSocket 프록시 핸들러가 등록되어야 함")
    
    // 참고: 실제 WebSocket 연결 테스트는 통합 테스트에서 수행하는 것이 좋음
}
