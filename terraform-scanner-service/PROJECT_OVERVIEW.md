# Terraform Scanner Service

독립적으로 실행되는 Terraform 보안 스캐너 마이크로서비스입니다. Trivy의 실행 파일 없이 내장된 정책으로 Terraform 파일을 분석합니다.

## 🎯 프로젝트 목표

- **독립 실행**: Trivy 실행 파일에 의존하지 않는 독립 서비스
- **내장 정책**: trivy-checks 정책을 서비스에 내장
- **Trivy 호환**: Trivy와 동일한 JSON 출력 형식
- **HTTP API**: REST API를 통한 스캔 요청/결과 조회

## 📁 프로젝트 구조

```
terraform-scanner-service/
├── main.go                      # 서비스 진입점
├── go.mod                       # Go 모듈 정의
├── README.md                    # 이 파일
├── QUICKSTART.md               # 빠른 시작 가이드
├── ARCHITECTURE.md             # 아키텍처 문서
├── Dockerfile                  # Docker 이미지 빌드
├── docker-compose.yml          # Docker Compose 설정
├── build.sh                    # 빌드 스크립트
├── test.sh                     # 테스트 스크립트
├── .gitignore
│
├── internal/
│   ├── api/
│   │   └── handler.go          # HTTP 요청 핸들러
│   ├── scanner/
│   │   ├── scanner.go          # 메인 스캐너
│   │   ├── policy_loader.go    # Rego 정책 로더
│   │   ├── terraform_parser.go # Terraform HCL 파서
│   │   └── rego_engine.go      # OPA Rego 엔진
│   ├── types/
│   │   └── result.go           # 결과 타입 정의
│   └── utils/
│       └── file.go             # 파일 유틸리티
│
├── examples/                    # 테스트용 Terraform 파일
│   ├── insecure-s3.tf          # 취약한 S3 설정
│   ├── secure-s3.tf            # 안전한 S3 설정
│   └── insecure-ec2.tf         # 취약한 EC2 설정
│
└── scan-results/               # 스캔 결과 저장 (자동 생성)
    └── YYYY-MM-DD/
        └── *.json
```

## 🚀 빠른 시작

### 전제 조건

- Go 1.21 이상
- trivy-checks-source 디렉토리 (정책 소스)

### 1. 의존성 설치

```bash
cd terraform-scanner-service
go mod download
```

### 2. 서비스 시작

```bash
# 기본 실행 (자동으로 ../trivy-checks-source/checks 사용)
go run main.go

# 정책 디렉토리 명시
POLICY_DIR=/path/to/trivy-checks-source/checks go run main.go
```

출력:
```
Loaded 150 policy modules
Compiled 150 policy modules successfully
Initializing Terraform scanner...
Scanner initialized with 150 policies
Starting Terraform Scanner API on :8080
```

### 3. 테스트

```bash
# 헬스 체크
curl http://localhost:8080/health

# 단일 파일 스캔
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/insecure-s3.tf"}'

# 전체 테스트 실행
chmod +x test.sh
./test.sh
```

## 📖 API 문서

### POST /scan

Terraform 파일 또는 디렉토리를 스캔합니다.

**요청:**
```json
{
  "target": "/path/to/terraform/file-or-directory"
}
```

**응답 (성공):**
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

**응답 (실패):**
```json
{
  "status": "error",
  "error": "target path not found: /invalid/path"
}
```

### GET /health

서비스 상태를 확인합니다.

**응답:**
```json
{
  "status": "healthy",
  "policies_loaded": 150,
  "timestamp": "2025-10-31T12:00:00Z"
}
```

### GET /policies

로드된 정책 목록을 반환합니다.

**응답:**
```json
{
  "count": 150,
  "policies": [
    {
      "id": "AVD-AWS-0088",
      "avd_id": "AVD-AWS-0088",
      "title": "S3 Bucket Encryption",
      "severity": "HIGH",
      "service": "s3",
      "provider": "AWS"
    }
  ]
}
```

## 📋 스캔 결과 형식

