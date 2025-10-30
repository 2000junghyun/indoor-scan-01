package types

import "time"

// Trivy 호환 JSON 구조

// ScanResult는 Trivy의 최상위 결과 구조입니다
type ScanResult struct {
	SchemaVersion int       `json:"SchemaVersion"`
	CreatedAt     time.Time `json:"CreatedAt"`
	ArtifactName  string    `json:"ArtifactName"`
	ArtifactType  string    `json:"ArtifactType"`
	Results       []Result  `json:"Results"`
}

// Result는 스캔 대상별 결과입니다
type Result struct {
	Target            string             `json:"Target"`
	Class             string             `json:"Class"`
	Type              string             `json:"Type"`
	Misconfigurations []Misconfiguration `json:"Misconfigurations,omitempty"`
}

// Misconfiguration은 발견된 보안 문제입니다
type Misconfiguration struct {
	Type          string         `json:"Type"`
	ID            string         `json:"ID"`
	AVDID         string         `json:"AVDID"`
	Title         string         `json:"Title"`
	Description   string         `json:"Description"`
	Message       string         `json:"Message"`
	Namespace     string         `json:"Namespace"`
	Query         string         `json:"Query"`
	Resolution    string         `json:"Resolution"`
	Severity      string         `json:"Severity"`
	PrimaryURL    string         `json:"PrimaryURL"`
	References    []string       `json:"References"`
	Status        string         `json:"Status"`
	Layer         Layer          `json:"Layer"`
	CauseMetadata *CauseMetadata `json:"CauseMetadata,omitempty"`
}

// Layer는 레이어 정보입니다 (컨테이너용, 파일시스템에선 빈 객체)
type Layer struct{}

// CauseMetadata는 문제의 원인 위치 정보입니다
type CauseMetadata struct {
	Resource  string     `json:"Resource"`
	Provider  string     `json:"Provider"`
	Service   string     `json:"Service"`
	StartLine int        `json:"StartLine"`
	EndLine   int        `json:"EndLine"`
	Code      *CodeLines `json:"Code,omitempty"`
}

// CodeLines는 문제가 발생한 코드 라인입니다
type CodeLines struct {
	Lines []CodeLine `json:"Lines"`
}

// CodeLine은 개별 코드 라인입니다
type CodeLine struct {
	Number      int    `json:"Number"`
	Content     string `json:"Content"`
	IsCause     bool   `json:"IsCause"`
	Annotation  string `json:"Annotation"`
	Truncated   bool   `json:"Truncated"`
	Highlighted string `json:"Highlighted,omitempty"`
	FirstCause  bool   `json:"FirstCause"`
	LastCause   bool   `json:"LastCause"`
}

// PolicyMetadata는 정책의 메타데이터입니다
type PolicyMetadata struct {
	ID          string   `json:"id"`
	AVDID       string   `json:"avd_id"`
	Title       string   `json:"title"`
	ShortCode   string   `json:"short_code"`
	Description string   `json:"description"`
	Service     string   `json:"service"`
	Provider    string   `json:"provider"`
	Severity    string   `json:"severity"`
	Resolution  string   `json:"resolution"`
	References  []string `json:"references"`
}
