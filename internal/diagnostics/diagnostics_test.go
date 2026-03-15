package diagnostics

import (
	"testing"

	"github.com/axelspire/at3am/internal/confidence"
	"github.com/axelspire/at3am/internal/resolver"
	"github.com/axelspire/at3am/internal/ttl"
)

func TestDiagnose_NoRecord(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Found: false},
		{Resolver: "1.1.1.1", Found: false},
	}
	score := confidence.Score{AuthFound: 0, PublicFound: 0}
	diag := e.Diagnose(results, score, ttl.Report{})

	if diag.Scenario != "no_record" {
		t.Errorf("expected scenario 'no_record', got %q", diag.Scenario)
	}
	if diag.Severity != "error" {
		t.Errorf("expected severity 'error', got %q", diag.Severity)
	}
	if len(diag.Recommendations) == 0 {
		t.Error("expected recommendations")
	}
}

func TestDiagnose_NoRecord_WithErrors(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "8.8.8.8", Error: "timeout"},
		{Resolver: "1.1.1.1", Found: false},
	}
	score := confidence.Score{AuthFound: 0, PublicFound: 0}
	diag := e.Diagnose(results, score, ttl.Report{})

	if diag.Scenario != "no_record" {
		t.Errorf("expected scenario 'no_record', got %q", diag.Scenario)
	}
}

func TestDiagnose_AuthOnly(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: false},
	}
	score := confidence.Score{AuthFound: 1, AuthTotal: 1, PublicFound: 0, PublicTotal: 1}
	ttlReport := ttl.Report{MaxTTL: 300}
	diag := e.Diagnose(results, score, ttlReport)

	if diag.Scenario != "auth_only" {
		t.Errorf("expected scenario 'auth_only', got %q", diag.Scenario)
	}
	if diag.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", diag.Severity)
	}
}

func TestDiagnose_AuthOnly_NoTTL(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{}
	score := confidence.Score{AuthFound: 1, AuthTotal: 1, PublicFound: 0, PublicTotal: 1}
	diag := e.Diagnose(results, score, ttl.Report{MaxTTL: 0})

	if diag.Scenario != "auth_only" {
		t.Errorf("expected scenario 'auth_only', got %q", diag.Scenario)
	}
}

func TestDiagnose_PartialPropagation(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}},
		{Resolver: "1.1.1.1", Found: false},
	}
	score := confidence.Score{
		Overall: 75.0, AuthFound: 1, AuthTotal: 1,
		PublicFound: 1, PublicTotal: 2,
	}
	diag := e.Diagnose(results, score, ttl.Report{})

	if diag.Scenario != "partial_propagation" {
		t.Errorf("expected scenario 'partial_propagation', got %q", diag.Scenario)
	}
	if diag.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", diag.Severity)
	}
}

func TestDiagnose_FullPropagation(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}},
	}
	score := confidence.Score{
		Overall: 100.0, AuthFound: 1, AuthTotal: 1,
		PublicFound: 1, PublicTotal: 1,
	}
	diag := e.Diagnose(results, score, ttl.Report{})

	if diag.Scenario != "full_propagation" {
		t.Errorf("expected scenario 'full_propagation', got %q", diag.Scenario)
	}
	if diag.Severity != "info" {
		t.Errorf("expected severity 'info', got %q", diag.Severity)
	}
}

func TestDiagnose_PartialWithMissingResolvers(t *testing.T) {
	e := NewEngine()
	results := []resolver.Result{
		{Resolver: "10.0.0.1", Found: true, Values: []string{"token"}, AuthoritativeNS: true},
		{Resolver: "8.8.8.8", Found: true, Values: []string{"token"}},
		{Resolver: "1.1.1.1", Found: false},
		{Resolver: "9.9.9.9", Error: "timeout"},
	}
	score := confidence.Score{
		Overall: 50.0, AuthFound: 1, AuthTotal: 1,
		PublicFound: 1, PublicTotal: 3,
	}
	diag := e.Diagnose(results, score, ttl.Report{})

	if diag.Scenario != "partial_propagation" {
		t.Errorf("expected scenario 'partial_propagation', got %q", diag.Scenario)
	}
}

