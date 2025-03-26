package mocks

import (
    "fmt"
    "net/http"
    "net/http/httptest"

    "github.com/gin-gonic/gin"
    "github.com/phayes/freeport"
)

// MockServer는 모의 백엔드 서비스를 제공합니다
type MockServer struct {
    Server *httptest.Server
    Router *gin.Engine
    Port   int
}

// NewMockServer는 새 모의 서버를 생성합니다
func NewMockServer() (*MockServer, error) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.Use(gin.Recovery())

    // 사용 가능한 포트 가져오기
    port, err := freeport.GetFreePort()
    if err != nil {
        return nil, fmt.Errorf("free port를 가져올 수 없음: %v", err)
    }

    // API 테스트 엔드포인트 등록
    registerMockAPIEndpoints(router)

    server := httptest.NewServer(router)

    return &MockServer{
        Server: server,
        Router: router,
        Port:   port,
    }, nil
}

// Close는 모의 서버를 종료합니다
func (m *MockServer) Close() {
    if m.Server != nil {
        m.Server.Close()
    }
}

// URL은 모의 서버 URL을 반환합니다
func (m *MockServer) URL() string {
    return m.Server.URL
}

// registerMockAPIEndpoints는 모의 API 엔드포인트를 등록합니다
func registerMockAPIEndpoints(router *gin.Engine) {
    // 사용자 API 엔드포인트
    userGroup := router.Group("/api/users")
    {
        userGroup.GET("", func(c *gin.Context) {
            c.JSON(http.StatusOK, gin.H{
                "data": []gin.H{
                    {
                        "id": "1",
                        "username": "testuser1",
                        "email": "test1@example.com",
                        "firstName": "Test",
                        "lastName": "User1",
                    },
                    {
                        "id": "2",
                        "username": "testuser2",
                        "email": "test2@example.com",
                        "firstName": "Test",
                        "lastName": "User2",
                    },
                },
            })
        })

        userGroup.GET("/:id", func(c *gin.Context) {
            id := c.Param("id")
            c.JSON(http.StatusOK, gin.H{
                "id": id,
                "username": "testuser" + id,
                "email": "test" + id + "@example.com",
                "firstName": "Test",
                "lastName": "User" + id,
            })
        })

        userGroup.POST("", func(c *gin.Context) {
            var user struct {
                Username  string `json:"username"`
                Email     string `json:"email"`
                FirstName string `json:"firstName"`
                LastName  string `json:"lastName"`
                Password  string `json:"password"`
            }

            if err := c.ShouldBindJSON(&user); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusCreated, gin.H{
                "id": "3",
                "username": user.Username,
                "email": user.Email,
                "firstName": user.FirstName,
                "lastName": user.LastName,
            })
        })

        userGroup.PUT("/:id", func(c *gin.Context) {
            id := c.Param("id")
            var updates struct {
                FirstName string `json:"firstName"`
                LastName  string `json:"lastName"`
            }

            if err := c.ShouldBindJSON(&updates); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusOK, gin.H{
                "id": id,
                "username": "testuser" + id,
                "email": "test" + id + "@example.com",
                "firstName": updates.FirstName,
                "lastName": updates.LastName,
            })
        })

        userGroup.DELETE("/:id", func(c *gin.Context) {
            c.Status(http.StatusNoContent)
        })
        
        // 프로필 엔드포인트 (인증 필요)
        userGroup.GET("/profile", func(c *gin.Context) {
            authHeader := c.GetHeader("Authorization")
            if authHeader == "" || authHeader != "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QgVXNlciIsImlhdCI6MTUxNjIzOTAyMn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
                return
            }
            
            c.JSON(http.StatusOK, gin.H{
                "id": "1",
                "username": "testuser",
                "email": "test@example.com",
                "firstName": "Test",
                "lastName": "User",
                "roles": []string{"user", "admin"},
            })
        })
    }

    // 인증 API 엔드포인트
    authGroup := router.Group("/api/auth")
    {
        authGroup.POST("/login", func(c *gin.Context) {
            var credentials struct {
                Username string `json:"username"`
                Password string `json:"password"`
            }

            if err := c.ShouldBindJSON(&credentials); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            // 테스트 사용자 확인
            if credentials.Username == "testuser" && credentials.Password == "testpass" {
                c.JSON(http.StatusOK, gin.H{
                    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QgVXNlciIsImlhdCI6MTUxNjIzOTAyMn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
                    "user": gin.H{
                        "id": "1",
                        "username": "testuser",
                        "email": "test@example.com",
                    },
                })
            } else {
                c.JSON(http.StatusUnauthorized, gin.H{
                    "error": "인증 실패",
                })
            }
        })
    }

    // 보고서 API 엔드포인트
    reportGroup := router.Group("/api/reports")
    {
        reportGroup.GET("", func(c *gin.Context) {
            // 쿼리 파라미터 처리
            reportType := c.Query("reportType")
            fromDate := c.Query("fromDate")
            toDate := c.Query("toDate")
            limit := c.DefaultQuery("limit", "10")
            offset := c.DefaultQuery("offset", "0")

            // 쿼리 정보 로깅
            fmt.Printf("Report query: type=%s, from=%s, to=%s, limit=%s, offset=%s\n",
                reportType, fromDate, toDate, limit, offset)

            c.JSON(http.StatusOK, gin.H{
                "data": []gin.H{
                    {
                        "id": "1",
                        "title": "월간 보고서",
                        "description": "3월 월간 보고서",
                        "reportDate": "2025-03-01",
                        "reportType": "MONTHLY",
                        "authorId": "1",
                    },
                    {
                        "id": "2",
                        "title": "분기 보고서",
                        "description": "1분기 보고서",
                        "reportDate": "2025-03-15",
                        "reportType": "QUARTERLY",
                        "authorId": "2",
                    },
                },
                "limit": limit,
                "offset": offset,
                "total": 2,
            })
        })

        reportGroup.GET("/:id", func(c *gin.Context) {
            id := c.Param("id")
            c.JSON(http.StatusOK, gin.H{
                "id": id,
                "title": "보고서 " + id,
                "description": "보고서 " + id + " 설명",
                "reportDate": "2025-03-25",
                "content": "보고서 " + id + "의 내용입니다.",
                "reportType": "TEST",
                "authorId": "1",
            })
        })

        reportGroup.POST("", func(c *gin.Context) {
            var report struct {
                Title       string `json:"title"`
                Description string `json:"description"`
                ReportDate  string `json:"reportDate"`
                Content     string `json:"content"`
                ReportType  string `json:"reportType"`
            }

            if err := c.ShouldBindJSON(&report); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusCreated, gin.H{
                "id": "3",
                "title": report.Title,
                "description": report.Description,
                "reportDate": report.ReportDate,
                "content": report.Content,
                "reportType": report.ReportType,
                "authorId": "1",
            })
        })

        reportGroup.PUT("/:id", func(c *gin.Context) {
            id := c.Param("id")
            var updates struct {
                Title       string `json:"title"`
                Description string `json:"description"`
                Content     string `json:"content"`
            }

            if err := c.ShouldBindJSON(&updates); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusOK, gin.H{
                "id": id,
                "title": updates.Title,
                "description": updates.Description,
                "reportDate": "2025-03-25",
                "content": updates.Content,
                "reportType": "TEST",
                "authorId": "1",
            })
        })

        reportGroup.DELETE("/:id", func(c *gin.Context) {
            c.Status(http.StatusNoContent)
        })

        reportGroup.GET("/statistics", func(c *gin.Context) {
            c.JSON(http.StatusOK, gin.H{
                "totalReports": 10,
                "reportsByType": gin.H{
                    "MONTHLY": 4,
                    "QUARTERLY": 2,
                    "ANNUAL": 1,
                    "TEST": 3,
                },
                "recentReports": 5,
            })
        })
    }

    // 상태 엔드포인트 (공개)
    router.GET("/api/public/status", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "version": "1.0.0",
            "timestamp": "2025-03-26T12:00:00Z",
        })
    })
}
