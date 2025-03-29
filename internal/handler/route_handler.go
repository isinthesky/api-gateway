package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/isinthesky/api-gateway/internal/auth"
	"github.com/isinthesky/api-gateway/internal/config"
	"github.com/isinthesky/api-gateway/internal/proxy"
	"github.com/isinthesky/api-gateway/pkg/cache"
	"github.com/isinthesky/api-gateway/pkg/circuitbreaker"
	"github.com/isinthesky/api-gateway/pkg/loadbalancer"
)

// RouteHandler는 API 라우트를 처리하는 핸들러입니다.
type RouteHandler struct {
	loadBalancer    loadbalancer.LoadBalancer
	circuitBreaker  *circuitbreaker.CircuitBreaker
	cache           cache.CacheProvider
	config          *config.Config
	wsUpgrader      websocket.Upgrader
	authenticator   auth.Authenticator
}

// NewRouteHandler는 새로운 RouteHandler를 생성합니다.
func NewRouteHandler(
	lb loadbalancer.LoadBalancer,
	cb *circuitbreaker.CircuitBreaker,
	cacheProvider cache.CacheProvider,
	cfg *config.Config,
) *RouteHandler {
	// WebSocket 업그레이더 설정
	wsUpgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, allowed := range cfg.AllowedOrigins {
				if allowed == origin {
					return true
				}
			}
			return false
		},
	}

	// 인증 처리기 설정
	authenticator := auth.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTExpirationDelta)

	return &RouteHandler{
		loadBalancer:    lb,
		circuitBreaker:  cb,
		cache:           cacheProvider,
		config:          cfg,
		wsUpgrader:      wsUpgrader,
		authenticator:   authenticator,
	}
}

// RegisterRoutes는 라우터에 모든 라우트를 등록합니다.
func (h *RouteHandler) RegisterRoutes(router *gin.Engine) error {
	// 헬스 체크 엔드포인트
	router.GET("/health", h.HealthCheckHandler)


	// 라우트 설정 로드
	routes, err := h.config.LoadRoutes()
	if err != nil {
		return err
	}

	// 라우트 그룹화
	var rootRoutes []config.Route        // 루트 경로 라우트 ("/")
	var apiRoutes []config.Route         // API 관련 라우트 ("/api/*")
	var specificRoutes []config.Route    // 특정 경로 라우트 (예: "/login")
	var rootCatchAllRoute *config.Route  // 루트 캐치올 라우트 ("/*proxyPath")
	var wsRoutes []config.Route          // WebSocket 라우트

	// 라우트 분류
	for _, route := range routes {
		// WebSocket 라우트
		if strings.HasPrefix(route.Path, "/ws") || strings.HasPrefix(route.Path, "/websocket") {
			wsRoutes = append(wsRoutes, route)
			continue
		}

		// 일반 HTTP 라우트
		if route.Path == "/" {
			rootRoutes = append(rootRoutes, route)
		} else if strings.HasPrefix(route.Path, "/api") {
			apiRoutes = append(apiRoutes, route)
		} else if route.Path == "/*proxyPath" || route.Path == "/*path" {
			routeCopy := route
			rootCatchAllRoute = &routeCopy
		} else {
			specificRoutes = append(specificRoutes, route)
		}
	}

	// 1. 루트 라우트 등록
	for _, route := range rootRoutes {
		h.registerHTTPRoute(router, route)
	}

	// 2. 특정 경로 라우트 등록
	for _, route := range specificRoutes {
		h.registerHTTPRoute(router, route)
	}

	// 3. API 라우트 등록 (그룹 사용)
	apiGroup := router.Group("/api")
	for _, route := range apiRoutes {
		// "/api" 접두사 제거
		subPath := strings.TrimPrefix(route.Path, "/api")
		h.registerHTTPRouteGroup(apiGroup, subPath, route)
	}

	// 4. WebSocket 라우트 등록
	for _, route := range wsRoutes {
		h.registerWebSocketRoute(router, route)
	}

	// 5. 루트 캐치올 라우트 등록 (있는 경우)
	if rootCatchAllRoute != nil {
		log.Println("루트 캐치올 라우트 등록:", rootCatchAllRoute.Path, "->", rootCatchAllRoute.TargetURL)
		
		// 특정 정적 경로에 대한 캐치올 처리
		for _, prefix := range []string{"/web", "/assets", "/static", "/public", "/images"} {
			for _, method := range rootCatchAllRoute.Methods {
				pathWithSuffix := fmt.Sprintf("%s/*path", prefix)
				log.Printf("캐치올 핸들러 등록: %s %s -> %s", method, pathWithSuffix, rootCatchAllRoute.TargetURL)
				
				handlers := h.buildHandlerChain(*rootCatchAllRoute)
				router.Handle(method, pathWithSuffix, handlers...)
			}
		}
		
		// NoRoute 핸들러 등록 (매칭되지 않는 모든 경로)
		router.NoRoute(h.buildHandlerChain(*rootCatchAllRoute)...)
	}

	return nil
}

