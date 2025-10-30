# Terraform Scanner Service

내장된 정책으로 Terraform 파일을 스캔하는 HTTP 마이크로서비스입니다.

## 특징

- **내장 정책**: trivy-checks 정책을 서비스 시작 시 로드
- **Trivy 비의존적**: 외부 Trivy 실행 파일 없이 독립 실행
- **Trivy 호환 JSON**: Trivy와 동일한 JSON 출력 형식
- **REST API**: HTTP POST로 Terraform 파일 스캔

## 아키텍처

```
┌─────────────────────────────────────────────┐
│  Terraform Scanner Service (:8080)          │
├─────────────────────────────────────────────┤
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │  API Layer (Gin)                     │  │
│  │  - POST /scan                        │  │
│  │  - GET /health                       │  │
│  │  - GET /policies                     │  │
│  └──────────────────────────────────────┘  │
│                   ↓                         │
│  ┌──────────────────────────────────────┐  │
│  │  Scanner Engine                      │  │
│  │  - Terraform Parser (HCL)            │  │
│  │  - State Evaluator                   │  │
│  │  - Rego Policy Engine (OPA)          │  │
│  └──────────────────────────────────────┘  │
│                   ↓                         │
│  ┌──────────────────────────────────────┐  │
│  │  Policy Store (embedded)             │  │
│  │  - trivy-checks/cloud/*.rego         │  │
│  │  - Compiled at startup               │  │
│  └──────────────────────────────────────┘  │
│                   ↓                         │
│  ┌──────────────────────────────────────┐  │
│  │  Result Formatter                    │  │
│  │  - Trivy JSON format                 │  │
│  │  - File storage                      │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## 디렉토리 구조

```
terraform-scanner-service/
├── main.go                    # 서비스 진입점
├── go.mod
├── README.md
├── internal/
│   ├── api/
│   │   └── handler.go        # HTTP 핸들러
│   ├── scanner/
│   │   ├── scanner.go        # 스캔 오케스트레이터
│   │   ├── policy_loader.go  # Rego 정책 로더
│   │   ├── terraform_parser.go # Terraform HCL 파서
│   │   └── rego_engine.go    # OPA Rego 실행 엔진
│   ├── types/
│   │   └── result.go         # Trivy JSON 타입 정의
│   └── utils/
│       └── file.go           # 파일 유틸리티
└── scan-results/             # 스캔 결과 저장
    └── YYYY-MM-DD/
        └── *.json
```

## 사용법

### 1. 서비스 시작

```bash
cd terraform-scanner-service
go mod tidy
go run main.go
```

환경 변수로 정책 디렉토리 지정:
```bash
POLICY_DIR=/path/to/trivy-checks-source/checks go run main.go
```

### 2. Terraform 파일 스캔

**단일 파일 스캔:**
```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./test.tf"}'
```

**디렉토리 스캔:**
```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./terraform-configs/"}'
```

### 3. 헬스 체크

```bash
curl http://localhost:8080/health
```

### 4. 로드된 정책 목록 확인

```bash
curl http://localhost:8080/policies
```

## API 엔드포인트

### POST /scan

Terraform 파일 또는 디렉토리를 스캔합니다.

**Request:**
```json
{
  "target": "/path/to/terraform/file-or-directory"
}
```

**Response (성공):**
```json
{
  "status": "success",
  "results": [
    {
      "file": "scan-results/2025-10-31/main-scan-result.json",
      "target": "main.tf"
    }
  ]
}
```

### GET /health

서비스 상태를 확인합니다.

**Response:**
```json
{
  "status": "healthy",
  "policies_loaded": 150
}
```

### GET /policies

로드된 정책 목록을 반환합니다.

**Response:**
```json
{
  "count": 150,
  "policies": [
    {
      "id": "AVD-AWS-0001",
      "title": "S3 Bucket Encryption",
      "severity": "HIGH"
    }
  ]
}
```

## 출력 형식

스캔 결과는 Trivy와 동일한 JSON 형식으로 저장됩니다:

```json
{
  "SchemaVersion": 2,
  "CreatedAt": "2025-10-31T12:00:00Z",
  "ArtifactName": "main.tf",
  "ArtifactType": "terraform",
  "Results": [
    {
      "Target": "main.tf",
      "Class": "config",
      "Type": "terraform",
      "Misconfigurations": [
        {
          "Type": "Terraform Security Check",
          "ID": "AVD-AWS-0001",
          "AVDID": "AVD-AWS-0001",
          "Title": "S3 Bucket does not have encryption enabled",
          "Description": "...",
          "Message": "Bucket 'my-bucket' does not have encryption enabled",
          "Namespace": "builtin.aws.s3",
          "Query": "data.builtin.aws.s3.deny",
          "Resolution": "Enable encryption for S3 bucket",
          "Severity": "HIGH",
          "PrimaryURL": "https://avd.aquasec.com/...",
          "References": [...],
          "Status": "FAIL",
          "Layer": {},
          "CauseMetadata": {
            "Resource": "aws_s3_bucket.example",
            "Provider": "AWS",
            "Service": "s3",
            "StartLine": 10,
            "EndLine": 15,
            "Code": {
              "Lines": [...]
            }
          }
        }
      ]
    }
  ]
}
```

## 구현 상세

### 스캔 프로세스

1. **정책 로딩** (서비스 시작 시 1회)
   - trivy-checks-source/checks 디렉토리에서 .rego 파일 로드
   - OPA 컴파일러로 정책 컴파일
   - 메타데이터 추출 (AVDID, Severity 등)

2. **Terraform 파싱**
   - HCL2 파서로 .tf 파일 파싱
   - 변수 및 참조 평가
   - Cloud provider별 상태 구조로 변환

3. **정책 실행**
   - OPA Rego 엔진으로 정책 평가
   - deny/warn 규칙 실행
   - 위반 사항 수집

4. **결과 생성**
   - Trivy JSON 형식으로 변환
   - scan-results/YYYY-MM-DD/ 디렉토리에 저장

## 의존성

- **gin-gonic/gin**: HTTP 웹 프레임워크
- **hashicorp/hcl/v2**: Terraform HCL 파서
- **open-policy-agent/opa**: Rego 정책 엔진
- **zclconf/go-cty**: Terraform 타입 시스템

## 제한사항

- Terraform 0.12+ HCL2 문법만 지원
- 원격 모듈은 로컬에 다운로드 필요
- State 파일 스캔 미지원 (HCL 소스만)

## 라이선스

참고용으로만 사용하세요.
