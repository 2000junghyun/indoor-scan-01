package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"terraform-scanner-service/internal/scanner"
	"terraform-scanner-service/internal/types"
)

// Handler는 HTTP 요청을 처리합니다
type Handler struct {
	scanner *scanner.TerraformScanner
}

// NewHandler는 Handler를 생성합니다
func NewHandler(scanner *scanner.TerraformScanner) *Handler {
	return &Handler{
		scanner: scanner,
	}
}

// ScanRequest는 스캔 요청 구조입니다
type ScanRequest struct {
	Target string `json:"target" binding:"required"`
}

// ScanResponse는 스캔 응답 구조입니다
type ScanResponse struct {
	Status  string       `json:"status"`
	Results []ScanResult `json:"results,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// ScanResult는 개별 스캔 결과입니다
type ScanResult struct {
	File   string `json:"file"`
	Target string `json:"target"`
}

// ScanTerraform은 Terraform 파일을 스캔합니다
func (h *Handler) ScanTerraform(c *gin.Context) {
	var targetPath string

	// multipart file upload 처리
	if file, err := c.FormFile("file"); err == nil {
		// 임시 디렉토리에 파일 저장
		tempDir := os.TempDir()
		tempFile := filepath.Join(tempDir, file.Filename)

		if err := c.SaveUploadedFile(file, tempFile); err != nil {
			c.JSON(http.StatusInternalServerError, ScanResponse{
				Status: "error",
				Error:  "failed to save uploaded file: " + err.Error(),
			})
			return
		}

		// 스캔 후 임시 파일 삭제
		defer os.Remove(tempFile)
		targetPath = tempFile
	} else {
		// JSON 요청 처리
		var req ScanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ScanResponse{
				Status: "error",
				Error:  "invalid request: " + err.Error(),
			})
			return
		}
		targetPath = req.Target

		// 타겟 경로 확인
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, ScanResponse{
				Status: "error",
				Error:  "target path not found: " + targetPath,
			})
			return
		}
	}

	// 스캔 실행
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := h.scanner.ScanTarget(ctx, targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScanResponse{
			Status: "error",
			Error:  "scan failed: " + err.Error(),
		})
		return
	}

	// 결과 저장
	savedFiles, err := h.saveResults(results)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScanResponse{
			Status: "error",
			Error:  "failed to save results: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ScanResponse{
		Status:  "success",
		Results: savedFiles,
	})
}

// saveResults는 스캔 결과를 JSON 파일로 저장합니다
func (h *Handler) saveResults(results []*types.ScanResult) ([]ScanResult, error) {
	// 저장 디렉토리 생성
	baseDir := "scan-results"
	dateDir := time.Now().Format("2006-01-02")
	saveDir := filepath.Join(baseDir, dateDir)

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var savedFiles []ScanResult

	for _, result := range results {
		// 파일명 생성
		fileName := strings.TrimSuffix(result.ArtifactName, filepath.Ext(result.ArtifactName))
		resultFile := filepath.Join(saveDir, fmt.Sprintf("%s-scan-result.json", fileName))

		// JSON으로 직렬화
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		// 파일 저장
		if err := os.WriteFile(resultFile, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		savedFiles = append(savedFiles, ScanResult{
			File:   resultFile,
			Target: result.ArtifactName,
		})

		fmt.Printf("Scan result saved: %s\n", resultFile)
	}

	return savedFiles, nil
}

// HealthCheck는 서비스 상태를 확인합니다
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"policies_loaded": h.scanner.PolicyCount(),
		"timestamp":       time.Now().Format(time.RFC3339),
	})
}

// ListPolicies는 로드된 정책 목록을 반환합니다
func (h *Handler) ListPolicies(c *gin.Context) {
	policies := h.scanner.GetPolicies()

	c.JSON(http.StatusOK, gin.H{
		"count":    len(policies),
		"policies": policies,
	})
}
