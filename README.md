# indoor-scan-01

<img width="761" height="185" alt="image" src="https://github.com/user-attachments/assets/de580a23-3c20-447e-a238-bbbe23e1cb18" />

### 명령어
```
현재 세 개의 디렉토리를 확인할 수 있을 거야.

trivy-checks-source는 Terraform 코드의 misconfiguration을 판단하는데 필요한 규칙을 다운받았어. 이건 실제 Trivy가 참조하는 외부 정책과 동일해.
trivy-source는 Trivy의 소스코드야. 여기서 파일을 입력 받는 것부터 스캔 결과를 출력하는 과정까지의 메인 로직과 코드를 확인할 수 있어.
trivy-scan-api-source는 HTTP microservice (REST) 형태로 8080 포트 같은 곳에 서비스를 띄워두고 curl과 같은 명령어로 요청을 보내서 결과를 받는 구조야. 현재는 Terraform 파일을 입력하면 Trivy 실행 파일을 직접 호출해서 결과를 저장하는 구조야.
이 세 가지 소스를 이용해서 새로운 작업을 진행하고 싶어. 작업의 목적은 trivy-scan-api-source처럼 HTTP microservice 형태로 Terraform 파일을 스캔하는 것인데, Trivy를 직접 호출하지 않고 내부에 저장된 정책으로 파일을 스캔하는 거야. 정책의 리소스는 tricy-checks-source를 활용할 거야. 정리하면 서비스를 동작 시키고 curl과 같은 방법으로 Terraform 파일을 입력하면 내부에 저장된 정책으로 Terraform 파일을 스캔하고 스캔 결과를 json 형태로 저장하는 거야. Trivy가 외부에서 정책을 끌어와서 저장하지만, 나는 내부에 정책을 미리 저장하는 형태로 만들고 싶어. 그리고 json 결과물은 Trivy를 이용해서 얻는 스캔 결과물의 양식과 동일하게 저장하고 싶어.

기존 세 개의 디렉토리 내 코드는 참고만 하고 변형시키지 말아줘. 모든 코드는 새로운 파일과 디렉토리에 작성해줘.
```

### 핵심 구현 내용
1. 정책 로딩 (policy_loader.go)
trivy-checks-source/checks/cloud/ 에서 .rego 파일 스캔
OPA 컴파일러로 정책 컴파일
메타데이터 추출 (AVDID, Severity, Title 등)
2. Terraform 파싱 (terraform_parser.go)
HCL2 파서로 .tf 파일 파싱
Go map 구조로 변환
변수/참조 평가
3. 정책 평가 (rego_engine.go)
OPA Rego 엔진으로 정책 실행
deny/warn 규칙 평가
Misconfiguration 결과 생성
4. HTTP API (handler.go)
POST /scan: 파일/디렉토리 스캔
GET /health: 상태 확인
GET /policies: 로드된 정책 목록
결과를 Trivy JSON 형식으로 저장

### 차별화 포인트
독립성: Trivy 바이너리 불필요
성능: 정책 1회 로드 후 재사용
커스터마이징: 정책/파서 수정 가능
투명성: 스캔 로직 완전 공개

### 주요 기능
✅ HTTP API 서버 - Port 8080에서 실행
✅ Terraform 파일 스캔 - HCL 파싱 및 보안 검사
✅ Rego 정책 엔진 - OPA 기반 정책 평가
✅ Trivy 호환 출력 - JSON 형식 결과
✅ 정책 로딩 - trivy-checks 또는 custom 정책

### API 엔드포인트
GET /health - 서비스 상태 확인
POST /scan - Terraform 파일 스캔 (multipart/form-data)
GET /policies - 로드된 정책 목록

### 현재 상태
✅ 서비스 정상 작동
✅ Custom 정책으로 취약점 탐지 성공
⚠️ trivy-checks 정책은 18개 컴파일되었으나 특수 함수(result.new) 없이는 작동 불가

```
terraform-scanner-service/
├── 📄 main.go                     # 서비스 진입점 (HTTP 서버)
├── 📄 go.mod                      # Go 의존성
├── 📄 README.md                   # 프로젝트 메인 문서
├── 📄 PROJECT_OVERVIEW.md         # 전체 개요
├── 📄 QUICKSTART.md              # 빠른 시작 가이드
├── 📄 ARCHITECTURE.md            # 아키텍처 상세 설명
├── 📄 Dockerfile                 # Docker 이미지
├── 📄 docker-compose.yml         # Docker Compose
├── 📜 build.sh                   # 빌드 스크립트
├── 📜 test.sh                    # 테스트 스크립트
├── 📄 .gitignore
│
├── 📁 internal/
│   ├── 📁 api/
│   │   └── handler.go           # HTTP 핸들러 (POST /scan, GET /health, GET /policies)
│   ├── 📁 scanner/
│   │   ├── scanner.go           # 메인 스캐너 오케스트레이터
│   │   ├── policy_loader.go     # Rego 정책 로더 (trivy-checks 읽기)
│   │   ├── terraform_parser.go  # Terraform HCL 파서
│   │   └── rego_engine.go       # OPA Rego 실행 엔진
│   ├── 📁 types/
│   │   └── result.go            # Trivy 호환 타입 정의
│   └── 📁 utils/
│       └── file.go              # 파일 유틸리티
│
├── 📁 examples/                  # 테스트용 Terraform 파일
│   ├── insecure-s3.tf           # 취약한 S3 설정 (문제 발견용)
│   ├── secure-s3.tf             # 안전한 S3 설정 (정상 케이스)
│   └── insecure-ec2.tf          # 취약한 EC2 설정
│
└── 📁 scan-results/              # 스캔 결과 저장 (자동 생성)
    └── YYYY-MM-DD/
        └── *-scan-result.json
```
