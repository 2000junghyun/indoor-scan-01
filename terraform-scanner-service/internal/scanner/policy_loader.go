package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"

	"terraform-scanner-service/internal/types"
)

// PolicyLoader는 Rego 정책을 로드하고 컴파일합니다
type PolicyLoader struct {
	policyDir string
	modules   map[string]*ast.Module
	metadata  map[string]*types.PolicyMetadata
	compiler  *ast.Compiler
}

// NewPolicyLoader는 PolicyLoader를 생성합니다
func NewPolicyLoader(policyDir string) (*PolicyLoader, error) {
	pl := &PolicyLoader{
		policyDir: policyDir,
		modules:   make(map[string]*ast.Module),
		metadata:  make(map[string]*types.PolicyMetadata),
	}

	if err := pl.loadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	if err := pl.compilePolicies(); err != nil {
		return nil, fmt.Errorf("failed to compile policies: %w", err)
	}

	return pl, nil
}

// loadPolicies는 디렉토리에서 .rego 파일을 로드합니다
func (pl *PolicyLoader) loadPolicies() error {
	count := 0

	// 1. lib 디렉토리 로드 (라이브러리 함수들)
	libDir := filepath.Join(pl.policyDir, "../lib")
	if _, err := os.Stat(libDir); err == nil {
		fmt.Println("Loading library functions from lib directory...")
		libCount, err := pl.loadFromDirectory(libDir)
		if err != nil {
			fmt.Printf("Warning: failed to load lib: %v\n", err)
		} else {
			count += libCount
			fmt.Printf("Loaded %d library modules\n", libCount)
		}
	}

	// 2. cloud 디렉토리 로드 (Terraform 관련 정책)
	cloudDir := filepath.Join(pl.policyDir, "cloud")
	if _, err := os.Stat(cloudDir); os.IsNotExist(err) {
		return fmt.Errorf("cloud policy directory not found: %s", cloudDir)
	}

	cloudCount, err := pl.loadFromDirectory(cloudDir)
	if err != nil {
		return err
	}
	count += cloudCount

	if count == 0 {
		return fmt.Errorf("no policy files found")
	}

	fmt.Printf("Loaded %d total policy modules\n", count)
	return nil
}

// loadFromDirectory는 디렉토리에서 .rego 파일을 로드합니다
func (pl *PolicyLoader) loadFromDirectory(dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// .rego 파일만 처리 (_test.rego 제외)
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".rego") &&
			!strings.HasSuffix(d.Name(), "_test.rego") {

			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", path, err)
			}

			// Rego 모듈 파싱
			module, err := ast.ParseModule(path, string(content))
			if err != nil {
				// 파싱 에러는 경고만 하고 계속 진행
				fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
				return nil
			}

			pl.modules[path] = module

			// 메타데이터 추출
			if meta := pl.extractMetadata(module); meta != nil {
				pl.metadata[meta.ID] = meta
			}

			count++
		}
		return nil
	})

	return count, err
}

