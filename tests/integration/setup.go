package integration

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"

    "github.com/isinthesky/api-gateway/config"
    "github.com/isinthesky/api-gateway/tests/mocks"
    "github.com/isinthesky/api-gateway/tests/utils"
)

// TestSetup은 통합 테스트를 위한 환경을 설정합니다
type TestSetup struct {
    Config      *config.Config
    APIGateway  *exec.Cmd
    MockServers []*mocks.MockServer
    Port        int
    TempDir     string
    Logger      *utils.TestLogger
}

// NewTestSetup은 통합 테스트 환경을 생성합니다
func NewTestSetup(t *testing.T) *TestSetup {
    logger := utils.NewTestLogger(utils.LevelDebug)
    logger.Info("테스트 환경 설정 시작")
    
    // 임시 디렉토리 생성
    tempDir, err := ioutil.TempDir("", "api-gateway-test")
    if err != nil {
        t.Fatalf("임시 디렉토리 생성 실패: %v", err)
    }
    
    // 테스트용 포트 설정
    port := 18080
    
    // 환경 변수 설정
    os.Setenv("PORT", fmt.Sprintf("%d", port))
    os.Setenv("LOG_LEVEL", "debug")
    os.Setenv("TEST_ENV", "true")
    
    // 모의 서버 시작
    logger.Info("모의 서버 시작")
    mockServer1, err := mocks.NewMockServer()
    if err != nil {
        t.Fatalf("모의 서버 1 시작 실패: %v", err)
    }
    
    mockServer2, err := mocks.NewMockServer()
    if err != nil {
        mockServer1.Close()
        t.Fatalf("모의 서버 2 시작 실패: %v", err)
    }
    
    logger.Info("모의 서버 1 URL: %s", mockServer1.URL())
    logger.Info("모의 서버 2 URL: %s", mockServer2.URL())
    
    // 테스트용 라우트 구성 생성
    routesConfig := createTestRoutesConfig(mockServer1.URL(), mockServer2.URL())
    routesConfigPath := filepath.Join(tempDir, "routes.json")
    
    routesJSON, err := json.MarshalIndent(routesConfig, "", "  ")
    if err != nil {
        mockServer1.Close()
        mockServer2.Close()
        os.RemoveAll(tempDir)
        t.Fatalf("라우트 구성 마샬링 실패: %v", err)
    }
    
    if err := ioutil.WriteFile(routesConfigPath, routesJSON, 0644); err != nil {
        mockServer1.Close()
        mockServer2.Close()
        os.RemoveAll(tempDir)
        t.Fatalf("라우트 구성 파일 쓰기 실패: %v", err)
    }
    
    // 환경 변수로 라우트 구성 경로 설정
    os.Setenv("ROUTES_CONFIG_PATH", routesConfigPath)
    
    // API 게이트웨이 설정
    cfg := config.LoadConfig()
    
    // API 게이트웨이 시작 (실제 구현에서는 별도 프로세스로 실행하거나 코드에서 직접 시작)
    // 여기서는 예시로 별도 프로세스 실행을 보여줍니다
    logger.Info("API 게이트웨이 시작")
    
    /*
    // 주석: 실제 API 게이트웨이 서버를 별도 프로세스로 실행하는 코드 (필요시 주석 해제)
    cmd := exec.Command("go", "run", "main.go")
    cmd.Env = os.Environ()
    if err := cmd.Start(); err != nil {
        mockServer1.Close()
        mockServer2.Close()
        os.RemoveAll(tempDir)
        t.Fatalf("API 게이트웨이 시작 실패: %v", err)
    }
    */
    
    // 서버가 시작될 때까지 잠시 대기
    time.Sleep(500 * time.Millisecond)
    
    setup := &TestSetup{
        Config:      cfg,
        APIGateway:  nil, // 실제 실행 시 cmd 할당
        MockServers: []*mocks.MockServer{mockServer1, mockServer2},
        Port:        port,
        TempDir:     tempDir,
        Logger:      logger,
    }
    
    // 게이트웨이 준비 확인
    if err := setup.waitForGateway(5 * time.Second); err != nil {
        setup.Cleanup()
        t.Fatalf("API 게이트웨이 준비 대기 실패: %v", err)
    }
    
    logger.Info("테스트 환경 설정 완료")
    return setup
}

