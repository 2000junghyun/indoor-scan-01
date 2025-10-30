# Terraform Scanner Service - Quick Start Guide

## 빠른 시작

### 1. 의존성 설치

```bash
cd terraform-scanner-service
go mod download
```

### 2. 서비스 시작

```bash
# 기본 설정 (자동으로 ../trivy-checks-source/checks 사용)
go run main.go

# 또는 정책 디렉토리 명시
POLICY_DIR=/path/to/trivy-checks-source/checks go run main.go
```

서비스가 시작되면 다음과 같은 로그가 표시됩니다:
```
Loaded 150 policy modules
Compiled 150 policy modules successfully
Initializing Terraform scanner...
Scanner initialized with 150 policies
Starting Terraform Scanner API on :8080
```

### 3. 테스트 실행

다른 터미널에서:

```bash
# 실행 권한 부여
chmod +x test.sh

# 테스트 스크립트 실행
./test.sh
```

### 4. 수동 테스트

#### Health Check
```bash
curl http://localhost:8080/health
```

**응답:**
```json
{
  "policies_loaded": 150,
  "status": "healthy",
  "timestamp": "2025-10-31T12:00:00Z"
}
```

#### 단일 파일 스캔
```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/insecure-s3.tf"}'
```

**응답:**
```json
{
  "status": "success",
  "results": [
    {
      "file": "scan-results/2025-10-31/insecure-s3-scan-result.json",
      "target": "insecure-s3.tf"
    }
  ]
}
```

#### 디렉토리 스캔
```bash
curl -X POST http://localhost:8080/scan \
  -H "Content-Type: application/json" \
  -d '{"target": "./examples/"}'
```

#### 정책 목록 조회
```bash
curl http://localhost:8080/policies | jq '.'
```

### 5. 스캔 결과 확인

```bash
# 오늘 날짜의 스캔 결과 조회
ls -la scan-results/$(date +%Y-%m-%d)/

# 특정 결과 파일 보기
cat scan-results/$(date +%Y-%m-%d)/insecure-s3-scan-result.json | jq '.'
```

## 예제 파일 설명

### `examples/insecure-s3.tf`
의도적으로 보안 문제가 있는 S3 설정:
- ❌ 암호화 미설정
- ❌ 버저닝 미설정
- ❌ 퍼블릭 액세스 허용
- ❌ 로깅 미설정

### `examples/secure-s3.tf`
보안이 강화된 S3 설정:
- ✅ KMS 암호화 설정
- ✅ 버저닝 활성화
- ✅ 퍼블릭 액세스 차단
- ✅ 액세스 로깅 활성화

### `examples/insecure-ec2.tf`
의도적으로 보안 문제가 있는 EC2 설정:
- ❌ EBS 암호화 미설정
- ❌ 퍼블릭 IP 할당
- ❌ IMDSv1 사용
- ❌ 과도하게 개방된 보안 그룹

## 스캔 결과 형식

스캔 결과는 Trivy와 동일한 JSON 형식으로 저장됩니다:

```json
{
  "SchemaVersion": 2,
  "CreatedAt": "2025-10-31T12:00:00Z",
  "ArtifactName": "insecure-s3.tf",
  "ArtifactType": "terraform",
  "Results": [
    {
      "Target": "insecure-s3.tf",
      "Class": "config",
      "Type": "terraform",
      "Misconfigurations": [
        {
          "Type": "Terraform Security Check",
          "ID": "AVD-AWS-0088",
          "AVDID": "AVD-AWS-0088",
          "Title": "S3 Bucket does not have encryption enabled",
          "Description": "...",
          "Message": "Bucket 'my-insecure-bucket' does not have encryption enabled",
          "Severity": "HIGH",
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

## 문제 해결

### "no policy files found" 오류

정책 디렉토리 경로를 확인하세요:
```bash
ls -la ../trivy-checks-source/checks/cloud/
```

명시적으로 경로 지정:
```bash
POLICY_DIR=/absolute/path/to/trivy-checks-source/checks go run main.go
```

### 컴파일 에러

의존성 재설치:
```bash
go mod tidy
go mod download
```

### 포트 충돌

다른 포트 사용 (main.go 수정 필요):
```go
srv := &http.Server{
    Addr: ":8081",  // 포트 변경
    ...
}
```

## 성능 최적화

- 정책은 서비스 시작 시 1회만 로드되므로 재시작 없이 빠른 스캔 가능
- 디렉토리 스캔 시 각 파일은 병렬로 처리되지 않으므로 큰 디렉토리는 시간이 걸릴 수 있음
- 스캔 타임아웃: 30초 (handler.go에서 조정 가능)

## 다음 단계

1. **커스텀 정책 추가**: trivy-checks-source에 .rego 파일 추가 후 서비스 재시작
2. **CI/CD 통합**: Jenkins, GitLab CI 등에서 API 호출
3. **대시보드 구축**: 스캔 결과를 시각화하는 프론트엔드 추가
4. **알림 설정**: 중대한 취약점 발견 시 Slack/Email 알림

## 참고사항

- 이 서비스는 Terraform HCL 파일만 지원 (plan 파일 미지원)
- State 파일 스캔 불가 (소스 코드 분석만)
- 원격 모듈은 로컬에 다운로드 필요
