package proxy

import (
	"log"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketProxy는 WebSocket 연결을 프록시하는 구조체입니다.
type WebSocketProxy struct {
	backendURL *url.URL
	upgrader   websocket.Upgrader
}

// NewWebSocketProxy는 새로운 WebSocket 프록시를 생성합니다.
func NewWebSocketProxy(backendBaseURL string, upgrader websocket.Upgrader) *WebSocketProxy {
	url, err := url.Parse(backendBaseURL)
	if err != nil {
		log.Printf("백엔드 URL 파싱 오류: %v", err)
		return nil
	}

	return &WebSocketProxy{
		backendURL: url,
		upgrader:   upgrader,
	}
}

// WebSocketProxyHandler는 WebSocket 연결을 프록시하는 Gin 핸들러 함수를 반환합니다.
func WebSocketProxyHandler(proxy *WebSocketProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 클라이언트와의 WebSocket 연결 업그레이드
		clientConn, err := proxy.upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("클라이언트 WebSocket 업그레이드 실패: %v", err)
			return
		}
		defer clientConn.Close()

		// 백엔드 URL 구성
		backendURL := *proxy.backendURL
		// 프록시 경로 파싱
		path := c.Param("proxyPath")
		// 접두사 제거 (선택 사항)
		backendURL.Path = path

		// 프로토콜 변경 (HTTP -> WS, HTTPS -> WSS)
		if backendURL.Scheme == "http" {
			backendURL.Scheme = "ws"
		} else if backendURL.Scheme == "https" {
			backendURL.Scheme = "wss"
		}

		// 백엔드 서버에 연결
		backendConn, _, err := websocket.DefaultDialer.Dial(backendURL.String(), nil)
		if err != nil {
			log.Printf("백엔드 WebSocket 연결 실패: %v", err)
			return
		}
		defer backendConn.Close()

		log.Printf("WebSocket 프록시 연결 성공: %s -> %s", c.Request.RemoteAddr, backendURL.String())

		// 양방향 메시지 릴레이 채널 설정
		clientToBackend := make(chan []byte)
		backendToClient := make(chan []byte)
		errorChan := make(chan error)

		// 클라이언트 -> 백엔드 메시지 릴레이
		go relay(clientConn, backendConn, clientToBackend, errorChan)
		// 백엔드 -> 클라이언트 메시지 릴레이
		go relay(backendConn, clientConn, backendToClient, errorChan)

		// 에러 또는 연결 종료 대기
		<-errorChan
		log.Printf("WebSocket 연결 종료: %s", c.Request.RemoteAddr)
	}
}

// relay는 한 WebSocket 연결에서 다른 연결로 메시지를 릴레이합니다.
func relay(src, dst *websocket.Conn, messageChan chan []byte, errorChan chan error) {
	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			log.Printf("WebSocket 읽기 오류: %v", err)
			errorChan <- err
			return
		}

		// 메시지를 다른 연결로 전송
		err = dst.WriteMessage(messageType, message)
		if err != nil {
			log.Printf("WebSocket 쓰기 오류: %v", err)
			errorChan <- err
			return
		}

		messageChan <- message
	}
} 