결과는 Trivy와 동일한 JSON 형식으로 저장됩니다:

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
          "ID": "AVD-AWS-0088",
          "AVDID": "AVD-AWS-0088",
          "Title": "S3 Bucket does not have encryption enabled",
          "Description": "S3 bucket should have encryption enabled to protect data at rest",
          "Message": "Bucket 'my-bucket' does not have encryption enabled",
          "Namespace": "builtin.aws.s3",
          "Query": "data.builtin.aws.s3.deny",
          "Resolution": "Enable encryption for the S3 bucket",
          "Severity": "HIGH",
          "PrimaryURL": "https://avd.aquasec.com/misconfig/avd-aws-0088",
          "References": [
            "https://docs.aws.amazon.com/..."
          ],
          "Status": "FAIL",
          "CauseMetadata": {
            "Resource": "aws_s3_bucket.example",
            "Provider": "AWS",
            "Service": "s3",
            "StartLine": 3,
            "EndLine": 8
          }
        }
      ]
    }
  ]
}
```

## 🏗️ 아키텍처

### 핵심 컴포넌트

1. **Policy Loader**: trivy-checks에서 Rego 정책 로드 및 컴파일
2. **Terraform Parser**: HCL 파일을 Go 구조체로 파싱
3. **Rego Engine**: OPA를 사용한 정책 평가
4. **API Handler**: HTTP 요청 처리 및 결과 저장

### 스캔 프로세스

```
HTTP Request → Handler → Scanner → Parser → Rego Engine → Result
                                       ↓
                                 Policy Store
                              (trivy-checks)
```

상세한 내용은 [ARCHITECTURE.md](ARCHITECTURE.md)를 참조하세요.

## 🔧 빌드 및 배포

### 로컬 빌드

```bash
./build.sh
./terraform-scanner-service
```

### Docker 빌드

```bash
docker build -t terraform-scanner:latest .
docker run -p 8080:8080 \
  -v $(pwd)/../trivy-checks-source/checks:/policies:ro \
  -v $(pwd)/scan-results:/root/scan-results \
  terraform-scanner:latest
```

### Docker Compose

```bash
docker-compose up -d
```

## 📊 사용 예제

### 예제 1: 취약한 S3 설정 스캔

```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/insecure-s3.tf"}'
```

발견될 문제들:
- ❌ S3 버킷 암호화 미설정
- ❌ 버저닝 미활성화
- ❌ 퍼블릭 액세스 허용
- ❌ 액세스 로깅 미설정

### 예제 2: 안전한 S3 설정 스캔

```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/secure-s3.tf"}'
```

결과: 문제 없음 또는 경미한 경고만 발견

### 예제 3: 디렉토리 전체 스캔

```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./my-terraform-project/"}'
```

## 🎨 특징

### ✅ 구현된 기능

- [x] Rego 정책 로드 및 컴파일
- [x] Terraform HCL 파싱
- [x] OPA 기반 정책 평가
- [x] Trivy 호환 JSON 출력
- [x] REST API (스캔, 헬스체크, 정책 목록)
- [x] 파일/디렉토리 스캔
- [x] 결과 파일 저장
- [x] Docker 지원

### 🚧 제한사항

- Terraform HCL 소스만 지원 (plan/state 파일 미지원)
- 원격 모듈은 로컬 다운로드 필요
- 단일 인스턴스만 테스트됨 (수평 확장 미검증)
- 인증/권한 관리 미구현

## 🔍 문제 해결

### 정책 로드 실패

```bash
# 경로 확인
ls -la ../trivy-checks-source/checks/cloud/

# 명시적 경로 지정
POLICY_DIR=/absolute/path/to/checks go run main.go
```

### 파싱 에러

Terraform 파일의 HCL 문법을 확인하세요:
```bash
terraform fmt -check ./examples/
terraform validate
```

### 포트 충돌

`main.go`에서 포트 변경:
```go
srv := &http.Server{
    Addr: ":8081",  // 다른 포트 사용
    ...
}
```

## 📚 관련 문서

- [QUICKSTART.md](QUICKSTART.md) - 빠른 시작 가이드
- [ARCHITECTURE.md](ARCHITECTURE.md) - 아키텍처 상세 설명
- [examples/](examples/) - Terraform 예제 파일

## 🤝 기여

이 프로젝트는 참고용 프로토타입입니다. 개선 아이디어:

1. **성능 최적화**: 디렉토리 스캔 병렬화
2. **추가 포맷**: CloudFormation, Kubernetes 지원
3. **결과 포맷**: SARIF, HTML 리포트
4. **인증**: API 키 기반 인증
5. **웹 UI**: 결과 시각화 대시보드

## 📝 라이선스

교육 및 참고 목적으로 제작되었습니다.

## 🔗 참고 자료

- [Trivy](https://github.com/aquasecurity/trivy) - 원본 프로젝트
- [trivy-checks](https://github.com/aquasecurity/trivy-checks) - 정책 저장소
- [Open Policy Agent](https://www.openpolicyagent.org/) - Rego 정책 엔진
- [Terraform](https://www.terraform.io/) - Infrastructure as Code

## 📧 연락처

질문이나 제안이 있으시면 이슈를 생성해주세요.

---

**Note**: 이 서비스는 Trivy를 직접 호출하지 않고 trivy-checks의 정책을 내장하여 독립적으로 실행됩니다. 프로덕션 사용 전 충분한 테스트가 필요합니다.
