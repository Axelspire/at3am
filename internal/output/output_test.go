package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/diagnostics"
	"github.com/axelspire/at3am/internal/ttl"
)

func TestFormatter_Human_PollStatus(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("human", &buf)
	status := PollStatus{
		Domain:   "example.com",
		Expected: "token",
		Attempt:  1,
		Score: confidence.Score{
			Overall: 75.0, AuthScore: 100.0, PublicScore: 50.0,
			AuthFound: 2, AuthTotal: 2, PublicFound: 5, PublicTotal: 10,
		},
		TTLReport:      ttl.Report{},
		Diagnosis:      diagnostics.Diagnosis{Scenario: "partial_propagation", Summary: "Partial"},
		Elapsed:        10 * time.Second,
		ConsecPasses:   0,
		RequiredPasses: 2,
	}
	f.EmitPollStatus(status)
	out := buf.String()
	if !strings.Contains(out, "75.0%") {
		t.Errorf("expected confidence in output: %s", out)
	}
	if !strings.Contains(out, "Poll #1") {
		t.Errorf("expected poll number: %s", out)
	}
	if !strings.Contains(out, "Partial") {
		t.Errorf("expected diagnosis summary: %s", out)
	}
}

func TestFormatter_Human_Errors(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("human", &buf)
	status := PollStatus{
		Score: confidence.Score{AuthErrors: 1, PublicErrors: 2},
	}
	f.EmitPollStatus(status)
	if !strings.Contains(buf.String(), "⚠") {
		t.Error("expected error warning symbol")
	}
}

func TestFormatter_Human_FullPropagation(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("human", &buf)
	status := PollStatus{
		Score:    confidence.Score{Overall: 100},
		Diagnosis: diagnostics.Diagnosis{Scenario: "full_propagation"},
	}
	f.EmitPollStatus(status)
	// Full propagation scenario should not print diagnosis
	if strings.Contains(buf.String(), "→") {
		t.Error("should not print diagnosis for full propagation")
	}
}

func TestFormatter_JSON_PollStatus(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("json", &buf)
	status := PollStatus{Domain: "example.com", Attempt: 1}
	f.EmitPollStatus(status)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["domain"] != "example.com" {
		t.Errorf("expected domain example.com, got %v", result["domain"])
	}
}

func TestFormatter_Quiet_PollStatus(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("quiet", &buf)
	status := PollStatus{Domain: "example.com"}
	f.EmitPollStatus(status)
	if buf.Len() != 0 {
		t.Error("quiet mode should produce no poll output")
	}
}

func TestFormatter_Human_FinalResult_Success(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("human", &buf)
	f.EmitFinalResult(FinalResult{
		Success: true, Domain: "example.com",
		Confidence: 99.5, Elapsed: 30 * time.Second, Attempts: 6,
	})
	if !strings.Contains(buf.String(), "✓ READY") {
		t.Error("expected READY in output")
	}
}

func TestFormatter_Human_FinalResult_Failure(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("human", &buf)
	f.EmitFinalResult(FinalResult{
		Success: false, Domain: "example.com",
		Elapsed: 300 * time.Second, Attempts: 60,
		Message: "timeout exceeded",
	})
	if !strings.Contains(buf.String(), "✗ TIMEOUT") {
		t.Error("expected TIMEOUT in output")
	}
}

func TestFormatter_JSON_FinalResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("json", &buf)
	f.EmitFinalResult(FinalResult{Success: true, Domain: "x.com"})
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestFormatter_Quiet_FinalResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter("quiet", &buf)
	f.EmitFinalResult(FinalResult{Success: true})
	if !strings.Contains(buf.String(), "READY") {
		t.Error("quiet success should output READY")
	}

	buf.Reset()
	f.EmitFinalResult(FinalResult{Success: false})
	if buf.Len() != 0 {
		t.Error("quiet failure should produce no output")
	}
}

func TestExpandTemplate(t *testing.T) {
	result := ExpandTemplate("certbot renew $DOMAIN conf=$CONFIDENCE time=$ELAPSED",
		"example.com", 99.5, 30*time.Second)
	if !strings.Contains(result, "example.com") {
		t.Error("expected domain in template")
	}
	if !strings.Contains(result, "99.5") {
		t.Error("expected confidence in template")
	}
	if !strings.Contains(result, "30s") {
		t.Error("expected elapsed in template")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct{ d time.Duration; want string }{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{5 * time.Minute, "5m00s"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%s) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

