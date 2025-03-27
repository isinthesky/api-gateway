package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

// ForwardRequest는 HTTP 요청을 대상 서버로 전달합니다.
func ForwardRequest(ctx context.Context, req *http.Request, targetURL string, stripPath bool, stripPrefix string) (*http.Response, error) {
	// 대상 URL 파싱
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("대상 URL 파싱 실패: %v", err)
	}

	// 원본 요청에서 새 요청 생성
	targetReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("대상 요청 생성 실패: %v", err)
	}

	// 헤더 복사
	for key, values := range req.Header {
		for _, value := range values {
			targetReq.Header.Add(key, value)
		}
	}

	// 경로 처리
	if stripPath && stripPrefix != "" {
		// 스트립 접두사 제거
		path := strings.TrimPrefix(req.URL.Path, stripPrefix)
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// 대상 URL 업데이트
		if target.Path != "" && target.Path != "/" {
			if strings.HasSuffix(target.Path, "/") {
				targetReq.URL.Path = target.Path + strings.TrimPrefix(path, "/")
			} else {
				targetReq.URL.Path = target.Path + path
			}
		} else {
			targetReq.URL.Path = path
		}
	} else {
		// 기존 경로 유지
		targetReq.URL.Path = req.URL.Path
	}

	// 쿼리 파라미터 복사
	targetReq.URL.RawQuery = req.URL.RawQuery

	// 호스트 헤더 설정
	targetReq.Host = target.Host

	// 클라이언트 IP 헤더 추가
	if clientIP := req.Header.Get("X-Forwarded-For"); clientIP != "" {
		targetReq.Header.Set("X-Forwarded-For", clientIP)
	} else if clientIP := req.Header.Get("X-Real-IP"); clientIP != "" {
		targetReq.Header.Set("X-Forwarded-For", clientIP)
	} else {
		targetReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	}

	// 프록시 정보 헤더 추가
	targetReq.Header.Set("X-Forwarded-Host", req.Host)
	targetReq.Header.Set("X-Forwarded-Proto", req.URL.Scheme)

	// 요청 전송 로깅
	log.Printf("[PROXY] 요청 전달: %s %s -> %s", targetReq.Method, req.URL.Path, targetReq.URL.String())

	// HTTP 클라이언트 생성
	client := &http.Client{}

	// 요청 전송
	resp, err := client.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("프록시 요청 실패: %v", err)
	}

	// 응답 로깅
	log.Printf("[PROXY] 응답 수신: %s %s -> 상태 코드: %d", targetReq.Method, req.URL.Path, resp.StatusCode)

	return resp, nil
}

// WebSocketProxy는 WebSocket 연결을 프록시합니다.
func WebSocketProxy(w http.ResponseWriter, r *http.Request, targetURL string, upgrader websocket.Upgrader) {
	// 대상 URL 파싱
	target, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "유효하지 않은 WebSocket 대상 URL", http.StatusInternalServerError)
		return
	}

	// 클라이언트와의 WebSocket 연결 업그레이드
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] 클라이언트 연결 업그레이드 실패: %v", err)
		return
	}
	defer clientConn.Close()

	// 헤더 구성
	requestHeader := http.Header{}
	for k, vs := range r.Header {
		// Sec-WebSocket-* 헤더는 복사하지 않음 (새 연결에서 자동 생성)
		if !strings.HasPrefix(k, "Sec-WebSocket-") && k != "Connection" && k != "Upgrade" {
			for _, v := range vs {
				requestHeader.Add(k, v)
			}
		}
	}

	// 클라이언트 IP 헤더 설정
	requestHeader.Set("X-Forwarded-For", r.RemoteAddr)
	requestHeader.Set("X-Real-IP", r.RemoteAddr)

	// 대상 서버로 WebSocket 연결
	log.Printf("[WS] 대상 서버에 연결 시도: %s", targetURL)
	serverConn, resp, err := websocket.DefaultDialer.Dial(targetURL, requestHeader)
	if err != nil {
		if resp != nil {
			log.Printf("[WS] 대상 서버 연결 실패: %d %s", resp.StatusCode, resp.Status)
		} else {
			log.Printf("[WS] 대상 서버 연결 실패: %v", err)
		}
		return
	}
	defer serverConn.Close()

	log.Printf("[WS] 연결 성공: %s -> %s", r.RemoteAddr, target.String())

	// 양방향 메시지 릴레이를 위한 채널
	clientDone := make(chan struct{})
	serverDone := make(chan struct{})

	// 클라이언트 → 서버 메시지 전달
	go func() {
		defer close(clientDone)
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WS] 클라이언트 읽기 오류: %v", err)
				}
				break
			}

			err = serverConn.WriteMessage(messageType, message)
			if err != nil {
				log.Printf("[WS] 서버 쓰기 오류: %v", err)
				break
			}
		}
	}()

	// 서버 → 클라이언트 메시지 전달
	go func() {
		defer close(serverDone)
		for {
			messageType, message, err := serverConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WS] 서버 읽기 오류: %v", err)
				}
				break
			}

			err = clientConn.WriteMessage(messageType, message)
			if err != nil {
				log.Printf("[WS] 클라이언트 쓰기 오류: %v", err)
				break
			}
		}
	}()

	// 어느 한쪽이 종료될 때까지 대기
	select {
	case <-clientDone:
		log.Printf("[WS] 클라이언트 연결 종료: %s", r.RemoteAddr)
	case <-serverDone:
		log.Printf("[WS] 서버 연결 종료: %s", target.String())
	}
}
