FROM golang:1.21-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 의존성 설치를 위한 모듈 파일 복사
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# 소스 코드 복사
COPY . .

# 애플리케이션 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway .

# 최종 이미지를 위한 경량 베이스 이미지
FROM alpine:latest

# CA 인증서를 설치하여 TLS 연결이 동작하도록 함
RUN apk --no-cache add ca-certificates tzdata

# 애플리케이션 디렉토리 설정
WORKDIR /app

# 빌드 스테이지에서 빌드한 바이너리 복사
COPY --from=builder /app/api-gateway .

# 설정 파일 복사
COPY --from=builder /app/config/routes.json ./config/
COPY --from=builder /app/.env.example ./.env

# 필요한 환경 변수 설정
ENV GIN_MODE=release
ENV PORT=8080

# 포트 설정
EXPOSE 8080

# 애플리케이션 실행
CMD ["./api-gateway"]
