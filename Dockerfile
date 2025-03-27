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
FROM scratch

# 타임존 데이터 및 인증서 복사
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 바이너리 및 설정 파일 복사
COPY --from=builder /build/gateway /gateway
COPY --from=builder /build/configs /configs

# 비특권 사용자로 실행 (보안 강화)
USER 1000:1000

# 환경 변수 설정
ENV PORT=8080 \
    LOG_LEVEL=info \
    ROUTES_CONFIG_PATH=/configs/routes.json

# 상태 점검 설정
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD [ "/gateway", "health" ] || exit 1

# 포트 노출
EXPOSE 8080

# 컨테이너 실행 명령
ENTRYPOINT ["/gateway"]
