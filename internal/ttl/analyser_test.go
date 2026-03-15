package ttl

import (
	"testing"

	"github.com/axelspire/at3am/internal/resolver"
)

func TestAnalyse_NoResults(t *testing.T) {
	a := NewAnalyser()
	report := a.Analyse(nil)
	if report.SampleCount != 0 {
		t.Errorf("expected 0 samples, got %d", report.SampleCount)
	}
	if len(report.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(report.Warnings))
	}
}

func TestAnalyse_AllErrors(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Error: "timeout"},
		{Resolver: "1.1.1.1", Error: "timeout"},
	}
	report := a.Analyse(results)
	if report.SampleCount != 0 {
		t.Errorf("expected 0 samples, got %d", report.SampleCount)
	}
}

func TestAnalyse_NotFound(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: false},
	}
	report := a.Analyse(results)
	if report.SampleCount != 0 {
		t.Errorf("expected 0 samples, got %d", report.SampleCount)
	}
}

func TestAnalyse_SingleResult(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 300},
	}
	report := a.Analyse(results)
	if report.SampleCount != 1 {
		t.Errorf("expected 1 sample, got %d", report.SampleCount)
	}
	if report.MinTTL != 300 || report.MaxTTL != 300 || report.AvgTTL != 300 {
		t.Errorf("expected TTLs all 300, got min=%d max=%d avg=%d", report.MinTTL, report.MaxTTL, report.AvgTTL)
	}
}

func TestAnalyse_MultipleResults(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 100},
		{Resolver: "1.1.1.1", Found: true, Values: []string{"token"}, TTL: 200},
		{Resolver: "9.9.9.9", Found: true, Values: []string{"token"}, TTL: 300},
	}
	report := a.Analyse(results)
	if report.MinTTL != 100 {
		t.Errorf("expected min 100, got %d", report.MinTTL)
	}
	if report.MaxTTL != 300 {
		t.Errorf("expected max 300, got %d", report.MaxTTL)
	}
	if report.AvgTTL != 200 {
		t.Errorf("expected avg 200, got %d", report.AvgTTL)
	}
	if report.SampleCount != 3 {
		t.Errorf("expected 3 samples, got %d", report.SampleCount)
	}
}

func TestAnalyse_HighTTLWarning(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 7200},
	}
	report := a.Analyse(results)
	found := false
	for _, w := range report.Warnings {
		if len(w) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected high TTL warning")
	}
}

func TestAnalyse_LowTTLWarning(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 10},
	}
	report := a.Analyse(results)
	found := false
	for _, w := range report.Warnings {
		if len(w) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected low TTL warning")
	}
}

func TestAnalyse_LargeSpreadWarning(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 10},
		{Resolver: "1.1.1.1", Found: true, Values: []string{"token"}, TTL: 3600},
	}
	report := a.Analyse(results)
	if len(report.Warnings) == 0 {
		t.Error("expected warnings for large spread")
	}
}

func TestAnalyse_EstimatedPropagation(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 300},
		{Resolver: "1.1.1.1", Found: true, Values: []string{"token"}, TTL: 600},
	}
	report := a.Analyse(results)
	if report.EstimatedFullPropagation.Seconds() != 600 {
		t.Errorf("expected 600s propagation, got %s", report.EstimatedFullPropagation)
	}
}

func TestAnalyse_MixedErrorsAndFound(t *testing.T) {
	a := NewAnalyser()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}, TTL: 300},
		{Resolver: "1.1.1.1", Error: "timeout"},
		{Resolver: "9.9.9.9", Found: false},
	}
	report := a.Analyse(results)
	if report.SampleCount != 1 {
		t.Errorf("expected 1 sample, got %d", report.SampleCount)
	}
}