// compilePolicies는 로드된 정책들을 컴파일합니다
func (pl *PolicyLoader) compilePolicies() error {
	pl.compiler = ast.NewCompiler()
	pl.compiler.Compile(pl.modules)

	if pl.compiler.Failed() {
		// 컴파일 에러가 있는 모듈들을 수집
		failedFiles := make(map[string]bool)
		for _, err := range pl.compiler.Errors {
			if err.Location != nil && err.Location.File != "" {
				failedFiles[err.Location.File] = true
			}
		}

		fmt.Printf("Warning: %d policies failed to compile, removing them...\n", len(failedFiles))

		// 실패한 모듈 및 메타데이터 제거
		for file := range failedFiles {
			delete(pl.modules, file)

			// 해당 모듈의 메타데이터도 제거
			for id, meta := range pl.metadata {
				// metadata는 file path 정보가 없으므로, 성공한 모듈에만 남기도록 처리
				_ = id
				_ = meta
			}
		}

		if len(pl.modules) == 0 {
			return fmt.Errorf("no policies compiled successfully")
		}

		// 재컴파일
		pl.compiler = ast.NewCompiler()
		pl.compiler.Compile(pl.modules)

		if pl.compiler.Failed() {
			// 여전히 실패하면 계속 반복
			fmt.Printf("Warning: still have compilation errors, continuing cleanup...\n")
			return pl.compilePolicies() // 재귀적으로 다시 시도
		}
	}

	// 컴파일 성공 후 메타데이터 재추출
	pl.metadata = make(map[string]*types.PolicyMetadata)
	for path, module := range pl.modules {
		_ = path // path 사용
		if meta := pl.extractMetadata(module); meta != nil {
			pl.metadata[meta.ID] = meta
			fmt.Printf("  Policy: %s (%s) - %s\n", meta.ID, meta.Severity, meta.Title)
		}
	}

	fmt.Printf("Compiled %d policy modules successfully\n", len(pl.modules))
	fmt.Printf("Extracted metadata from %d policies\n", len(pl.metadata))

	// 디버깅: 컴파일된 모듈 목록 출력
	fmt.Println("\nSuccessfully compiled modules:")
	for path := range pl.modules {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Println()

	return nil
}

// extractMetadata는 Rego 모듈에서 메타데이터를 추출합니다
func (pl *PolicyLoader) extractMetadata(module *ast.Module) *types.PolicyMetadata {
	meta := &types.PolicyMetadata{}

	// 어노테이션 확인
	if len(module.Annotations) == 0 {
		// 어노테이션이 없으면 nil 반환
		return nil
	}

	// 어노테이션에서 메타데이터 추출
	for _, annotation := range module.Annotations {
		if annotation.Scope != "package" && annotation.Scope != "rule" {
			continue
		}

		// title
		if title, ok := annotation.Custom["title"].(string); ok {
			meta.Title = title
		}

		// description
		if desc, ok := annotation.Custom["description"].(string); ok {
			meta.Description = desc
		}

		// id / avd_id
		if id, ok := annotation.Custom["id"].(string); ok {
			meta.ID = id
			meta.AVDID = id
		}

		// severity
		if severity, ok := annotation.Custom["severity"].(string); ok {
			meta.Severity = strings.ToUpper(severity)
		}

		// service
		if service, ok := annotation.Custom["service"].(string); ok {
			meta.Service = service
		}

		// provider
		if provider, ok := annotation.Custom["provider"].(string); ok {
			meta.Provider = provider
		}

		// resolution / remediation
		if resolution, ok := annotation.Custom["remediation"].(string); ok {
			meta.Resolution = resolution
		}
		if resolution, ok := annotation.Custom["resolution"].(string); ok {
			meta.Resolution = resolution
		}

		// references
		if refs, ok := annotation.Custom["references"].([]interface{}); ok {
			for _, ref := range refs {
				if refStr, ok := ref.(string); ok {
					meta.References = append(meta.References, refStr)
				}
			}
		}
	}

	// 패키지 이름에서 provider/service 추출
	if module.Package != nil {
		parts := strings.Split(module.Package.Path.String(), ".")
		if len(parts) >= 3 {
			// data.builtin.aws.s3 -> provider: aws, service: s3
			if meta.Provider == "" && len(parts) > 2 {
				meta.Provider = strings.ToUpper(parts[2])
			}
			if meta.Service == "" && len(parts) > 3 {
				meta.Service = parts[3]
			}
		}
	}

	// ID가 없으면 사용하지 않음
	if meta.ID == "" || meta.AVDID == "" {
		return nil
	}

	// 기본값 설정
	if meta.Severity == "" {
		meta.Severity = "MEDIUM"
	}

	return meta
}

// GetCompiler는 컴파일된 정책 컴파일러를 반환합니다
func (pl *PolicyLoader) GetCompiler() *ast.Compiler {
	return pl.compiler
}

// GetModules는 로드된 모듈을 반환합니다
func (pl *PolicyLoader) GetModules() map[string]*ast.Module {
	return pl.modules
}

// GetMetadata는 정책 메타데이터를 반환합니다
func (pl *PolicyLoader) GetMetadata() map[string]*types.PolicyMetadata {
	return pl.metadata
}

// Count는 로드된 정책 수를 반환합니다
func (pl *PolicyLoader) Count() int {
	return len(pl.modules)
}

// PrepareQuery는 Rego 쿼리를 준비합니다
func (pl *PolicyLoader) PrepareQuery(query string, input interface{}) (rego.PreparedEvalQuery, error) {
	r := rego.New(
		rego.Query(query),
		rego.Compiler(pl.compiler),
		rego.Input(input),
	)

	return r.PrepareForEval(nil)
}
