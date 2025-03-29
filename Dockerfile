# 다단계 빌드를 사용한 최적화된 Dockerfile

# 1. 빌드 스테이지
FROM golang:1.21-alpine AS builder

# 빌드에 필요한 도구 설치
RUN apk add --no-cache ca-certificates tzdata git && \
    update-ca-certificates

# 작업 디렉토리 설정
WORKDIR /build

# 종속성 먼저 복사 및 다운로드 (캐싱 활용)
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사 및 빌드
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') \
    -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o gateway ./cmd/gateway

# 2. 최종 이미지
FROM alpine:latest

# 타임존 데이터 및 인증서 복사
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# curl 설치 (헬스체크용)
RUN apk add --no-cache curl

# 바이너리 및 설정 파일 복사
COPY --from=builder /build/gateway /usr/local/bin/gateway
COPY --from=builder /build/configs /etc/gateway/configs

# 작업 디렉토리 생성
WORKDIR /app

# 비특권 사용자로 실행 (보안 강화)
USER 1000:1000

# 환경 변수 설정
ENV PORT=8000 \
    LOG_LEVEL=debug \
    ROUTES_CONFIG_PATH=/etc/gateway/configs/routes.json \
    BACKEND_URL=http://receipt-service:8000 \
    FRONTEND_URL=http://web-client:3000 \
    AUTH_API_URL=http://auth-service:8000

# 상태 점검 설정
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

# 포트 노출
EXPOSE 8000

# 컨테이너 실행 명령
ENTRYPOINT ["/usr/local/bin/gateway"]
