package circuitbreaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// 상태 상수
const (
	StateClosed   int32 = iota // 정상 상태
	StateHalfOpen              // 반열림 상태 (복구 시도)
	StateOpen                  // 열림 상태 (요청 차단)
)

// 오류 정의
var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
	ErrInvalidState = errors.New("invalid circuit breaker state")
)

// Config는 서킷 브레이커 설정입니다.
type Config struct {
	ErrorThreshold   float64       // 오류 비율 임계값 (0.0-1.0)
	MinRequests      int           // 상태 결정을 위한 최소 요청 수
	TimeoutDuration  time.Duration // 열림 상태에서 반열림으로 전환하는 시간
	HalfOpenMaxReqs  int           // 반열림 상태에서 허용할 최대 요청 수
	SuccessThreshold int           // 닫힘 상태로 돌아가기 위한 연속 성공 횟수
}

// CircuitBreaker는 서킷 브레이커 패턴을 구현하는 구조체입니다.
type CircuitBreaker struct {
	state           int32         // 현재 상태 (닫힘, 반열림, 열림)
	config          Config        // 서킷 브레이커 설정
	mutex           sync.RWMutex  // 동시성 제어용 뮤텍스
	
	// 메트릭 관련 필드
	successCount    int64         // 성공한 요청 수
	failureCount    int64         // 실패한 요청 수
	requestCount    int64         // 총 요청 수
	consecutiveSuccesses int64    // 연속 성공 횟수
	
	// 시간 관련 필드
	lastStateChange time.Time     // 마지막 상태 변경 시간
	timeoutStart    time.Time     // 타임아웃 시작 시간
}

// New는 새로운 서킷 브레이커를 생성합니다.
func New(config Config) *CircuitBreaker {
	// 기본값 설정
	if config.ErrorThreshold <= 0 {
		config.ErrorThreshold = 0.5 // 기본 50% 오류율
	}
	if config.MinRequests <= 0 {
		config.MinRequests = 10 // 기본 10개 요청
	}
	if config.TimeoutDuration <= 0 {
		config.TimeoutDuration = 60 * time.Second // 기본 1분 타임아웃
	}
	if config.HalfOpenMaxReqs <= 0 {
		config.HalfOpenMaxReqs = 5 // 기본 5개 테스트 요청
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2 // 기본 2번 연속 성공
	}

	now := time.Now()
	return &CircuitBreaker{
		state:           StateClosed,
		config:          config,
		lastStateChange: now,
		timeoutStart:    now,
	}
}

// Execute는 서킷 브레이커를 통해 함수를 실행합니다.
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	// 현재 상태 확인
	state := atomic.LoadInt32(&cb.state)

	// 상태에 따른 처리
	switch state {
	case StateClosed:
		// 닫힘 상태 - 정상 실행
		return cb.executeClosed(fn)
		
	case StateHalfOpen:
		// 반열림 상태 - 제한된 요청 허용
		return cb.executeHalfOpen(fn)
		
	case StateOpen:
		// 열림 상태 - 타임아웃 확인 후 처리
		return cb.executeOpen(fn)
		
	default:
		return nil, ErrInvalidState
	}
}

// GetState는 현재 서킷 브레이커 상태를 반환합니다.
func (cb *CircuitBreaker) GetState() string {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// GetMetrics는 서킷 브레이커의 현재 메트릭을 반환합니다.
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	// 에러율 계산
	totalRequests := cb.successCount + cb.failureCount
	errorRate := float64(0)
	if totalRequests > 0 {
		errorRate = float64(cb.failureCount) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"state": cb.GetState(),
		"total_requests": totalRequests,
		"success_count": cb.successCount,
		"failure_count": cb.failureCount,
		"error_rate": errorRate,
		"consecutive_successes": cb.consecutiveSuccesses,
		"last_state_change": cb.lastStateChange,
		"timeout_start": cb.timeoutStart,
	}
}

// Reset은 서킷 브레이커 상태를 초기화합니다.
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	atomic.StoreInt32(&cb.state, StateClosed)
	cb.successCount = 0
	cb.failureCount = 0
	cb.requestCount = 0
	cb.consecutiveSuccesses = 0
	cb.lastStateChange = time.Now()
}

// executeClosed는 닫힘 상태에서 함수를 실행합니다.
func (cb *CircuitBreaker) executeClosed(fn func() (interface{}, error)) (interface{}, error) {
	result, err := fn()
	
	// 요청 결과 처리
	if err != nil {
		cb.recordFailure()
		return result, err
	}
	
	cb.recordSuccess()
	return result, nil
}

