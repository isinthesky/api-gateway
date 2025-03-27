// +build integration

package integration

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

const (
	DefaultGatewayURL = "http://localhost:8080"
	MaxWaitTime       = 30 * time.Second
	CheckInterval     = 1 * time.Second
)

var (
	gatewayProcess *os.Process
	gatewayURL     string
)

// SetupIntegrationTests는 통합 테스트 환경을 설정합니다.
func SetupIntegrationTests() error {
	// 환경 변수에서 설정 가져오기
	gatewayURL = os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = DefaultGatewayURL
	}

	// 이미 실행 중인 API Gateway가 있는지 확인
	if isServiceRunning(gatewayURL) {
		log.Println("기존 API Gateway 인스턴스 사용:", gatewayURL)
		return nil
	}

	// 백그라운드에서 API Gateway 실행
	log.Println("API Gateway 시작 중...")
	cmd := exec.Command("../build/api-gateway")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("API Gateway 시작 실패: %v", err)
	}
	
	gatewayProcess = cmd.Process
	log.Println("API Gateway 프로세스 시작됨. PID:", gatewayProcess.Pid)
	
	// API Gateway가 준비될 때까지 대기
	if err := waitForService(gatewayURL, MaxWaitTime); err != nil {
		// 시작 실패 시 프로세스 종료
		gatewayProcess.Kill()
		return err
	}
	
	log.Println("API Gateway 시작 완료:", gatewayURL)
	return nil
}

// TeardownIntegrationTests는 통합 테스트 환경을 정리합니다.
func TeardownIntegrationTests() {
	if gatewayProcess != nil {
		log.Println("API Gateway 종료 중...")
		
		// SIGTERM 시그널 전송
		if err := gatewayProcess.Signal(syscall.SIGTERM); err != nil {
			log.Printf("SIGTERM 시그널 전송 실패: %v", err)
			// 강제 종료 시도
			if err := gatewayProcess.Kill(); err != nil {
				log.Printf("API Gateway 프로세스 강제 종료 실패: %v", err)
			}
		}
		
		// 프로세스 상태 수집
		if _, err := gatewayProcess.Wait(); err != nil {
			log.Printf("API Gateway 프로세스 대기 실패: %v", err)
		}
		
		log.Println("API Gateway 종료됨")
	}
}

// TestMain은 모든 테스트의 설정과 정리를 관리합니다.
func TestMain(m *testing.M) {
	// 통합 테스트 환경 설정
	if err := SetupIntegrationTests(); err != nil {
		log.Printf("통합 테스트 환경 설정 실패: %v", err)
		os.Exit(1)
	}
	
	// 테스트 실행
	exitCode := m.Run()
	
	// 통합 테스트 환경 정리
	TeardownIntegrationTests()
	
	os.Exit(exitCode)
}

// isServiceRunning은 서비스가 실행 중인지 확인합니다.
func isServiceRunning(url string) bool {
	resp, err := http.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// waitForService는 서비스가 준비될 때까지 대기합니다.
func waitForService(url string, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("서비스 대기 시간 초과 (%s)", url)
		}
		
		if isServiceRunning(url) {
			return nil
		}
		
		// 잠시 대기 후 재시도
		time.Sleep(CheckInterval)
	}
}
