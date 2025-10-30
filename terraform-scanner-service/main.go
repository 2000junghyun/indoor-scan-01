package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"terraform-scanner-service/internal/api"
	"terraform-scanner-service/internal/scanner"

	"github.com/gin-gonic/gin"
)

func main() {
	// 정책 디렉토리 확인
	policyDir := os.Getenv("POLICY_DIR")
	if policyDir == "" {
		policyDir = "../trivy-checks-source/checks"
	}

	// Scanner 초기화
	log.Println("Initializing Terraform scanner...")
	tfScanner, err := scanner.NewTerraformScanner(policyDir)
	if err != nil {
		log.Fatalf("Failed to initialize scanner: %v", err)
	}
	log.Printf("Scanner initialized with %d policies\n", tfScanner.PolicyCount())

	// Gin 라우터 설정
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// API 핸들러 등록
	handler := api.NewHandler(tfScanner)
	router.POST("/scan", handler.ScanTerraform)
	router.GET("/health", handler.HealthCheck)
	router.GET("/policies", handler.ListPolicies)

	// HTTP 서버 설정
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 서버 시작 (고루틴)
	go func() {
		log.Println("Starting Terraform Scanner API on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
