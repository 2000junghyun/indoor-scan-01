package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// TerraformParser는 Terraform 파일을 파싱합니다
type TerraformParser struct {
	parser *hclparse.Parser
}

// NewTerraformParser는 TerraformParser를 생성합니다
func NewTerraformParser() *TerraformParser {
	return &TerraformParser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile은 단일 Terraform 파일을 파싱합니다
func (tp *TerraformParser) ParseFile(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var file *hcl.File
	var diags hcl.Diagnostics

	if strings.HasSuffix(path, ".json") {
		file, diags = tp.parser.ParseJSON(content, path)
	} else {
		file, diags = tp.parser.ParseHCL(content, path)
	}

	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	// HCL 바디를 맵으로 변환
	result, err := tp.decodeBody(file.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode body: %w", err)
	}

	return result, nil
}

// ParseDirectory는 디렉토리의 모든 .tf 파일을 파싱합니다
func (tp *TerraformParser) ParseDirectory(dir string) (map[string]interface{}, error) {
	merged := make(map[string]interface{})

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tf.json") {
			path := filepath.Join(dir, name)
			result, err := tp.ParseFile(path)
			if err != nil {
				fmt.Printf("Warning: failed to parse %s: %v\n", name, err)
				continue
			}

			// 결과 병합
			tp.mergeResults(merged, result)
		}
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("no valid Terraform files found in %s", dir)
	}

	return merged, nil
}

// decodeBody는 HCL Body를 맵으로 변환합니다
func (tp *TerraformParser) decodeBody(body hcl.Body) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Body의 스키마를 동적으로 분석
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
			{Type: "locals"},
			{Type: "module", LabelNames: []string{"name"}},
			{Type: "provider", LabelNames: []string{"name"}},
			{Type: "terraform"},
		},
		Attributes: []hcl.AttributeSchema{},
	}

	content, _, diags := body.PartialContent(schema)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to get content: %s", diags.Error())
	}

	// 블록 처리
	for _, block := range content.Blocks {
		blockData := tp.processBlock(block)

		switch block.Type {
		case "resource":
			if _, ok := result["resource"]; !ok {
				result["resource"] = make(map[string]interface{})
			}
			resources := result["resource"].(map[string]interface{})

			resourceType := block.Labels[0]
			resourceName := block.Labels[1]

			if _, ok := resources[resourceType]; !ok {
				resources[resourceType] = make(map[string]interface{})
			}
			resources[resourceType].(map[string]interface{})[resourceName] = blockData

		case "data":
			if _, ok := result["data"]; !ok {
				result["data"] = make(map[string]interface{})
			}
			dataResources := result["data"].(map[string]interface{})

			dataType := block.Labels[0]
			dataName := block.Labels[1]

			if _, ok := dataResources[dataType]; !ok {
				dataResources[dataType] = make(map[string]interface{})
			}
			dataResources[dataType].(map[string]interface{})[dataName] = blockData

		case "variable":
			if _, ok := result["variable"]; !ok {
				result["variable"] = make(map[string]interface{})
			}
			result["variable"].(map[string]interface{})[block.Labels[0]] = blockData

		case "output":
			if _, ok := result["output"]; !ok {
				result["output"] = make(map[string]interface{})
			}
			result["output"].(map[string]interface{})[block.Labels[0]] = blockData

		case "module":
			if _, ok := result["module"]; !ok {
				result["module"] = make(map[string]interface{})
			}
			result["module"].(map[string]interface{})[block.Labels[0]] = blockData

		case "provider":
			if _, ok := result["provider"]; !ok {
				result["provider"] = make(map[string]interface{})
			}
			result["provider"].(map[string]interface{})[block.Labels[0]] = blockData

		case "terraform":
			result["terraform"] = blockData

		case "locals":
			result["locals"] = blockData
		}
	}

	return result, nil
}

// processBlock은 블록을 재귀적으로 처리합니다
func (tp *TerraformParser) processBlock(block *hcl.Block) map[string]interface{} {
	result := make(map[string]interface{})

	// 속성 처리
	attrs, diags := block.Body.JustAttributes()
	if !diags.HasErrors() {
		for name, attr := range attrs {
			val, err := tp.evalAttribute(attr)
			if err == nil {
				result[name] = val
			} else {
				// 평가 실패 시 원본 표현식 저장
				result[name] = string(attr.Expr.Range().SliceBytes(attr.Expr.Range().SliceBytes(nil)))
			}
		}
	}

	// 중첩 블록 처리
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "*"},
		},
	}
	content, _, _ := block.Body.PartialContent(schema)

	for _, nestedBlock := range content.Blocks {
		nestedData := tp.processBlock(nestedBlock)

		// 같은 타입의 블록이 여러 개면 배열로
		if existing, ok := result[nestedBlock.Type]; ok {
			if arr, isArray := existing.([]interface{}); isArray {
				result[nestedBlock.Type] = append(arr, nestedData)
			} else {
				result[nestedBlock.Type] = []interface{}{existing, nestedData}
			}
		} else {
			result[nestedBlock.Type] = nestedData
		}
	}

	return result
}

// evalAttribute는 속성 값을 평가합니다
func (tp *TerraformParser) evalAttribute(attr *hcl.Attribute) (interface{}, error) {
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return nil, fmt.Errorf("evaluation error: %s", diags.Error())
	}

	return tp.ctyToGo(val), nil
}

// ctyToGo는 cty.Value를 Go 타입으로 변환합니다
func (tp *TerraformParser) ctyToGo(val cty.Value) interface{} {
	if val.IsNull() {
		return nil
	}

	ty := val.Type()

	switch {
	case ty == cty.String:
		return val.AsString()
	case ty == cty.Number:
		f, _ := val.AsBigFloat().Float64()
		return f
	case ty == cty.Bool:
		return val.True()
	case ty.IsListType() || ty.IsSetType() || ty.IsTupleType():
		var result []interface{}
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			result = append(result, tp.ctyToGo(v))
		}
		return result
	case ty.IsMapType() || ty.IsObjectType():
		result := make(map[string]interface{})
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			result[k.AsString()] = tp.ctyToGo(v)
		}
		return result
	default:
		return val.GoString()
	}
}

// mergeResults는 두 결과를 병합합니다
func (tp *TerraformParser) mergeResults(dst, src map[string]interface{}) {
	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// 둘 다 맵이면 재귀적으로 병합
			if dstMap, dstOk := dstVal.(map[string]interface{}); dstOk {
				if srcMap, srcOk := srcVal.(map[string]interface{}); srcOk {
					tp.mergeResults(dstMap, srcMap)
					continue
				}
			}
		}
		dst[key] = srcVal
	}
}
