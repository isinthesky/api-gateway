package loadbalancer

import (
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// 로드 밸런싱 오류 정의
var (
	ErrNoAvailableTargets = errors.New("no available targets")
	ErrTargetUnreachable = errors.New("target is unreachable")
)

// Target은 로드 밸런서 대상 서버입니다.
type Target struct {
	URL           string
	Healthy       bool
	LastChecked   time.Time
	FailureCount  int
	SuccessCount  int
	Weight        int
	ActiveConns   int64 // 활성 연결 수
}

// LoadBalancer는 부하 분산 기능을 제공하는 인터페이스입니다.
type LoadBalancer interface {
	NextTarget() (string, error)
	AddTarget(url string, weight int) error
	RemoveTarget(url string) error
	MarkTargetDown(url string) error
	MarkTargetUp(url string) error
	GetTargets() []*Target
}

// RoundRobinBalancer는 라운드 로빈 부하 분산 구현체입니다.
type RoundRobinBalancer struct {
	targets  []*Target
	position int64
	mu       sync.RWMutex
}

// NewRoundRobin은 새로운 라운드 로빈 로드 밸런서를 생성합니다.
func NewRoundRobin(urls []string) *RoundRobinBalancer {
	lb := &RoundRobinBalancer{
		targets:  make([]*Target, 0, len(urls)),
		position: 0,
	}

	// 초기 타겟 추가
	for _, urlStr := range urls {
		lb.AddTarget(urlStr, 1) // 기본 가중치 1
	}

	return lb
}

// NextTarget은 다음 대상 서버를 반환합니다.
func (lb *RoundRobinBalancer) NextTarget() (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.targets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 건강한 대상 서버만 필터링
	healthyTargets := make([]*Target, 0)
	for _, target := range lb.targets {
		if target.Healthy {
			healthyTargets = append(healthyTargets, target)
		}
	}

	if len(healthyTargets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 라운드 로빈으로 다음 대상 선택
	position := atomic.AddInt64(&lb.position, 1) % int64(len(healthyTargets))
	selectedTarget := healthyTargets[position]

	// 활성 연결 수 증가
	atomic.AddInt64(&selectedTarget.ActiveConns, 1)

	return selectedTarget.URL, nil
}

// AddTarget은 새로운 대상 서버를 추가합니다.
func (lb *RoundRobinBalancer) AddTarget(urlStr string, weight int) error {
	// URL 유효성 검사
	_, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 이미 존재하는지 확인
	for _, target := range lb.targets {
		if target.URL == urlStr {
			return nil // 이미 존재하는 대상
		}
	}

	// 가중치 확인
	if weight <= 0 {
		weight = 1
	}

	// 새 대상 추가
	lb.targets = append(lb.targets, &Target{
		URL:          urlStr,
		Healthy:      true, // 기본적으로 건강함으로 설정
		LastChecked:  time.Now(),
		FailureCount: 0,
		SuccessCount: 0,
		Weight:       weight,
		ActiveConns:  0,
	})

	return nil
}

// RemoveTarget은 대상 서버를 제거합니다.
func (lb *RoundRobinBalancer) RemoveTarget(urlStr string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, target := range lb.targets {
		if target.URL == urlStr {
			// 인덱스 i에서 요소 제거
			lb.targets = append(lb.targets[:i], lb.targets[i+1:]...)
			return nil
		}
	}

	return errors.New("target not found")
}

// MarkTargetDown은 대상 서버를 비정상 상태로 표시합니다.
func (lb *RoundRobinBalancer) MarkTargetDown(urlStr string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, target := range lb.targets {
		if target.URL == urlStr {
			target.Healthy = false
			target.LastChecked = time.Now()
			target.FailureCount++
			return nil
		}
	}

	return errors.New("target not found")
}

// MarkTargetUp은 대상 서버를 정상 상태로 표시합니다.
func (lb *RoundRobinBalancer) MarkTargetUp(urlStr string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, target := range lb.targets {
		if target.URL == urlStr {
			target.Healthy = true
			target.LastChecked = time.Now()
			target.SuccessCount++
			target.FailureCount = 0 // 실패 횟수 초기화
			return nil
		}
	}

	return errors.New("target not found")
}

// GetTargets는 모든 대상 서버 목록을 반환합니다.
func (lb *RoundRobinBalancer) GetTargets() []*Target {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// 대상 목록 복사본 생성
	targets := make([]*Target, len(lb.targets))
	copy(targets, lb.targets)

	return targets
}

// WeightedRoundRobinBalancer는 가중치 기반 라운드 로빈 로드 밸런서 구현체입니다.
type WeightedRoundRobinBalancer struct {
	RoundRobinBalancer
	weights map[string]int // URL별 가중치
}

// NewWeightedRoundRobin은 새 가중치 기반 라운드 로빈 로드 밸런서를 생성합니다.
func NewWeightedRoundRobin(urlWeights map[string]int) *WeightedRoundRobinBalancer {
	lb := &WeightedRoundRobinBalancer{
		RoundRobinBalancer: RoundRobinBalancer{
			targets:  make([]*Target, 0, len(urlWeights)),
			position: 0,
		},
		weights: make(map[string]int),
	}

	// 초기 타겟 추가
	for url, weight := range urlWeights {
		lb.AddTarget(url, weight)
	}

	return lb
}

// NextTarget은 가중치를 고려하여 다음 대상 서버를 반환합니다.
func (lb *WeightedRoundRobinBalancer) NextTarget() (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.targets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 건강한 대상만 필터링하고 가중치에 따라 확장
	var weightedTargets []string
	for _, target := range lb.targets {
		if target.Healthy {
			// 가중치만큼 URL 반복 추가
			for i := 0; i < target.Weight; i++ {
				weightedTargets = append(weightedTargets, target.URL)
			}
		}
	}

	if len(weightedTargets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 라운드 로빈으로 다음 대상 선택
	position := atomic.AddInt64(&lb.position, 1) % int64(len(weightedTargets))
	selectedURL := weightedTargets[position]

	// 선택된 URL에 해당하는 Target 찾기
	for _, target := range lb.targets {
		if target.URL == selectedURL {
			// 활성 연결 수 증가
			atomic.AddInt64(&target.ActiveConns, 1)
			break
		}
	}

	return selectedURL, nil
}

// AddTarget은 가중치와 함께 새 대상을 추가합니다.
func (lb *WeightedRoundRobinBalancer) AddTarget(urlStr string, weight int) error {
	err := lb.RoundRobinBalancer.AddTarget(urlStr, weight)
	if err != nil {
		return err
	}

	lb.weights[urlStr] = weight
	return nil
}

// LeastConnectionBalancer는 최소 연결 기반 로드 밸런서 구현체입니다.
type LeastConnectionBalancer struct {
	RoundRobinBalancer
}

// NewLeastConnection은 새 최소 연결 기반 로드 밸런서를 생성합니다.
func NewLeastConnection(urls []string) *LeastConnectionBalancer {
	lb := &LeastConnectionBalancer{
		RoundRobinBalancer: RoundRobinBalancer{
			targets:  make([]*Target, 0, len(urls)),
			position: 0,
		},
	}

	// 초기 타겟 추가
	for _, urlStr := range urls {
		lb.AddTarget(urlStr, 1)
	}

	return lb
}

// NextTarget은 가장 적은 활성 연결을 가진 대상 서버를 반환합니다.
func (lb *LeastConnectionBalancer) NextTarget() (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.targets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 건강한 대상만 필터링
	var healthyTargets []*Target
	for _, target := range lb.targets {
		if target.Healthy {
			healthyTargets = append(healthyTargets, target)
		}
	}

	if len(healthyTargets) == 0 {
		return "", ErrNoAvailableTargets
	}

	// 최소 연결을 가진 대상 찾기
	var selectedTarget *Target
	minConn := int64(^uint64(0) >> 1) // 최대 int64 값

	for _, target := range healthyTargets {
		connections := atomic.LoadInt64(&target.ActiveConns)
		if connections < minConn {
			minConn = connections
			selectedTarget = target
		}
	}

	// 활성 연결 수 증가
	if selectedTarget != nil {
		atomic.AddInt64(&selectedTarget.ActiveConns, 1)
		return selectedTarget.URL, nil
	}

	return "", ErrNoAvailableTargets
}

// SingleTarget은 단일 대상을 위한 로드 밸런서 구현체입니다.
type SingleTarget struct {
	target *Target
	mu     sync.RWMutex
}

// NewSingle은 단일 대상 서버를 위한 로드 밸런서를 생성합니다.
func NewSingle(urlStr string) *SingleTarget {
	// URL 유효성 검사
	_, err := url.Parse(urlStr)
	if err != nil {
		// 오류 발생 시 기본 로컬호스트 사용
		urlStr = "http://localhost:8080"
	}

	return &SingleTarget{
		target: &Target{
			URL:          urlStr,
			Healthy:      true,
			LastChecked:  time.Now(),
			FailureCount: 0,
			SuccessCount: 0,
			Weight:       1,
			ActiveConns:  0,
		},
	}
}

// NextTarget은 항상 동일한 대상 서버를 반환합니다.
func (lb *SingleTarget) NextTarget() (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.target == nil || !lb.target.Healthy {
		return "", ErrNoAvailableTargets
	}

	// 활성 연결 수 증가
	atomic.AddInt64(&lb.target.ActiveConns, 1)

	return lb.target.URL, nil
}

// AddTarget은 단일 대상에서는 지원되지 않습니다.
func (lb *SingleTarget) AddTarget(urlStr string, weight int) error {
	return errors.New("single target balancer does not support adding targets")
}

// RemoveTarget은 단일 대상에서는 지원되지 않습니다.
func (lb *SingleTarget) RemoveTarget(urlStr string) error {
	return errors.New("single target balancer does not support removing targets")
}

// MarkTargetDown은 대상 서버를 비정상 상태로 표시합니다.
func (lb *SingleTarget) MarkTargetDown(urlStr string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.target == nil || lb.target.URL != urlStr {
		return errors.New("target not found")
	}

	lb.target.Healthy = false
	lb.target.LastChecked = time.Now()
	lb.target.FailureCount++

	return nil
}

// MarkTargetUp은 대상 서버를 정상 상태로 표시합니다.
func (lb *SingleTarget) MarkTargetUp(urlStr string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.target == nil || lb.target.URL != urlStr {
		return errors.New("target not found")
	}

	lb.target.Healthy = true
	lb.target.LastChecked = time.Now()
	lb.target.SuccessCount++
	lb.target.FailureCount = 0

	return nil
}

// GetTargets는 단일 대상 서버를 반환합니다.
func (lb *SingleTarget) GetTargets() []*Target {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if lb.target == nil {
		return []*Target{}
	}

	// 복사본 생성
	target := *lb.target
	return []*Target{&target}
}

// ReleaseConn은 활성 연결 수를 감소시킵니다.
func ReleaseConn(lb LoadBalancer, urlStr string) {
	targets := lb.GetTargets()
	for _, target := range targets {
		if target.URL == urlStr && target.ActiveConns > 0 {
			atomic.AddInt64(&target.ActiveConns, -1)
			break
		}
	}
}
