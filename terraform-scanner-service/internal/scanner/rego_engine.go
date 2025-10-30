package scanner

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/rego"

	"terraform-scanner-service/internal/types"
)

// RegoEngine은 Rego 정책을 실행하는 엔진입니다
type RegoEngine struct {
	policyLoader *PolicyLoader
}

// NewRegoEngine은 RegoEngine을 생성합니다
func NewRegoEngine(policyLoader *PolicyLoader) *RegoEngine {
	return &RegoEngine{
		policyLoader: policyLoader,
	}
}

// Scan은 Terraform 데이터를 정책으로 스캔합니다
func (re *RegoEngine) Scan(ctx context.Context, tfData map[string]interface{}, targetPath string) ([]types.Misconfiguration, error) {
	var misconfigs []types.Misconfiguration

	// 모든 정책 모듈을 순회하며 평가
	for modulePath, module := range re.policyLoader.GetModules() {
		// 패키지 경로 추출
		packagePath := module.Package.Path.String()
		if !strings.HasPrefix(packagePath, "data.") {
			continue
		}

		namespace := strings.TrimPrefix(packagePath, "data.")

		// deny, violation, warn 규칙 찾기
		results, err := re.evaluateModule(ctx, namespace, tfData)
		if err != nil {
			fmt.Printf("Warning: failed to evaluate %s: %v\n", modulePath, err)
			continue
		}

		// 결과를 Misconfiguration으로 변환
		for _, result := range results {
			misconfig := re.resultToMisconfiguration(result, namespace, targetPath)
			if misconfig != nil {
				misconfigs = append(misconfigs, *misconfig)
			}
		}
	}

	return misconfigs, nil
}

// evaluateModule은 모듈의 규칙을 평가합니다
func (re *RegoEngine) evaluateModule(ctx context.Context, namespace string, input interface{}) ([]map[string]interface{}, error) {
	var allResults []map[string]interface{}

	// deny 규칙 평가
	denyResults, err := re.evaluateRule(ctx, namespace, "deny", input)
	if err == nil {
		for _, result := range denyResults {
			result["_severity"] = "FAIL"
			allResults = append(allResults, result)
		}
	}

	// violation 규칙 평가
	violationResults, err := re.evaluateRule(ctx, namespace, "violation", input)
	if err == nil {
		for _, result := range violationResults {
			result["_severity"] = "FAIL"
			allResults = append(allResults, result)
		}
	}

	// warn 규칙 평가
	warnResults, err := re.evaluateRule(ctx, namespace, "warn", input)
	if err == nil {
		for _, result := range warnResults {
			result["_severity"] = "WARN"
			allResults = append(allResults, result)
		}
	}

	return allResults, nil
}

// evaluateRule은 특정 규칙을 평가합니다
func (re *RegoEngine) evaluateRule(ctx context.Context, namespace, ruleName string, input interface{}) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("data.%s.%s", namespace, ruleName)

	r := rego.New(
		rego.Query(query),
		rego.Compiler(re.policyLoader.GetCompiler()),
		rego.Input(input),
	)

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for _, result := range rs {
		for _, expr := range result.Expressions {
			if expr.Value == nil {
				continue
			}

			// 결과가 배열이면 각 항목을 추출
			switch v := expr.Value.(type) {
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						results = append(results, m)
					}
				}
			case map[string]interface{}:
				results = append(results, v)
			}
		}
	}

	return results, nil
}

// resultToMisconfiguration은 Rego 결과를 Misconfiguration으로 변환합니다
func (re *RegoEngine) resultToMisconfiguration(result map[string]interface{}, namespace, targetPath string) *types.Misconfiguration {
	// 메타데이터 찾기
	var meta *types.PolicyMetadata
	for _, m := range re.policyLoader.GetMetadata() {
		if strings.Contains(namespace, strings.ToLower(m.Service)) {
			meta = m
			break
		}
	}

	if meta == nil {
		// 메타데이터가 없으면 기본값 사용
		meta = &types.PolicyMetadata{
			ID:       namespace,
			AVDID:    namespace,
			Title:    "Security Check",
			Severity: "MEDIUM",
		}
	}

	// 메시지 추출
	msg, _ := result["msg"].(string)
	if msg == "" {
		msg, _ = result["message"].(string)
	}
	if msg == "" {
		msg = meta.Description
	}

	// 리소스 정보 추출
	resource, _ := result["resource"].(string)

	// 라인 정보 추출
	startLine := 0
	endLine := 0
	if line, ok := result["startline"].(float64); ok {
		startLine = int(line)
	}
	if line, ok := result["endline"].(float64); ok {
		endLine = int(line)
	}

	// Status 결정
	status := "FAIL"
	if severity, ok := result["_severity"].(string); ok && severity == "WARN" {
		status = "WARN"
	}

	misconfig := &types.Misconfiguration{
		Type:        "Terraform Security Check",
		ID:          meta.ID,
		AVDID:       meta.AVDID,
		Title:       meta.Title,
		Description: meta.Description,
		Message:     msg,
		Namespace:   namespace,
		Query:       fmt.Sprintf("data.%s.deny", namespace),
		Resolution:  meta.Resolution,
		Severity:    meta.Severity,
		PrimaryURL:  fmt.Sprintf("https://avd.aquasec.com/misconfig/%s", strings.ToLower(meta.AVDID)),
		References:  meta.References,
		Status:      status,
		Layer:       types.Layer{},
	}

	// CauseMetadata 추가
	if resource != "" || startLine > 0 {
		misconfig.CauseMetadata = &types.CauseMetadata{
			Resource:  resource,
			Provider:  meta.Provider,
			Service:   meta.Service,
			StartLine: startLine,
			EndLine:   endLine,
		}
	}

	return misconfig
}
