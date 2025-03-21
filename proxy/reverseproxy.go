package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTTPProxy는 HTTP 요청을 프록시하는 구조체입니다.
type HTTPProxy struct {
	backendURL *url.URL
}

// NewHTTPProxy는 새로운 HTTP 프록시를 생성합니다.
func NewHTTPProxy(backendBaseURL string) *HTTPProxy {
	url, err := url.Parse(backendBaseURL)
	if err != nil {
		return nil
	}
	
	return &HTTPProxy{
		backendURL: url,
	}
}

// HTTPProxyHandler는 HTTP 요청을 프록시하는 Gin 핸들러 함수를 반환합니다.
func HTTPProxyHandler(proxy *HTTPProxy, targetPath string, stripPath bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 대상 URL 구성
		targetURL, err := url.Parse(targetPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "대상 URL 파싱 오류"})
			return
		}
		
		// 리버스 프록시 생성
		reverseProxy := httputil.NewSingleHostReverseProxy(proxy.backendURL)
		
		// 프록시 디렉터 설정
		originalDirector := reverseProxy.Director
		reverseProxy.Director = func(req *http.Request) {
			originalDirector(req)
			
			// 기본 설정
			req.URL.Scheme = proxy.backendURL.Scheme
			req.URL.Host = proxy.backendURL.Host
			
			// 경로 설정
			if stripPath {
				// 경로 접두사를 제거하고 대상 경로로 대체
				p := strings.TrimPrefix(req.URL.Path, c.FullPath())
				if targetURL.Path == "" {
					req.URL.Path = "/"
				} else {
					req.URL.Path = targetURL.Path
				}
				if p != "" && p != "/" {
					req.URL.Path = req.URL.Path + p
				}
			} else {
				// 대상 경로 뒤에 현재 경로 추가
				req.URL.Path = targetURL.Path + req.URL.Path
			}
			
			// 기존 호스트 헤더 유지
			if _, ok := req.Header["Host"]; !ok {
				req.Header["Host"] = []string{proxy.backendURL.Host}
			}
			
			// 원본 IP 헤더 추가
			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Real-IP", c.ClientIP())
		}
		
		// 프록시 응답 처리자 설정
		reverseProxy.ModifyResponse = func(resp *http.Response) error {
			// 필요한 경우 응답 수정 로직 추가
			return nil
		}
		
		// 오류 처리자 설정
		reverseProxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "백엔드 서비스 오류",
			})
		}
		
		// 프록시 요청 실행
		reverseProxy.ServeHTTP(c.Writer, c.Request)
	}
}