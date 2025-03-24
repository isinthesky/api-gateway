package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/isinthesky/api-gateway/internal/proxy"
)

// Route는 단일 라우트 설정을 정의합니다.
type Route struct {
	Path         string   `json:"path"`         // API 경로 패턴
	TargetURL    string   `json:"targetURL"`    // 전달할 대상 서비스 URL
	Methods      []string `json:"methods"`      // 허용할 HTTP 메서드
	RequireAuth  bool     `json:"requireAuth"`  // 인증 필요 여부
	StripPrefix  string   `json:"stripPrefix"`  // 요청 경로에서 제거할 접두사
}

// RoutesConfig는 여러 라우트 설정을 담는 구조체입니다.
type RoutesConfig struct {
	Routes []Route `json:"routes"`
}

// RouteManager는 라우트를 관리하는 구조체입니다.
type RouteManager struct {
	routes []Route
}

// LoadRoutes는 지정된 파일 경로에서 라우트 설정을 로드합니다.
func LoadRoutes(filePath string) (*RouteManager, error) {
	// 파일 경로가 지정되지 않았다면 기본 경로 사용
	if filePath == "" {
		filePath = "config/routes.json"
	}

	// 파일 읽기
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("라우트 설정 파일을 열 수 없습니다: %w", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("라우트 설정 파일을 읽을 수 없습니다: %w", err)
	}

	// JSON 파싱
	var config RoutesConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, fmt.Errorf("라우트 설정 파일 파싱 실패: %w", err)
	}

	return &RouteManager{routes: config.Routes}, nil
}

// RegisterRoutes는 로드된 라우트를 Gin 라우터에 등록합니다.
func (rm *RouteManager) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) error {
	for _, route := range rm.routes {
		// 대상 URL 검증
		targetURL := route.TargetURL
		if targetURL == "" {
			return fmt.Errorf("라우트 '%s'에 대한 대상 URL이 지정되지 않았습니다", route.Path)
		}

		// 프록시 생성
		reverseProxy, err := proxy.NewReverseProxy(targetURL)
		if err != nil {
			return fmt.Errorf("라우트 '%s'에 대한 프록시 생성 실패: %w", route.Path, err)
		}

		// 각 HTTP 메서드에 대한 핸들러 등록
		for _, method := range route.Methods {
			handler := func(c *gin.Context) {
				// 경로 접두사 제거
				if route.StripPrefix != "" && strings.HasPrefix(c.Request.URL.Path, route.StripPrefix) {
					c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, route.StripPrefix)
					if c.Request.URL.Path == "" {
						c.Request.URL.Path = "/"
					}
				}
				
				// 요청 전달
				reverseProxy.ServeHTTP(c.Writer, c.Request)
			}

			// 경로 및 HTTP 메서드에 따라 핸들러 등록
			// 인증이 필요한 경우 인증 미들웨어 추가
			if route.RequireAuth {
				router.Handle(method, route.Path, authMiddleware, handler)
			} else {
				router.Handle(method, route.Path, handler)
			}
		}
	}

	return nil
}
