FROM golang:1.21-alpine

# 작업 디렉토리 설정
WORKDIR /app

# 필요한 개발 도구 설치
RUN apk add --no-cache git curl tzdata ca-certificates && \
    go install github.com/cosmtrek/air@v1.44.0

# 의존성 설치를 위한 모듈 파일 복사
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# 설정 파일 복사
COPY ./config /app/config
COPY .env.example /app/.env

# 소스 코드 복사
COPY . /app/

# 포트 설정
EXPOSE 8000

# 환경 변수 설정
ENV GIN_MODE=debug
ENV PORT=8000

# 개발 모드에서 코드 변경 시 자동 재시작을 위한 air 사용
CMD ["air", "-c", ".air.toml"]
