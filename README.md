# API Gateway

고성능 마이크로서비스 API 게이트웨이 서비스입니다. Go와 Gin 프레임워크로 구현되었으며, 다양한 마이크로서비스 환경에서 사용할 수 있습니다.

## 주요 기능

- **리버스 프록시**: 백엔드 서비스로 요청을 라우팅합니다.
- **WebSocket 프록시**: WebSocket 연결을 지원합니다.
- **인증 & 인가**: JWT 기반 인증을 지원합니다.
- **레이트 리미팅**: 과도한 요청으로부터 서비스를 보호합니다.
- **CORS 지원**: 크로스 오리진 요청을 안전하게 처리합니다.
- **로깅 & 모니터링**: Prometheus 메트릭을 통한 모니터링을 지원합니다.
- **타임아웃 관리**: 적절한 타임아웃 설정으로 시스템 안정성을 향상시킵니다.
- **환경 변수 기반 설정**: `.env` 파일을 통한 쉬운 구성이 가능합니다.

## 시작하기

### 요구 사항

- Go 1.18 이상
- Git (설치 및 소스 코드 관리용)

### 설치 및 실행

1. 저장소 클론:
```bash
git clone https://github.com/isinthesky/api-gateway.git
cd api-gateway
```

2. 의존성 설치:
```bash
go mod tidy
```

3. 환경 설정:
```bash
cp .env.example .env
# .env 파일을 필요에 맞게 수정하세요
```

4. 애플리케이션 실행:
```bash
go run main.go
```

### 설정

`config/routes.json` 파일을 통해 라우팅 경로를 구성할 수 있습니다:

```json
{
  "routes": [
    {
      "path": "/api/users/*path",
      "targetURL": "http://users-service:8081",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "requireAuth": true,
      "stripPrefix": "/api/users"
    }
  ]
}
```

환경 변수는 다음과 같이 설정할 수 있습니다:

```
# API Gateway 기본 설정
PORT=8080
LOG_LEVEL=info  # debug, info, warn, error

# 백엔드 서비스 설정
BACKEND_URL=http://localhost:8081

# CORS 설정
ALLOWED_ORIGINS=*  # 쉼표로 구분된 오리진 목록

# JWT 설정
JWT_SECRET=your_secret_key
JWT_ISSUER=api-gateway
JWT_EXPIRATION=3600  # 초 단위

# 레이트 리미팅 설정
RATE_LIMIT_WINDOW=60  # 윈도우 크기 (초)
RATE_LIMIT_MAX_REQUESTS=100  # 윈도우 당 최대 요청 수
```

## 아키텍처

API 게이트웨이는 다음과 같은 구조로 설계되었습니다:

```
api-gateway/
├── config/         # 구성 파일 및 설정 관리
├── internal/       # 내부 패키지
│   ├── middleware/ # 미들웨어 구현
│   └── proxy/      # 프록시 기능 구현 (기존 코드)
├── proxy/          # 프록시 기능 구현 (새 코드)
├── middleware/     # 미들웨어 구현 (새 코드)
├── main.go         # 애플리케이션 진입점
└── .env            # 환경 변수 설정
```

## 기여하기

1. 이 저장소를 포크하세요
2. 새로운 기능 브랜치를 만드세요 (`git checkout -b feature/amazing-feature`)
3. 변경사항을 커밋하세요 (`git commit -m 'Add some amazing feature'`)
4. 브랜치에 푸시하세요 (`git push origin feature/amazing-feature`)
5. Pull Request를 열어주세요

## 라이센스

이 프로젝트는 MIT 라이센스를 따릅니다. 자세한 내용은 `LICENSE` 파일을 참조하세요. 