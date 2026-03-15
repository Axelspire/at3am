package mock

import (
	"context"
	"testing"

	"github.com/axelspire/at3am/internal/resolver"
)

func TestPredefinedScenarios(t *testing.T) {
	scenarios := PredefinedScenarios()
	expected := []string{"instant", "slow_propagation", "timeout", "flaky", "partial"}
	for _, name := range expected {
		if _, ok := scenarios[name]; !ok {
			t.Errorf("missing scenario: %s", name)
		}
	}
	if len(scenarios) != len(expected) {
		t.Errorf("expected %d scenarios, got %d", len(expected), len(scenarios))
	}
}

func TestQuerier_Instant(t *testing.T) {
	s := PredefinedScenarios()["instant"]
	q := NewQuerier(s)
	ctx := context.Background()

	r := q.QueryTXT(ctx, "_acme-challenge.example.com", "8.8.8.8")
	if !r.Found {
		t.Error("expected found")
	}
	if len(r.Values) == 0 || r.Values[0] != "mock-validation-token" {
		t.Error("expected mock-validation-token")
	}
}

func TestQuerier_SlowPropagation(t *testing.T) {
	s := PredefinedScenarios()["slow_propagation"]
	q := NewQuerier(s)
	ctx := context.Background()

	// Auth should always find it
	r := q.QueryTXT(ctx, "test", "10.0.0.1")
	if !r.Found {
		t.Error("auth should always find it")
	}

	// Public: first 2 calls should not find it
	r = q.QueryTXT(ctx, "test", "8.8.8.8")
	if r.Found {
		t.Error("public should not find on first call")
	}
	r = q.QueryTXT(ctx, "test", "8.8.8.8")
	if r.Found {
		t.Error("public should not find on second call")
	}
	// Third call should find it
	r = q.QueryTXT(ctx, "test", "8.8.8.8")
	if !r.Found {
		t.Error("public should find on third call")
	}
}

func TestQuerier_Timeout(t *testing.T) {
	s := PredefinedScenarios()["timeout"]
	q := NewQuerier(s)
	ctx := context.Background()

	r := q.QueryTXT(ctx, "test", "8.8.8.8")
	if r.Found {
		t.Error("should never find")
	}
}

func TestQuerier_Flaky(t *testing.T) {
	s := PredefinedScenarios()["flaky"]
	q := NewQuerier(s)
	ctx := context.Background()

	// First call to 8.8.8.8 should error (callCount=1, odd)
	r := q.QueryTXT(ctx, "test", "8.8.8.8")
	if r.Error == "" {
		t.Error("expected error on first call")
	}
	// Second call should succeed (callCount=2, even)
	r = q.QueryTXT(ctx, "test", "8.8.8.8")
	if r.Error != "" {
		t.Errorf("expected no error on second call, got %s", r.Error)
	}
}

func TestQuerier_Partial(t *testing.T) {
	s := PredefinedScenarios()["partial"]
	q := NewQuerier(s)
	ctx := context.Background()

	// Known resolvers should find it
	r := q.QueryTXT(ctx, "test", "8.8.8.8")
	if !r.Found {
		t.Error("8.8.8.8 should find it")
	}

	// Unknown resolver should not
	r = q.QueryTXT(ctx, "test", "99.99.99.99")
	if r.Found {
		t.Error("99.99.99.99 should not find it")
	}
}

func TestQuerier_DiscoverAuthNS(t *testing.T) {
	s := PredefinedScenarios()["instant"]
	q := NewQuerier(s)
	ctx := context.Background()

	nss, err := q.DiscoverAuthoritativeNS(ctx, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nss) != 2 {
		t.Errorf("expected 2 auth NS, got %d", len(nss))
	}
}

func TestQuerier_DiscoverAuthNS_NoAuth(t *testing.T) {
	s := Scenario{
		Name:   "no-auth",
		AuthNS: nil,
		QueryFunc: func(_, _ string, _ int) resolver.Result {
			return resolver.Result{}
		},
	}
	q := NewQuerier(s)
	ctx := context.Background()

	_, err := q.DiscoverAuthoritativeNS(ctx, "example.com")
	if err == nil {
		t.Error("expected error when no auth NS")
	}
}