// executeHalfOpen는 반열림 상태에서 함수를 실행합니다.
func (cb *CircuitBreaker) executeHalfOpen(fn func() (interface{}, error)) (interface{}, error) {
	// 허용된 요청 수 확인
	if atomic.LoadInt64(&cb.requestCount) >= int64(cb.config.HalfOpenMaxReqs) {
		return nil, ErrTooManyRequests
	}
	
	// 요청 카운터 증가
	atomic.AddInt64(&cb.requestCount, 1)
	
	// 함수 실행
	result, err := fn()
	
	// 결과 처리
	if err != nil {
		cb.recordFailure()
		// 실패시 다시 열림 상태로 전환
		cb.transitionToOpen()
		return result, err
	}
	
	// 성공 처리
	cb.recordSuccess()
	
	// 연속 성공 확인
	if atomic.LoadInt64(&cb.consecutiveSuccesses) >= int64(cb.config.SuccessThreshold) {
		cb.transitionToClosed()
	}
	
	return result, nil
}

// executeOpen는 열림 상태에서 함수를 실행합니다.
func (cb *CircuitBreaker) executeOpen(fn func() (interface{}, error)) (interface{}, error) {
	// 타임아웃 경과 확인
	cb.mutex.RLock()
	timeout := cb.timeoutStart.Add(cb.config.TimeoutDuration)
	cb.mutex.RUnlock()
	
	// 타임아웃 경과 시 반열림 상태로 전환
	if time.Now().After(timeout) {
		cb.transitionToHalfOpen()
		return cb.executeHalfOpen(fn)
	}
	
	// 타임아웃 미경과 시 오류 반환
	return nil, ErrCircuitOpen
}

// 상태 전이 함수들

// transitionToOpen은 서킷 브레이커를 열림 상태로 전환합니다.
func (cb *CircuitBreaker) transitionToOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	if atomic.LoadInt32(&cb.state) == StateOpen {
		return // 이미 열림 상태
	}
	
	atomic.StoreInt32(&cb.state, StateOpen)
	cb.timeoutStart = time.Now()
	cb.lastStateChange = time.Now()
	cb.requestCount = 0
	cb.consecutiveSuccesses = 0
}

// transitionToHalfOpen은 서킷 브레이커를 반열림 상태로 전환합니다.
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	if atomic.LoadInt32(&cb.state) == StateHalfOpen {
		return // 이미 반열림 상태
	}
	
	atomic.StoreInt32(&cb.state, StateHalfOpen)
	cb.lastStateChange = time.Now()
	cb.requestCount = 0
	cb.consecutiveSuccesses = 0
}

// transitionToClosed는 서킷 브레이커를 닫힘 상태로 전환합니다.
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	if atomic.LoadInt32(&cb.state) == StateClosed {
		return // 이미 닫힘 상태
	}
	
	atomic.StoreInt32(&cb.state, StateClosed)
	cb.lastStateChange = time.Now()
	cb.requestCount = 0
	cb.consecutiveSuccesses = 0
	cb.successCount = 0
	cb.failureCount = 0
}

// 실행 결과 기록 함수들

// recordSuccess는 성공적인 요청을 기록합니다.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	atomic.AddInt64(&cb.successCount, 1)
	atomic.StoreInt64(&cb.consecutiveSuccesses, atomic.LoadInt64(&cb.consecutiveSuccesses)+1)
	
	// 닫힘 상태에서 오류율 확인
	if atomic.LoadInt32(&cb.state) == StateClosed {
		totalRequests := cb.successCount + cb.failureCount
		
		// 최소 요청 수 충족 및 오류율 임계값 초과 시 열림 상태로 전환
		if totalRequests >= int64(cb.config.MinRequests) {
			errorRate := float64(cb.failureCount) / float64(totalRequests)
			if errorRate >= cb.config.ErrorThreshold {
				// 뮤텍스 외부에서 상태 전이 호출
				go cb.transitionToOpen()
			}
		}
	}
}

// recordFailure는 실패한 요청을 기록합니다.
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	atomic.AddInt64(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.consecutiveSuccesses, 0)
	
	// 닫힘 상태에서 오류율 확인
	if atomic.LoadInt32(&cb.state) == StateClosed {
		totalRequests := cb.successCount + cb.failureCount
		
		// 최소 요청 수 충족 및 오류율 임계값 초과 시 열림 상태로 전환
		if totalRequests >= int64(cb.config.MinRequests) {
			errorRate := float64(cb.failureCount) / float64(totalRequests)
			if errorRate >= cb.config.ErrorThreshold {
				// 뮤텍스 외부에서 상태 전이 호출
				go cb.transitionToOpen()
			}
		}
	}
}

// FormatState는 현재 서킷 브레이커 상태를 문자열로 반환합니다.
func (cb *CircuitBreaker) FormatState() string {
	metrics := cb.GetMetrics()
	state := metrics["state"].(string)
	
	return fmt.Sprintf(
		"CircuitBreaker state: %s, error rate: %.2f%% (%d/%d), consecutive successes: %d",
		state,
		metrics["error_rate"].(float64) * 100.0,
		metrics["failure_count"].(int64),
		metrics["total_requests"].(int64),
		metrics["consecutive_successes"].(int64),
	)
}
