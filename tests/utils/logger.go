package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// 로그 레벨 정의
const (
    LevelDebug = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)

var levelNames = map[int]string{
    LevelDebug: "DEBUG",
    LevelInfo:  "INFO",
    LevelWarn:  "WARN",
    LevelError: "ERROR",
    LevelFatal: "FATAL",
}

// TestLogger는 테스트 전용 로거입니다
type TestLogger struct {
    logger *log.Logger
    level  int
}

// NewTestLogger는 새 테스트 로거를 생성합니다
func NewTestLogger(level int) *TestLogger {
    return &TestLogger{
        logger: log.New(os.Stdout, "", 0),
        level:  level,
    }
}

// log는 지정된 레벨에 메시지를 기록합니다
func (l *TestLogger) log(level int, format string, args ...interface{}) {
    if level < l.level {
        return
    }

    // 호출자 정보 가져오기
    _, file, line, _ := runtime.Caller(2)
    file = filepath.Base(file)

    // 타임스탬프 및 레벨 정보
    timestamp := time.Now().Format("2006-01-02 15:04:05.000")
    levelName := levelNames[level]

    // 최종 메시지 형식
    prefix := fmt.Sprintf("[%s] [%s] [%s:%d] ", timestamp, levelName, file, line)
    message := fmt.Sprintf(format, args...)
    
    l.logger.Println(prefix + message)
}

// Debug는 디버그 수준 메시지를 기록합니다
func (l *TestLogger) Debug(format string, args ...interface{}) {
    l.log(LevelDebug, format, args...)
}

// Info는 정보 수준 메시지를 기록합니다
func (l *TestLogger) Info(format string, args ...interface{}) {
    l.log(LevelInfo, format, args...)
}

// Warn은 경고 수준 메시지를 기록합니다
func (l *TestLogger) Warn(format string, args ...interface{}) {
    l.log(LevelWarn, format, args...)
}

// Error는 오류 수준 메시지를 기록합니다
func (l *TestLogger) Error(format string, args ...interface{}) {
    l.log(LevelError, format, args...)
}

// Fatal은 치명적 오류 메시지를 기록합니다
func (l *TestLogger) Fatal(format string, args ...interface{}) {
    l.log(LevelFatal, format, args...)
    os.Exit(1)
}

// TestContext는 테스트 컨텍스트와 로그 정보를 포함합니다
func (l *TestLogger) TestContext(testName string) *TestLogger {
    prefix := fmt.Sprintf("[%s] ", testName)
    contextLogger := &TestLogger{
        logger: log.New(os.Stdout, prefix, 0),
        level:  l.level,
    }
    return contextLogger
}
