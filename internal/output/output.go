// Package output handles formatting and emitting results in multiple formats.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/diagnostics"
	"github.com/axelspire/at3am/internal/ttl"
)

// PollStatus represents the current state of a polling cycle.
type PollStatus struct {
	Domain      string               `json:"domain"`
	Expected    string               `json:"expected"`
	Attempt     int                  `json:"attempt"`
	Score       confidence.Score     `json:"score"`
	TTLReport   ttl.Report           `json:"ttl_report"`
	Diagnosis   diagnostics.Diagnosis `json:"diagnosis"`
	Elapsed     time.Duration        `json:"elapsed"`
	ConsecPasses int                 `json:"consecutive_passes"`
	RequiredPasses int               `json:"required_passes"`
	Ready       bool                 `json:"ready"`
}

// FinalResult represents the final outcome.
type FinalResult struct {
	Success     bool          `json:"success"`
	Domain      string        `json:"domain"`
	Confidence  float64       `json:"confidence"`
	Elapsed     time.Duration `json:"elapsed"`
	Attempts    int           `json:"attempts"`
	Message     string        `json:"message"`
}

// Formatter outputs poll status and final results.
type Formatter struct {
	format string
	writer io.Writer
}

// NewFormatter creates a new formatter.
func NewFormatter(format string, writer io.Writer) *Formatter {
	return &Formatter{format: format, writer: writer}
}

// EmitPollStatus outputs the current poll status.
func (f *Formatter) EmitPollStatus(status PollStatus) {
	switch f.format {
	case "json":
		f.emitJSON(status)
	case "quiet":
		// no output during polling
	default:
		f.emitHuman(status)
	}
}

// EmitFinalResult outputs the final result.
func (f *Formatter) EmitFinalResult(result FinalResult) {
	switch f.format {
	case "json":
		f.emitJSON(result)
	case "quiet":
		// quiet mode: only output on success
		if result.Success {
			fmt.Fprintln(f.writer, "READY")
		}
	default:
		f.emitHumanFinal(result)
	}
}

func (f *Formatter) emitJSON(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Fprintln(f.writer, string(data))
}

func (f *Formatter) emitHuman(status PollStatus) {
	fmt.Fprintf(f.writer, "[%s] Poll #%d | Confidence: %.1f%% (auth: %.1f%% [%d/%d], public: %.1f%% [%d/%d]) | Passes: %d/%d\n",
		formatDuration(status.Elapsed),
		status.Attempt,
		status.Score.Overall,
		status.Score.AuthScore, status.Score.AuthFound, status.Score.AuthTotal,
		status.Score.PublicScore, status.Score.PublicFound, status.Score.PublicTotal,
		status.ConsecPasses, status.RequiredPasses,
	)
	if status.Score.AuthErrors > 0 || status.Score.PublicErrors > 0 {
		fmt.Fprintf(f.writer, "  ⚠ Errors: auth=%d, public=%d\n", status.Score.AuthErrors, status.Score.PublicErrors)
	}
	if status.Score.DNSSECChecked {
		fmt.Fprintf(f.writer, "  🔒 DNSSEC: %d/%d resolvers authenticated (AD bit)\n",
			status.Score.DNSSECValidCount, status.Score.DNSSECTotal)
	}
	if status.Diagnosis.Scenario != "full_propagation" && status.Diagnosis.Scenario != "" {
		fmt.Fprintf(f.writer, "  → %s\n", status.Diagnosis.Summary)
	}
}

func (f *Formatter) emitHumanFinal(result FinalResult) {
	if result.Success {
		fmt.Fprintf(f.writer, "\n✓ READY — %s propagated (confidence: %.1f%%, %d attempts, %s elapsed)\n",
			result.Domain, result.Confidence, result.Attempts, formatDuration(result.Elapsed))
	} else {
		fmt.Fprintf(f.writer, "\n✗ TIMEOUT — %s NOT ready after %s (%d attempts). %s\n",
			result.Domain, formatDuration(result.Elapsed), result.Attempts, result.Message)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ExpandTemplate expands template variables in a string.
func ExpandTemplate(template string, domain string, confidence float64, elapsed time.Duration) string {
	r := strings.NewReplacer(
		"$DOMAIN", domain,
		"$CONFIDENCE", fmt.Sprintf("%.1f", confidence),
		"$ELAPSED", formatDuration(elapsed),
	)
	return r.Replace(template)
}