// Cleanup은 테스트 환경을 정리합니다
func (s *TestSetup) Cleanup() {
    s.Logger.Info("테스트 환경 정리 시작")
    
    // 모의 서버 종료
    for _, server := range s.MockServers {
        server.Close()
    }
    
    // API 게이트웨이 종료
    if s.APIGateway != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        
        if err := s.APIGateway.Process.Kill(); err != nil {
            s.Logger.Error("API 게이트웨이 프로세스 종료 실패: %v", err)
        }
    }
    
    // 임시 디렉토리 삭제
    if s.TempDir != "" {
        os.RemoveAll(s.TempDir)
    }
    
    // 환경 변수 초기화
    os.Unsetenv("PORT")
    os.Unsetenv("LOG_LEVEL")
    os.Unsetenv("TEST_ENV")
    os.Unsetenv("ROUTES_CONFIG_PATH")
    
    s.Logger.Info("테스트 환경 정리 완료")
}

// waitForGateway는 API 게이트웨이가 준비될 때까지 대기합니다
func (s *TestSetup) waitForGateway(timeout time.Duration) error {
    s.Logger.Info("API 게이트웨이 준비 대기 중...")
    
    client := &http.Client{
        Timeout: 1 * time.Second,
    }
    
    url := fmt.Sprintf("http://localhost:%d/health", s.Port)
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        resp, err := client.Get(url)
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            s.Logger.Info("API 게이트웨이 준비됨")
            return nil
        }
        
        if resp != nil {
            resp.Body.Close()
        }
        
        time.Sleep(100 * time.Millisecond)
    }
    
    return fmt.Errorf("API 게이트웨이가 %s 내에 준비되지 않음", timeout)
}

// createTestRoutesConfig는 테스트용 라우트 구성을 생성합니다
func createTestRoutesConfig(mockServer1URL, mockServer2URL string) map[string]interface{} {
    return map[string]interface{}{
        "routes": []map[string]interface{}{
            {
                "path":        "/health",
                "targetURL":   "local", // 내부 처리
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/users",
                "targetURL":   mockServer1URL + "/api/users",
                "methods":     []string{"GET", "POST"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/users/:id",
                "targetURL":   mockServer1URL + "/api/users/:id",
                "methods":     []string{"GET", "PUT", "DELETE"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/users/profile",
                "targetURL":   mockServer1URL + "/api/users/profile",
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/auth/login",
                "targetURL":   mockServer1URL + "/api/auth/login",
                "methods":     []string{"POST"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/reports",
                "targetURL":   mockServer2URL + "/api/reports",
                "methods":     []string{"GET", "POST"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/reports/:id",
                "targetURL":   mockServer2URL + "/api/reports/:id",
                "methods":     []string{"GET", "PUT", "DELETE"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/reports/statistics",
                "targetURL":   mockServer2URL + "/api/reports/statistics",
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": true,
            },
            {
                "path":        "/api/public/status",
                "targetURL":   mockServer1URL + "/api/public/status",
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/fault-test/intermittent-failure",
                "targetURL":   "http://localhost:0/api/intermittent-failure", // 실제 테스트 시 FaultServer URL로 교체
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/fault-test/slow-response",
                "targetURL":   "http://localhost:0/api/slow-response", // 실제 테스트 시 FaultServer URL로 교체
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/fault-test/service-down",
                "targetURL":   "http://localhost:0/api/service-down", // 실제 테스트 시 FaultServer URL로 교체
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/circuit-test/failing-endpoint",
                "targetURL":   "http://localhost:0/api/circuit-test/failing-endpoint", // 실제 테스트 시 FaultServer URL로 교체
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
            {
                "path":        "/api/circuit-test/recovered-endpoint",
                "targetURL":   "http://localhost:0/api/circuit-test/recovered-endpoint", // 실제 테스트 시 FaultServer URL로 교체
                "methods":     []string{"GET"},
                "stripPrefix": "",
                "requireAuth": false,
            },
        },
    }
}

// TestConfig는 테스트 구성을 정의합니다
type TestConfig struct {
    LogLevel    int    // 로그 레벨
    LogOutput   string // 로그 출력 (stdout, stderr, file)
    LogFilePath string // 파일로 로깅할 경우 경로
}

// DefaultTestConfig는 기본 테스트 구성을 반환합니다
func DefaultTestConfig() TestConfig {
    return TestConfig{
        LogLevel:    utils.LevelInfo,
        LogOutput:   "stdout",
        LogFilePath: "",
    }
}

// SetupTestLogging은 테스트 로깅을 설정합니다
func SetupTestLogging(config TestConfig) *utils.TestLogger {
    var logger *utils.TestLogger
    
    if config.LogOutput == "file" && config.LogFilePath != "" {
        file, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            fmt.Printf("로그 파일 열기 실패: %v\n", err)
            os.Exit(1)
        }
        
        // 여기서는 간단한 구현으로 표준 로거 사용
        logger = utils.NewTestLogger(config.LogLevel)
    } else {
        logger = utils.NewTestLogger(config.LogLevel)
    }
    
    return logger
}
