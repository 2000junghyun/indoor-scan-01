package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"terraform-scanner-service/internal/types"
)

// TerraformScanner는 Terraform 파일을 스캔하는 메인 스캐너입니다
type TerraformScanner struct {
	policyLoader *PolicyLoader
	parser       *TerraformParser
	regoEngine   *RegoEngine
}

// NewTerraformScanner는 TerraformScanner를 생성합니다
func NewTerraformScanner(policyDir string) (*TerraformScanner, error) {
	// 정책 로더 초기화
	policyLoader, err := NewPolicyLoader(policyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize policy loader: %w", err)
	}

	// 파서 초기화
	parser := NewTerraformParser()

	// Rego 엔진 초기화
	regoEngine := NewRegoEngine(policyLoader)

	return &TerraformScanner{
		policyLoader: policyLoader,
		parser:       parser,
		regoEngine:   regoEngine,
	}, nil
}

// ScanFile은 단일 Terraform 파일을 스캔합니다
func (ts *TerraformScanner) ScanFile(ctx context.Context, path string) (*types.ScanResult, error) {
	// 파일 파싱
	tfData, err := ts.parser.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Rego 정책으로 스캔
	misconfigs, err := ts.regoEngine.Scan(ctx, tfData, path)
	if err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	// 결과 구성
	result := &types.ScanResult{
		SchemaVersion: 2,
		CreatedAt:     time.Now(),
		ArtifactName:  filepath.Base(path),
		ArtifactType:  "terraform",
		Results: []types.Result{
			{
				Target:            filepath.Base(path),
				Class:             "config",
				Type:              "terraform",
				Misconfigurations: misconfigs,
			},
		},
	}

	return result, nil
}

// ScanDirectory는 디렉토리의 모든 .tf 파일을 스캔합니다
func (ts *TerraformScanner) ScanDirectory(ctx context.Context, dir string) ([]*types.ScanResult, error) {
	var results []*types.ScanResult

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) == ".tf" || filepath.Ext(name) == ".tfvars" {
			path := filepath.Join(dir, name)

			result, err := ts.ScanFile(ctx, path)
			if err != nil {
				fmt.Printf("Warning: failed to scan %s: %v\n", name, err)
				continue
			}

			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no Terraform files found in %s", dir)
	}

	return results, nil
}

// ScanTarget은 파일 또는 디렉토리를 스캔합니다
func (ts *TerraformScanner) ScanTarget(ctx context.Context, target string) ([]*types.ScanResult, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("target not found: %w", err)
	}

	if info.IsDir() {
		return ts.ScanDirectory(ctx, target)
	}

	result, err := ts.ScanFile(ctx, target)
	if err != nil {
		return nil, err
	}

	return []*types.ScanResult{result}, nil
}

// PolicyCount는 로드된 정책 수를 반환합니다
func (ts *TerraformScanner) PolicyCount() int {
	return ts.policyLoader.Count()
}

// GetPolicies는 로드된 정책의 메타데이터를 반환합니다
func (ts *TerraformScanner) GetPolicies() []*types.PolicyMetadata {
	metadata := ts.policyLoader.GetMetadata()
	policies := make([]*types.PolicyMetadata, 0, len(metadata))

	for _, meta := range metadata {
		policies = append(policies, meta)
	}

	return policies
}