// HealthCheckHandler는 상태 확인 엔드포인트 핸들러입니다.
func (h *RouteHandler) HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	})
}

// registerHTTPRoute는 HTTP 라우트를 등록합니다.
func (h *RouteHandler) registerHTTPRoute(router *gin.Engine, route config.Route) {
	log.Printf("라우트 등록: %s %s -> %s", strings.Join(route.Methods, ","), route.Path, route.TargetURL)
	
	handlers := h.buildHandlerChain(route)
	
	for _, method := range route.Methods {
		switch method {
		case "GET":
			router.GET(route.Path, handlers...)
		case "POST":
			router.POST(route.Path, handlers...)
		case "PUT":
			router.PUT(route.Path, handlers...)
		case "DELETE":
			router.DELETE(route.Path, handlers...)
		case "PATCH":
			router.PATCH(route.Path, handlers...)
		case "HEAD":
			router.HEAD(route.Path, handlers...)
		case "OPTIONS":
			router.OPTIONS(route.Path, handlers...)
		}
	}
}

// registerHTTPRouteGroup는 라우터 그룹에 HTTP 라우트를 등록합니다.
func (h *RouteHandler) registerHTTPRouteGroup(group *gin.RouterGroup, path string, route config.Route) {
	log.Printf("그룹 라우트 등록: %s %s -> %s", strings.Join(route.Methods, ","), path, route.TargetURL)
	
	handlers := h.buildHandlerChain(route)
	
	for _, method := range route.Methods {
		switch method {
		case "GET":
			group.GET(path, handlers...)
		case "POST":
			group.POST(path, handlers...)
		case "PUT":
			group.PUT(path, handlers...)
		case "DELETE":
			group.DELETE(path, handlers...)
		case "PATCH":
			group.PATCH(path, handlers...)
		case "HEAD":
			group.HEAD(path, handlers...)
		case "OPTIONS":
			group.OPTIONS(path, handlers...)
		}
	}
}

// registerWebSocketRoute는 WebSocket 라우트를 등록합니다.
func (h *RouteHandler) registerWebSocketRoute(router *gin.Engine, route config.Route) {
	log.Printf("WebSocket 라우트 등록: %s -> %s", route.Path, route.TargetURL)
	
	var handlers []gin.HandlerFunc
	
	// 인증이 필요한 경우
	if route.RequireAuth {
		handlers = append(handlers, h.authMiddleware())
	}
	
	// WebSocket 프록시 핸들러 추가
	handlers = append(handlers, h.webSocketProxyHandler(route))
	
	router.GET(route.Path, handlers...)
}

// buildHandlerChain은 라우트에 필요한 미들웨어 핸들러 체인을 구성합니다.
func (h *RouteHandler) buildHandlerChain(route config.Route) []gin.HandlerFunc {
	var handlers []gin.HandlerFunc

	log.Printf("라우트 핸들러 체인 구성: %s, 인증 필요: %v\n", route.Path, route.RequireAuth)
    
	// 타임아웃 미들웨어 (지정된 경우)
	if route.Timeout > 0 {
		handlers = append(handlers, h.timeoutMiddleware(time.Duration(route.Timeout)*time.Second))
	}

	handlers = append(handlers, h.cookieToHeaderMiddleware())

	// 인증 미들웨어 (필요한 경우)
	if route.RequireAuth {
		log.Println("authMiddleware 추가")
		handlers = append(handlers, h.authMiddleware())
	}
	
	// 캐싱 미들웨어 (활성화된 경우)
	if h.config.EnableCaching && route.Cacheable {
		handlers = append(handlers, h.cacheMiddleware())
	}
	
	// 프록시 핸들러 추가
	handlers = append(handlers, h.httpProxyHandler(route))
	
	return handlers
}

