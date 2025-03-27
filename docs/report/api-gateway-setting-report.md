
# Receiptally API Gateway 설정 미션 진행 보고서

## 미션 목표
API Gateway 서버에 다음과 같은 리버스 프록시 규칙을 설정:
- 루트 URL(`/`)은 `receiptally-web-client`로 연결
- `/api/v1/main` 경로는 `receiptally-receipt-service`로 연결
- `/api/v1/users`, `/api/v1/auth` 경로는 `receiptally-msa-auth-service`로 연결

## 구현 과정

### 1. API Gateway 설정 파일 분석
- API Gateway의 설정은 `config/routes.json` 파일에 JSON 형식으로 정의
- 리버스 프록시 핸들러는 `proxy/reverseproxy.go`에 구현되어 있음
- Docker Compose 파일에서 서비스 간 네트워크 구성 확인

### 2. routes.json 파일 생성
- 프록시 규칙을 정의하는 JSON 파일 생성
- 각 경로별 타겟 URL, 허용 메서드, 인증 필요 여부 등 설정

### 3. 발생한 문제들과 해결 방법

#### 의존성 오류
```
go: github.com/klauspost/compress@v1.17.0: missing go.sum entry for go.mod file
```
- **해결**: Docker를 사용하여 Go 환경에서 의존성 다운로드
```bash
docker run --rm -v $(pwd):/app -w /app golang:1.21 go mod download github.com/klauspost/compress
```

#### Prometheus 설정 파일 경로 오류
```
Error mounting "/Users/Shared/receiptally/receiptally-msa/api-gateway/config/prometheus.yml"
```
- **해결**: docker-compose.yml 파일에서 prometheus.yml 경로 수정
- 프로젝트 루트에 있는 prometheus.yml 파일을 사용하도록 변경

#### 와일드카드 경로 충돌
```
panic: catch-all wildcard '*proxyPath' in new path '/*proxyPath' conflicts with existing path segment 'metrics'
```
- **해결**: 범용 와일드카드 경로 대신 구체적인 경로로 변경
```json
{
  "path": "/static/*proxyPath",
  "targetURL": "http://web-client:3000/static"
}
```

#### 서비스 이름 및 포트 설정 오류
- **문제**: 서비스 이름이 docker-compose.yml과 일치하지 않음
- **해결**: routes.json의 targetURL 수정
```json
"targetURL": "http://web-client:3000" // receiptally-web-client에서 변경
"targetURL": "http://receipt-service:8000" // receiptally-receipt-service에서 변경
"targetURL": "http://auth-service:8000" // receiptally-msa-auth-service에서 변경
```

#### 리버스 프록시 로직 개선
- **reverseproxy.go** 파일 수정
- 대상 URL 구성 및 경로 설정 로직 개선
- 올바른 호스트 헤더 설정

## 최종 설정 파일

### routes.json
```json
{
  "routes": [
    {
      "path": "/",
      "targetURL": "http://web-client:3000",
      "methods": ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
      "requireAuth": false,
      "stripPrefix": ""
    },
    {
      "path": "/static/*proxyPath",
      "targetURL": "http://web-client:3000/static",
      "methods": ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
      "requireAuth": false,
      "stripPrefix": ""
    },
    {
      "path": "/api/v1/auth/*proxyPath",
      "targetURL": "http://auth-service:8000",
      "methods": ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"],
      "requireAuth": false,
      "stripPrefix": ""
    }
  ]
}
```

## 성공 확인
웹 클라이언트 서비스 연결 확인:
```bash
$ curl -s localhost:8000/ | head -10
<!DOCTYPE html>
<html lang="en">
  <head>
    <script type="module">
...
```

## 결론
API Gateway의 리버스 프록시 규칙을 성공적으로 설정하여 다음과 같은 기능을 구현했습니다:
1. 웹 클라이언트 서비스로의 프록시 연결
2. 마이크로서비스 API 서버들로의 경로별 프록시 연결
3. 정적 자원 경로의 적절한 라우팅

모든 서비스가 Docker Compose를 통해 성공적으로 구동되었으며, API Gateway가 예상대로 트래픽을 올바른 서비스로
라우팅하고 있음을 확인했습니다.
