package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// RouteConfig는 API 경로 구성을 정의합니다.
type RouteConfig struct {
	// 경로 패턴 (예: /users/:id)
	Path string
	// 대상 서비스 URL (예: http://user-service:8080)
	TargetURL string
	// 요청 메서드 (GET, POST 등, 비어있으면 모든 메서드)
	Methods []string
	// 이 경로에 JWT 인증이 필요한지 여부
	RequireAuth bool
	// 전달 경로 리라이트 (예: /api/users/:id -> /users/:id)
	StripPrefix string
}

// RouteManager는 경로 관리를 담당하는 구조체입니다.
type RouteManager struct {
	routes []RouteConfig
}

// NewRouteManager는 새로운 경로 관리자를 생성합니다.
func NewRouteManager() *RouteManager {
	return &RouteManager{
		routes: []RouteConfig{},
	}
}

// AddRoute는 새로운 경로 구성을 추가합니다.
func (rm *RouteManager) AddRoute(route RouteConfig) {
	rm.routes = append(rm.routes, route)
}

// RegisterRoutes는 Gin 라우터에 모든 경로를 등록합니다.
func (rm *RouteManager) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) error {
	for _, route := range rm.routes {
		targetURL, err := url.Parse(route.TargetURL)
		if err != nil {
			return fmt.Errorf("경로 '%s'의 대상 URL '%s' 파싱 오류: %v", route.Path, route.TargetURL, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		
		// 디렉터 설정: URL 리라이트 및 헤더 설정
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			
			// 헤더 및 호스트 설정
			req.Host = targetURL.Host
			
			// 경로 프리픽스 제거가 설정된 경우 적용
			if route.StripPrefix != "" && strings.HasPrefix(req.URL.Path, route.StripPrefix) {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, route.StripPrefix)
				// 경로가 비어있으면 "/" 설정
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}
		}
		
		// 핸들러 함수 생성
		handler := func(c *gin.Context) {
			proxy.ServeHTTP(c.Writer, c.Request)
		}
		
		// 경로 등록
		routePath := route.Path
		if len(route.Methods) == 0 {
			// 메서드가 지정되지 않은 경우 모든 메서드 허용
			if route.RequireAuth {
				router.Use(authMiddleware).Any(routePath, handler)
			} else {
				router.Any(routePath, handler)
			}
		} else {
			// 지정된 메서드별로 등록
			for _, method := range route.Methods {
				if route.RequireAuth {
					switch method {
					case "GET":
						router.GET(routePath, authMiddleware, handler)
					case "POST":
						router.POST(routePath, authMiddleware, handler)
					case "PUT":
						router.PUT(routePath, authMiddleware, handler)
					case "DELETE":
						router.DELETE(routePath, authMiddleware, handler)
					case "PATCH":
						router.PATCH(routePath, authMiddleware, handler)
					case "HEAD":
						router.HEAD(routePath, authMiddleware, handler)
					case "OPTIONS":
						router.OPTIONS(routePath, authMiddleware, handler)
					}
				} else {
					switch method {
					case "GET":
						router.GET(routePath, handler)
					case "POST":
						router.POST(routePath, handler)
					case "PUT":
						router.PUT(routePath, handler)
					case "DELETE":
						router.DELETE(routePath, handler)
					case "PATCH":
						router.PATCH(routePath, handler)
					case "HEAD":
						router.HEAD(routePath, handler)
					case "OPTIONS":
						router.OPTIONS(routePath, handler)
					}
				}
			}
		}
	}
	
	return nil
}
