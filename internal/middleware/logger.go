package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogLevel은 로그 레벨을 정의합니다.
type LogLevel int

const (
	// LogLevelDebug는 디버그 레벨 로그입니다.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo는 정보 레벨 로그입니다.
	LogLevelInfo
	// LogLevelWarn는 경고 레벨 로그입니다.
	LogLevelWarn
	// LogLevelError는 오류 레벨 로그입니다.
	LogLevelError
)

// LogFormatter는 로그 형식화를 위한 함수 타입입니다.
type LogFormatter func(params LogFormatterParams) string

// LogFormatterParams는 로그 형식화에 사용되는 파라미터입니다.
type LogFormatterParams struct {
	RequestID   string        `json:"request_id"`
	TimeStamp   time.Time     `json:"time"`
	Level       LogLevel      `json:"level"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	Query       string        `json:"query"`
	Status      int           `json:"status"`
	Latency     time.Duration `json:"latency"`
	ClientIP    string        `json:"client_ip"`
	UserAgent   string        `json:"user_agent"`
	ErrorMsg    string        `json:"error,omitempty"`
	RequestBody string        `json:"request_body,omitempty"`
	Message     string        `json:"message,omitempty"`
}

// defaultLogFormatter는 기본 로그 형식화 함수입니다.
var defaultLogFormatter = func(param LogFormatterParams) string {
	// 로그 레벨 문자열
	var level string
	switch param.Level {
	case LogLevelDebug:
		level = "DEBUG"
	case LogLevelInfo:
		level = "INFO"
	case LogLevelWarn:
		level = "WARN"
	case LogLevelError:
		level = "ERROR"
	default:
		level = "INFO"
	}

	// 기본 형식화된 로그 메시지
	return fmt.Sprintf("[%s] %s | %s | %s | %s %s | %d | %v | %s | %s | %s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		level,
		param.RequestID,
		param.ClientIP,
		param.Method,
		param.Path,
		param.Status,
		param.Latency,
		param.UserAgent,
		param.ErrorMsg,
		param.Message,
	)
}

// jsonLogFormatter는 JSON 형식의 로그 형식화 함수입니다.
var jsonLogFormatter = func(param LogFormatterParams) string {
	// 불필요한 빈 필드 제거
	if param.ErrorMsg == "" {
		param.ErrorMsg = "null"
	}
	if param.RequestBody == "" {
		param.RequestBody = "null"
	}
	if param.Message == "" {
		param.Message = "null"
	}

	// JSON으로 인코딩
	jsonBytes, err := json.Marshal(param)
	if err != nil {
		return fmt.Sprintf("[로그 형식화 오류] %v", err)
	}
	return string(jsonBytes)
}

// Logger는 구조화된 로그를 위한 로거 구조체입니다.
type Logger struct {
	Output    io.Writer
	Formatter LogFormatter
	mu        sync.Mutex
}

// NewLogger는 새로운 구조화된 로거를 생성합니다.
func NewLogger(useJSON bool) *Logger {
	var formatter LogFormatter
	if useJSON {
		formatter = jsonLogFormatter
	} else {
		formatter = defaultLogFormatter
	}

	return &Logger{
		Output:    os.Stdout,
		Formatter: formatter,
	}
}

// log는 지정된 레벨로 로그를 출력합니다.
func (l *Logger) log(level LogLevel, param LogFormatterParams) {
	param.Level = level
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.Output, l.Formatter(param))
}

// Debug는 디버그 레벨 로그를 출력합니다.
func (l *Logger) Debug(param LogFormatterParams) {
	l.log(LogLevelDebug, param)
}

// Info는 정보 레벨 로그를 출력합니다.
func (l *Logger) Info(param LogFormatterParams) {
	l.log(LogLevelInfo, param)
}

// Warn는 경고 레벨 로그를 출력합니다.
func (l *Logger) Warn(param LogFormatterParams) {
	l.log(LogLevelWarn, param)
}

// Error는 오류 레벨 로그를 출력합니다.
func (l *Logger) Error(param LogFormatterParams) {
	l.log(LogLevelError, param)
}

// bodyLogWriter는 응답 본문을 캡처하기 위한 ResponseWriter 래퍼입니다.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write는 응답 본문을 캡처합니다.
func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// StructuredLogger는 요청과 응답을 로깅하는 미들웨어입니다.
func StructuredLogger() gin.HandlerFunc {
	logger := NewLogger(false) // 기본 텍스트 형식 사용

	return func(c *gin.Context) {
		// /health 엔드포인트는 로깅하지 않음
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}
		// 시작 시간
		start := time.Now()

		// 요청 ID 생성
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Request.Header.Set("X-Request-ID", requestID)
		}

		// 요청 경로와 쿼리 파라미터
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 요청 본문 캡처 (필요한 경우)
		var requestBody string
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.Request.Body != nil {
				bodyBytes, _ := io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				requestBody = string(bodyBytes)
			}
		}

		// 응답 본문 캡처를 위한 래퍼 설정
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 컨텍스트에 로거 저장
		c.Set("logger", logger)
		c.Set("requestID", requestID)

		// 요청 로깅
		logger.Info(LogFormatterParams{
			RequestID:   requestID,
			TimeStamp:   time.Now(),
			Method:      c.Request.Method,
			Path:        path,
			Query:       query,
			ClientIP:    c.ClientIP(),
			UserAgent:   c.Request.UserAgent(),
			RequestBody: requestBody,
			Message:     "요청 수신",
		})

		// 다음 핸들러 실행
		c.Next()

		// 응답 지연 시간
		latency := time.Since(start)

		// 오류 메시지 (있는 경우)
		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		// 응답 로깅
		logParam := LogFormatterParams{
			RequestID: requestID,
			TimeStamp: time.Now(),
			Method:    c.Request.Method,
			Path:      path,
			Query:     query,
			Status:    c.Writer.Status(),
			Latency:   latency,
			ClientIP:  c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			ErrorMsg:  errorMsg,
			Message:   fmt.Sprintf("응답 완료 (처리 시간: %v)", latency),
		}

		if len(c.Errors) > 0 {
			logger.Error(logParam)
		} else if c.Writer.Status() >= 400 {
			logger.Warn(logParam)
		} else {
			logger.Info(logParam)
		}
	}
}
