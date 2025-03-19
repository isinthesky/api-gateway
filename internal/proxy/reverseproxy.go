package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// NewReverseProxy는 지정된 대상 URL로 요청을 전달하는 리버스 프록시를 생성합니다.
func NewReverseProxy(targetURL string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	
	proxy := httputil.NewSingleHostReverseProxy(url)
	
	// 추가 설정이 필요하면 여기에 구현
	// 예: 프록시 에러 핸들러, 요청/응답 수정자 등
	
	return proxy, nil
}

// SetupProxyDirector는 프록시 디렉터를 설정하여 요청을 커스터마이징합니다.
func SetupProxyDirector(proxy *httputil.ReverseProxy, targetURL *url.URL) {
	originalDirector := proxy.Director
	
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// 필요한 추가 헤더나 처리를 여기에 구현
		req.Host = targetURL.Host
	}
} 