// httpProxyHandler는 HTTP 요청을 프록시하는 핸들러를 반환합니다.
func (h *RouteHandler) httpProxyHandler(route config.Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 로드 밸런서에서 대상 서버 선택
		targetURL, err := h.loadBalancer.NextTarget()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "사용 가능한 백엔드 서버가 없습니다"})
			c.Abort()
			return
		}

		// 라우트별 대상 경로 구성
		targetPath := route.TargetURL
		if !strings.HasPrefix(targetPath, "http://") && !strings.HasPrefix(targetPath, "https://") {
			targetPath = fmt.Sprintf("%s%s", targetURL, targetPath)
		}

		// 요청 컨텍스트 설정
		reqCtx := c.Request.Context()

		// 경로 스트립 여부 결정
		stripPath := route.StripPrefix != ""

		// 서킷 브레이커를 통해 요청 실행
		resp, err := h.circuitBreaker.Execute(
			func() (interface{}, error) {
				return proxy.ForwardRequest(reqCtx, c.Request, targetPath, stripPath, route.StripPrefix)
			},
		)

		if err != nil {
			// 요청 실패 처리
			statusCode := http.StatusBadGateway
			switch err {
			case circuitbreaker.ErrCircuitOpen:
				statusCode = http.StatusServiceUnavailable
				log.Printf("[CIRCUIT] 서킷 열림 상태로 요청 거부: %s %s", c.Request.Method, c.Request.URL.Path)
			case context.DeadlineExceeded, context.Canceled:
				statusCode = http.StatusGatewayTimeout
				log.Printf("[TIMEOUT] 요청 타임아웃: %s %s", c.Request.Method, c.Request.URL.Path)
			default:
				log.Printf("[ERROR] 프록시 요청 실패: %s %s - %v", c.Request.Method, c.Request.URL.Path, err)
			}

			c.JSON(statusCode, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// 성공 응답 처리
		httpResp, ok := resp.(*http.Response)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "예상치 못한 응답 유형"})
			c.Abort()
			return
		}

		// 응답 헤더 복사
		for k, values := range httpResp.Header {
			for _, v := range values {
				c.Writer.Header().Add(k, v)
			}
		}

		// 응답 상태 코드 설정
		c.Writer.WriteHeader(httpResp.StatusCode)

		// 응답 본문 복사
		if httpResp.Body != nil {
			defer httpResp.Body.Close()
			_, err = io.Copy(c.Writer, httpResp.Body)
			if err != nil {
				log.Printf("[ERROR] 응답 본문 읽기 오류: %v", err)
			}
		}
	}
}

// webSocketProxyHandler는 WebSocket 요청을 프록시하는 핸들러를 반환합니다.
func (h *RouteHandler) webSocketProxyHandler(route config.Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 로드 밸런서에서 대상 서버 선택
		targetURL, err := h.loadBalancer.NextTarget()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "사용 가능한 백엔드 서버가 없습니다"})
			c.Abort()
			return
		}

		// 라우트별 대상 경로 구성
		targetPath := route.TargetURL
		if !strings.HasPrefix(targetPath, "ws://") && !strings.HasPrefix(targetPath, "wss://") {
			// HTTP 또는 HTTPS 스킴을 WebSocket 스킴으로 변환
			if strings.HasPrefix(targetURL, "https://") {
				targetURL = "wss://" + strings.TrimPrefix(targetURL, "https://")
			} else {
				targetURL = "ws://" + strings.TrimPrefix(targetURL, "http://")
			}
			targetPath = fmt.Sprintf("%s%s", targetURL, targetPath)
		}

		// WebSocket 핸들러 호출
		proxy.WebSocketProxy(c.Writer, c.Request, targetPath, h.wsUpgrader)
	}
}

// timeoutMiddleware는 요청 타임아웃을 설정하는 핸들러를 반환합니다.
func (h *RouteHandler) timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 타임아웃이 있는 컨텍스트 생성
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 요청에 새 컨텍스트 설정
		c.Request = c.Request.WithContext(ctx)

		// 타임아웃 처리를 위한 채널 생성
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// 정상 완료
			return
		case <-ctx.Done():
			// 타임아웃 발생
			if ctx.Err() == context.DeadlineExceeded {
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
					"error": "요청 처리 시간이 초과되었습니다",
				})
			}
		}
	}
}


// authMiddleware는 JWT 인증을 수행하는 핸들러를 반환합니다.
func (h *RouteHandler) cookieToHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {		
		tokenCookie, err := c.Cookie("access_token")
		if err == nil && tokenCookie != "" {
			c.Request.Header.Set("Authorization", "Bearer "+tokenCookie)
		}

		c.Next()
	}
}

