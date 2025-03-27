# API Gateway

API Gateway는 HTTP/WebSocket 요청을 다양한 백엔드 서비스로 라우팅하는 고성능 리버스 프록시입니다. 이 프로젝트는 마이크로서비스 아키텍처에서 진입점 역할을 하도록 설계되었습니다.

## 주요 기능

- **동적 라우팅**: 구성 기반 라우팅으로 요청을 다양한 백엔드 서비스로 전달
- **부하 분산**: 다양한 알고리즘(라운드 로빈, 가중치 기반, 최소 연결)을 사용한 여러 백엔드 간 부하 분산
- **서킷 브레이커**: 장애가 있는 백엔드 서비스로부터 시스템 보호
- **캐싱**: 응답 캐싱으로 성능 향상 및 백엔드 부하 감소
- **인증/인가**: JWT 기반 인증 및 접근 제어
- **속도 제한**: 과도한 요청으로부터 API 보호
- **WebSocket 지원**: 양방향 실시간 통신을 위한 WebSocket 프록시
- **모니터링**: Prometheus 및 Grafana를 통한 포괄적인 메트릭 수집 및 시각화
- **고급 로깅**: 구조화된 로깅 및 추적 ID를 통한 디버깅 용이성

## 시작하기

### 필수 조건

- Go 1.21+
- Docker 및 Docker Compose (선택 사항)

### 로컬 개발

1. 저장소 복제:
   ```bash
   git clone https://github.com/yourusername/api-gateway.git
   cd api-gateway
   ```

2. 종속성 설치:
   ```bash
   go mod download
   ```

3. 환경 변수 설정:
   ```bash
   cp .env.example .env
   # .env 파일을 원하는 설정으로 편집
   ```

4. 애플리케이션 빌드 및 실행:
   ```bash
   make build
   ./build/api-gateway
   
   # 또는 직접 실행
   go run cmd/gateway/main.go
   ```

5. 개발 모드에서 실행(라이브 리로드 사용):
   ```bash
   make dev
   ```

### Docker로 실행

1. Docker 이미지 빌드:
   ```bash
   make docker-build
   ```

2. Docker 컨테이너 실행:
   ```bash
   make docker-run
   ```

3. Docker Compose로 전체 스택 실행(API Gateway, 백엔드 서비스, Prometheus, Grafana):
   ```bash
   docker-compose up -d
   ```

4. 서비스 접근:
   - API Gateway: http://localhost:8080
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (기본 인증: admin/admin)

## 설정

API Gateway는 다음과 같은 방법으로 설정할 수 있습니다:

1. 환경 변수
2. `.env` 파일
3. 명령줄 플래그

주요 설정 항목은 다음과 같습니다:

| 환경 변수 | 기본값 | 설명 |
|-----------|---------|-------------|
| PORT | 8080 | API Gateway 리스닝 포트 |
| LOG_LEVEL | info | 로그 레벨 (debug, info, warn, error) |
| BACKEND_URL | http://localhost:8081 | 단일 백엔드 서버 URL |
| BACKEND_URLS | - | 쉼표로 구분된 여러 백엔드 서버 URL |
| JWT_SECRET | your-secret-key | JWT 토큰 검증 비밀 키 |
| JWT_ISSUER | api-gateway | JWT 토큰 발행자 |
| JWT_EXPIRATION | 3600 | JWT 토큰 만료 시간(초) |
| ALLOWED_ORIGINS | * | CORS 허용 오리진 (쉼표 구분) |
| ROUTES_CONFIG_PATH | configs/routes.json | 라우트 설정 파일 경로 |
| ENABLE_METRICS | true | Prometheus 메트릭 활성화 여부 |
| ENABLE_CACHING | true | 응답 캐싱 활성화 여부 |
| CACHE_TTL | 300 | 캐시 항목 기본 수명(초) |

전체 설정 옵션은 `.env.example` 파일을 참조하세요.

## 라우트 설정

라우트 설정은 `configs/routes.json` 파일에서 정의됩니다:

```json
{
  "routes": [
    {
      "path": "/api/users",
      "targetURL": "http://users-service:8082/users",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "stripPrefix": "/api",
      "requireAuth": true,
      "cacheable": false,
      "timeout": 20
    },
    // ... 더 많은 라우트 ...
  ]
}
```

각 라우트는 다음 속성을 가질 수 있습니다:

- `path`: 매칭할 경로 패턴
- `targetURL`: 요청을 전달할 대상 URL
- `methods`: 허용된 HTTP 메소드 목록
- `stripPrefix`: 대상으로 전달하기 전에 제거할 경로 접두사
- `requireAuth`: JWT 인증 필요 여부
- `cacheable`: 응답 캐싱 활성화 여부
- `timeout`: 요청 타임아웃(초)

## 아키텍처

API Gateway는 다음과 같은 핵심 컴포넌트로 구성됩니다:

1. **프록시**: HTTP 및 WebSocket 요청을 백엔드 서비스로 전달
2. **라우터**: 들어오는 요청을 적절한 백엔드 서비스로 라우팅
3. **미들웨어**: 인증, 로깅, CORS, 속도 제한 등 처리
4. **부하 분산기**: 여러 백엔드 인스턴스 간 요청 분산
5. **서킷 브레이커**: 장애 감지 및 장애 확산 방지
6. **캐시**: 자주 접근하는 응답 저장
7. **메트릭 수집기**: 시스템 성능 및 상태 모니터링

## 사용 방법

### 요청 예시

```bash
# 기본 요청
curl http://localhost:8080/api/users

# 인증된 요청
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/api/protected-resource

# WebSocket 연결
wscat -c ws://localhost:8080/websocket/events
```

### 인증

API Gateway는 JWT 토큰 기반 인증을 지원합니다:

1. 인증 엔드포인트에서 토큰 획득:
   ```bash
   curl -X POST http://localhost:8080/api/auth/login -d '{"username":"user", "password":"pass"}'
   ```

2. 응답에서 받은 토큰으로 보호된 엔드포인트 접근:
   ```bash
   curl -H "Authorization: Bearer YOUR_JWT_TOKEN" http://localhost:8080/api/protected-resource
   ```

## 모니터링

API Gateway는 다음과 같은 모니터링 기능을 제공합니다:

1. **Prometheus 메트릭**: `/metrics` 엔드포인트에서 사용 가능
2. **Grafana 대시보드**: 요청 속도, 지연 시간, 오류율 등 시각화
3. **구조화된 로깅**: JSON 형식 로그로 쉬운 분석

## 테스트

```bash
# 전체 테스트 실행
make test

# 유닛 테스트만 실행
make test-unit

# 통합 테스트만 실행
make test-integration

# 커버리지 리포트 생성
make coverage
```

## 기여하기

1. 이 저장소를 포크합니다
2. 새 기능 브랜치를 생성합니다: `git checkout -b feature/amazing-feature`
3. 변경 사항을 커밋합니다: `git commit -m 'Add amazing feature'`
4. 브랜치를 푸시합니다: `git push origin feature/amazing-feature`
5. 풀 리퀘스트를 제출합니다

## 라이선스

이 프로젝트는 MIT 라이선스에 따라 라이선스가 부여됩니다 - 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 연락처

프로젝트 링크: [https://github.com/yourusername/api-gateway](https://github.com/yourusername/api-gateway)
