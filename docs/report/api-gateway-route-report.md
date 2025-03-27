
# API 게이트웨이 다중 포트 통합 미션

## 문제 상황
- facreport.iptime.org:8003/login (웹 클라이언트 직접 접속)
- facreport.iptime.org:8000/login (API 게이트웨이 경유)
- 두 URL이 동일한 로그인 페이지를 반환해야 하나, API 게이트웨이 라우팅 문제로 인해 불가능한 상태였음

## 문제 분석

### 시스템 구성 확인
- 8003 포트: 웹 클라이언트 컨테이너(React) 접속
- 8000 포트: API 게이트웨이 컨테이너(Golang/Gin) 접속
- API 게이트웨이는 routes.json을 통해 라우팅 규칙 정의

### 주요 문제점 발견
1. 라우트 충돌: `/*proxyPath` 와일드카드 라우트가 `/api` 경로와 충돌
2. 라우트 순서: 와일드카드 라우트가 구체적인 경로보다 먼저 등록되어 적절한 라우팅 불가
3. API 게이트웨이가 충돌로 인해 정상 시작되지 않음

## 해결 접근법

### 1. 라우트 설정 수정
- `/login` 경로를 routes.json에 추가
- 라우트 순서 최적화 (구체적 경로 먼저, 와일드카드 나중에)

### 2. 라우트 처리 로직 개선
- 라우트 타입별 그룹화 (루트, API, 특정 경로, 캐치올)
- API 경로는 라우터 그룹으로 등록하여 충돌 방지
- 와일드카드 캐치올 라우트는 NoRoute 핸들러로 등록

### 3. 다중 포트 지원
- API 게이트웨이에 AdditionalPorts 설정 추가
- 8000, 8003 포트 모두에서 동일한 라우팅 규칙 적용

## 주요 변경 사항

### 1. 파일 변경
- **api-gateway/config/routes.json**: `/login` 경로 추가
- **api-gateway/config/config.go**: `AdditionalPorts` 필드 추가
- **api-gateway/main.go**: 라우트 등록 로직 개선, 다중 포트 지원
- **api-gateway/proxy/reverseproxy.go**: 로깅 기능 강화

### 2. 라우트 처리 로직 핵심 개선점
```go
// 라우트 그룹화
var rootRoutes []RouteConfig
var apiRoutes []RouteConfig
var specificRoutes []RouteConfig
var rootCatchAllRoute *RouteConfig

// API 라우트는 그룹으로 등록
apiGroup := router.Group("/api")

// 캐치올 라우트는 NoRoute 핸들러로 등록
router.NoRoute(func(c *gin.Context) {
    proxy.HTTPProxyHandler(httpProxy, rootCatchAllRoute.TargetURL, false)(c)
})
```

### 3. 디버깅 기법
- 라우트 등록 순서 로깅
- 프록시 요청/응답 로깅
- 와일드카드 라우트 충돌 해결을 위한 특수 처리

## 해결 과정 중 맞닥뜨린 문제
1. 와일드카드 라우팅 충돌: Gin 라우터는 와일드카드(`*`)와 정적 경로 간 충돌 처리 제한
2. 라우트 등록 순서: 라우트 등록 순서가 매칭 우선순위에 영향
3. API 경로 충돌: `/*proxyPath`가 `/api` 경로와 충돌

## 최종 해결책
1. API 경로는 라우터 그룹으로 분리
2. 특정 정적 경로는 명시적으로 먼저, 높은 우선순위로 등록
3. 캐치올 라우트는 NoRoute 핸들러로 등록하여 충돌 방지
4. 다중 포트 지원으로 8000, 8003 모두에서 동일한 페이지 제공

## 교훈
1. 라우팅 순서는 API 게이트웨이에서 매우 중요
2. 와일드카드 라우팅은 신중하게 설계해야 함
3. 라우터 그룹 활용이 정교한 라우팅 구현에 효과적
4. 적절한 로깅은 문제 해결에 필수적