// authMiddleware는 JWT 인증을 수행하는 핸들러를 반환합니다.
func (h *RouteHandler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		
		// Authorization 헤더 확인
		authHeader := c.GetHeader("Authorization")
		
		// 토큰이 없는 경우
		if authHeader == "" {
			os.Stdout.Write([]byte(fmt.Sprintf("인증 실패: 토큰 없음\n")))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "인증 토큰이 필요합니다2"})
			c.Abort()
			return
		}

		// Bearer 토큰 추출
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			os.Stdout.Write([]byte(fmt.Sprintf("인증 실패: 유효하지 않은 토큰 형식\n")))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "유효하지 않은 인증 형식입니다"})
			c.Abort()
			return
		}

		// 토큰 검증
		claims, err := h.authenticator.VerifyToken(token)
		if err != nil {
			os.Stdout.Write([]byte(fmt.Sprintf("인증 실패: 토큰 검증 실패 - %v\n", err)))
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("인증 실패: %v", err)})
			c.Abort()
			return
		}

		os.Stdout.Write([]byte(fmt.Sprintf("인증 성공 - 사용자 ID: %s, 역할: %v\n", claims.Subject, claims.Roles)))

		// 인증 정보를 컨텍스트에 저장
		c.Set("userId", claims.Subject)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// cacheMiddleware는 응답 캐싱을 처리하는 핸들러를 반환합니다.
func (h *RouteHandler) cacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// GET 요청만 캐싱
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// 캐시 키 생성
		cacheKey := generateCacheKey(c.Request)

		// 캐시에서 응답 조회
		if cachedResponse, found := h.cache.Get(cacheKey); found {
			// 캐시된 응답 헤더 복원
			headers := cachedResponse.Headers
			for key, values := range headers {
				for _, value := range values {
					c.Writer.Header().Add(key, value)
				}
			}

			// 캐시 헤더 추가
			c.Writer.Header().Set("X-Cache", "HIT")

			// 상태 코드 및 내용 설정
			c.Writer.WriteHeader(cachedResponse.StatusCode)
			c.Writer.Write(cachedResponse.Body)

			// 처리 완료
			c.Abort()
			return
		}

		// 응답 캡처를 위한 래퍼 설정
		responseWriter := newCacheResponseWriter(c.Writer)
		c.Writer = responseWriter

		// 다음 핸들러 실행
		c.Next()

		// 성공 응답만 캐시 (2xx)
		if responseWriter.Status() >= 200 && responseWriter.Status() < 300 {
			// Cache-Control 헤더 확인
			cacheControl := responseWriter.Header().Get("Cache-Control")
			if !strings.Contains(cacheControl, "no-store") && !strings.Contains(cacheControl, "private") {
				// 캐시 TTL 계산
				ttl := h.config.CacheTTL
				if maxAge := extractMaxAge(cacheControl); maxAge > 0 {
					ttl = time.Duration(maxAge) * time.Second
				}

				// 캐시된 응답 저장
				h.cache.Set(cacheKey, &cache.CachedResponse{
					StatusCode: responseWriter.Status(),
					Headers:    responseWriter.Header(),
					Body:       responseWriter.Body(),
				}, ttl)
			}
		}
	}
}

// generateCacheKey는 요청에 대한 고유한 캐시 키를 생성합니다.
func generateCacheKey(req *http.Request) string {
	return fmt.Sprintf("%s:%s:%s", req.Method, req.URL.Path, req.URL.RawQuery)
}

// extractMaxAge는 Cache-Control 헤더에서 max-age 값을 추출합니다.
func extractMaxAge(cacheControl string) int {
	if cacheControl == "" {
		return 0
	}

	for _, directive := range strings.Split(cacheControl, ",") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			age := strings.TrimPrefix(directive, "max-age=")
			if maxAge, err := strconv.Atoi(age); err == nil {
				return maxAge
			}
		}
	}

	return 0
}

// cacheResponseWriter는 응답을 캡처하기 위한 http.ResponseWriter 래퍼입니다.
type cacheResponseWriter struct {
	gin.ResponseWriter
	body   []byte
	status int
}

// newCacheResponseWriter는 새로운 cacheResponseWriter를 생성합니다.
func newCacheResponseWriter(writer gin.ResponseWriter) *cacheResponseWriter {
	return &cacheResponseWriter{
		ResponseWriter: writer,
		status:         http.StatusOK,
	}
}

// Write는 응답 본문을 캡처합니다.
func (w *cacheResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// WriteHeader는 응답 상태 코드를 캡처합니다.
func (w *cacheResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Status는 캡처된 상태 코드를 반환합니다.
func (w *cacheResponseWriter) Status() int {
	return w.status
}

// Body는 캡처된 응답 본문을 반환합니다.
func (w *cacheResponseWriter) Body() []byte {
	return w.body
}
