# Terraform Scanner Service - Architecture

## 시스템 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│                     Terraform Scanner Service                    │
└─────────────────────────────────────────────────────────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
            ┌───────▼──────┐ ┌──▼──────┐ ┌──▼──────────┐
            │   API Layer  │ │ Scanner │ │Policy Store │
            │    (Gin)     │ │ Engine  │ │  (Rego)     │
            └──────────────┘ └─────────┘ └─────────────┘
```

## 컴포넌트 상세

### 1. API Layer (`internal/api/handler.go`)

HTTP 요청을 처리하는 레이어입니다.

**엔드포인트:**
- `POST /scan`: Terraform 파일/디렉토리 스캔
- `GET /health`: 서비스 상태 확인
- `GET /policies`: 로드된 정책 목록

**책임:**
- 요청 검증
- 스캔 오케스트레이션
- 결과 저장
- 응답 생성

### 2. Scanner Engine (`internal/scanner/`)

Terraform 파일을 분석하고 정책을 적용하는 핵심 엔진입니다.

#### 2.1 TerraformScanner (`scanner.go`)
메인 스캐너 오케스트레이터

**기능:**
- 파일/디렉토리 스캔 조율
- 결과 집계
- 정책 통계 제공

#### 2.2 PolicyLoader (`policy_loader.go`)
Rego 정책 로더

**기능:**
- .rego 파일 로드
- 정책 컴파일 (OPA)
- 메타데이터 추출

**로딩 프로세스:**
```
trivy-checks-source/checks/cloud/
  ├── aws/
  │   ├── s3/*.rego
  │   ├── ec2/*.rego
  │   └── ...
  ├── azure/
  └── google/
       ↓
  파일 스캔 (.rego)
       ↓
  AST 파싱 (ast.ParseModule)
       ↓
  메타데이터 추출 (annotations)
       ↓
  컴파일 (ast.Compiler)
       ↓
  메모리 저장 (map[string]*ast.Module)
```

#### 2.3 TerraformParser (`terraform_parser.go`)
HCL 파서

**기능:**
- .tf 파일 파싱
- HCL → Go map 변환
- 변수/참조 평가

**파싱 프로세스:**
```
.tf 파일
   ↓
HCL 파싱 (hclparse)
   ↓
AST 생성
   ↓
Body 디코딩
   ↓
cty.Value → Go map
   ↓
{
  "resource": {
    "aws_s3_bucket": {
      "example": {...}
    }
  }
}
```

#### 2.4 RegoEngine (`rego_engine.go`)
Rego 정책 실행 엔진

**기능:**
- 정책 평가
- deny/warn 규칙 실행
- 결과 변환

**평가 프로세스:**
```
Terraform 데이터 (map)
   ↓
각 정책 모듈 순회
   ↓
Rego 쿼리 실행
  - data.<namespace>.deny
  - data.<namespace>.warn
   ↓
결과 집계
   ↓
Misconfiguration 변환
```

### 3. Types (`internal/types/result.go`)

Trivy 호환 데이터 구조 정의

**주요 타입:**
- `ScanResult`: 최상위 결과
- `Result`: 타겟별 결과
- `Misconfiguration`: 개별 보안 문제
- `PolicyMetadata`: 정책 메타데이터

## 데이터 흐름

### 스캔 요청 플로우

```
1. HTTP POST /scan
   └─> handler.ScanTerraform()
       │
       ├─> scanner.ScanTarget()
       │   │
       │   ├─> parser.ParseFile() / ParseDirectory()
       │   │   └─> HCL → Go map
       │   │
       │   └─> regoEngine.Scan()
       │       │
       │       ├─> evaluateModule() (각 정책)
       │       │   ├─> evaluateRule("deny")
       │       │   ├─> evaluateRule("warn")
       │       │   └─> OPA Eval
       │       │
       │       └─> resultToMisconfiguration()
       │
       └─> saveResults()
           └─> JSON 파일 저장
```

### 정책 평가 상세

```
Input (Terraform 데이터):
{
  "resource": {
    "aws_s3_bucket": {
      "example": {
        "bucket": "my-bucket",
        "encryption": null  // ❌ 문제!
      }
    }
  }
}
    ↓
Rego 정책 (예: AWS S3 암호화):
package builtin.aws.s3

deny[msg] {
  bucket := input.resource.aws_s3_bucket[_]
  not bucket.encryption
  msg := {"msg": "암호화 미설정"}
}
    ↓
평가 결과:
[
  {
    "msg": "암호화 미설정",
    "resource": "aws_s3_bucket.example",
    "_severity": "FAIL"
  }
]
    ↓
Misconfiguration:
{
  "ID": "AVD-AWS-0088",
  "Title": "S3 암호화 필요",
  "Severity": "HIGH",
  "Status": "FAIL"
}
```

## 확장 가능성

### 새로운 정책 추가

1. `trivy-checks-source/checks/cloud/` 에 .rego 파일 추가
2. 서비스 재시작 → 자동 로드

### 새로운 IaC 지원 (예: CloudFormation)

1. `internal/scanner/cloudformation_parser.go` 추가
2. `scanner.go`에 분기 추가:
```go
if strings.HasSuffix(path, ".yaml") {
    return cfParser.ParseFile(path)
}
```

### 결과 포맷 추가

1. `internal/formatter/` 디렉토리 생성
2. `sarif_formatter.go`, `html_formatter.go` 등 구현

## 성능 고려사항

### 최적화 포인트

1. **정책 로딩**: 서비스 시작 시 1회만 (재사용)
2. **컴파일 캐싱**: OPA 컴파일러 결과 메모리 유지
3. **병렬 처리**: 디렉토리 스캔 시 고루틴 활용 가능
4. **결과 스트리밍**: 큰 디렉토리는 스트리밍 응답

### 메모리 사용

- 정책 로딩: ~50MB (150개 정책 기준)
- 파싱 버퍼: ~10MB per file
- OPA 평가: ~20MB

### 스캔 속도

- 단일 파일: ~100-500ms
- 디렉토리 (10개 파일): ~1-3s
- 대규모 (100개 파일): ~10-30s

## 보안 고려사항

### 입력 검증

- 파일 경로 검증 (path traversal 방지)
- 파일 크기 제한
- 타임아웃 설정

### 격리

- Docker 컨테이너 실행 권장
- 읽기 전용 정책 마운트
- 결과 디렉토리만 쓰기 권한

## 모니터링

### 로깅

```go
fmt.Printf("Loaded %d policy modules\n", count)
fmt.Printf("Scan result saved: %s\n", resultFile)
```

### 메트릭

- 스캔 횟수
- 평균 응답 시간
- 발견된 misconfiguration 수
- 정책 로드 성공률

## 트러블슈팅

### 일반적인 문제

1. **정책 로드 실패**
   - 경로 확인: `POLICY_DIR` 환경 변수
   - 권한 확인: 읽기 권한 필요

2. **파싱 에러**
   - HCL 문법 오류
   - Terraform 버전 호환성

3. **OPA 컴파일 에러**
   - Rego 문법 오류
   - 의존성 누락

### 디버깅

```bash
# 상세 로그 활성화
DEBUG=true go run main.go

# 특정 파일만 테스트
curl -X POST http://localhost:8080/scan \
  -d '{"target": "./problematic-file.tf"}'
```
