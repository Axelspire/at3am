package resolver

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockQuerier implements DNSQuerier for testing.
type mockQuerier struct {
	queryFunc func(domain, server string) Result
	authNS    []string
	authErr   error
}

func (m *mockQuerier) QueryTXT(_ context.Context, domain, server string) Result {
	if m.queryFunc != nil {
		return m.queryFunc(domain, server)
	}
	return Result{Resolver: server, Found: true, Values: []string{"test-token"}, Latency: time.Millisecond}
}

func (m *mockQuerier) DiscoverAuthoritativeNS(_ context.Context, _ string) ([]string, error) {
	if m.authErr != nil {
		return nil, m.authErr
	}
	return m.authNS, nil
}

func TestNewPool(t *testing.T) {
	q := &mockQuerier{}
	pool := NewPool(q, nil)
	if pool.PublicCount() != len(DefaultResolvers) {
		t.Errorf("expected %d public resolvers, got %d", len(DefaultResolvers), pool.PublicCount())
	}
	if pool.AuthCount() != 0 {
		t.Errorf("expected 0 auth resolvers, got %d", pool.AuthCount())
	}
}

func TestNewPool_WithCustomResolvers(t *testing.T) {
	q := &mockQuerier{}
	pool := NewPool(q, []string{"10.10.10.10", "10.10.10.11"})
	expected := len(DefaultResolvers) + 2
	if pool.PublicCount() != expected {
		t.Errorf("expected %d public resolvers, got %d", expected, pool.PublicCount())
	}
}

func TestPool_DiscoverAuthNS(t *testing.T) {
	q := &mockQuerier{authNS: []string{"10.0.0.1", "10.0.0.2"}}
	pool := NewPool(q, nil)
	err := pool.DiscoverAuthNS(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.AuthCount() != 2 {
		t.Errorf("expected 2 auth resolvers, got %d", pool.AuthCount())
	}
}

func TestPool_DiscoverAuthNS_Error(t *testing.T) {
	q := &mockQuerier{authErr: fmt.Errorf("discovery failed")}
	pool := NewPool(q, nil)
	err := pool.DiscoverAuthNS(context.Background(), "example.com")
	if err == nil {
		t.Error("expected error")
	}
}

func TestPool_SetAuthResolvers(t *testing.T) {
	q := &mockQuerier{}
	pool := NewPool(q, nil)
	pool.SetAuthResolvers([]string{"10.0.0.1"})
	if pool.AuthCount() != 1 {
		t.Errorf("expected 1 auth resolver, got %d", pool.AuthCount())
	}
}

func TestPool_TotalCount(t *testing.T) {
	q := &mockQuerier{}
	pool := NewPool(q, []string{"custom1"})
	pool.SetAuthResolvers([]string{"auth1", "auth2"})
	expected := len(DefaultResolvers) + 1 + 2
	if pool.TotalCount() != expected {
		t.Errorf("expected %d total, got %d", expected, pool.TotalCount())
	}
}

func TestPool_QueryAll(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	q := &mockQuerier{
		queryFunc: func(domain, server string) Result {
			mu.Lock()
			callCount++
			mu.Unlock()
			return Result{Resolver: server, Found: true, Values: []string{"token"}, Latency: time.Millisecond}
		},
	}
	pool := NewPool(q, nil)
	pool.SetAuthResolvers([]string{"auth1", "auth2"})

	results := pool.QueryAll(context.Background(), "_acme-challenge.example.com")

	expectedCount := len(DefaultResolvers) + 2
	if len(results) != expectedCount {
		t.Errorf("expected %d results, got %d", expectedCount, len(results))
	}
	mu.Lock()
	gotCalls := callCount
	mu.Unlock()
	if gotCalls != expectedCount {
		t.Errorf("expected %d calls, got %d", expectedCount, gotCalls)
	}

	// Verify auth resolvers are marked correctly
	authCount := 0
	for _, r := range results {
		if r.AuthoritativeNS {
			authCount++
		}
	}
	if authCount != 2 {
		t.Errorf("expected 2 auth results, got %d", authCount)
	}
}

func TestPool_QueryAuthFirst_AuthNotFound(t *testing.T) {
	q := &mockQuerier{
		queryFunc: func(domain, server string) Result {
			// Auth servers return not found — record not yet propagated to ANS
			return Result{Resolver: server, Found: false, Latency: time.Millisecond}
		},
	}
	pool := NewPool(q, nil)
	pool.SetAuthResolvers([]string{"auth1", "auth2"})

	results := pool.QueryAuthFirst(context.Background(), "_acme-challenge.example.com")

	// Gate: ANS did not confirm → public resolvers must NOT be queried.
	// Only the 2 auth results should be returned.
	if len(results) != 2 {
		t.Errorf("expected 2 auth-only results, got %d", len(results))
	}
	for _, r := range results {
		if !r.AuthoritativeNS {
			t.Error("expected only auth results when ANS did not confirm")
		}
	}
}

func TestPool_QueryAuthFirst_AuthFound(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	q := &mockQuerier{
		queryFunc: func(domain, server string) Result {
			mu.Lock()
			callCount++
			mu.Unlock()
			return Result{Resolver: server, Found: true, Values: []string{"token"}, Latency: time.Millisecond}
		},
	}
	pool := NewPool(q, nil)
	pool.SetAuthResolvers([]string{"auth1", "auth2"})

	results := pool.QueryAuthFirst(context.Background(), "_acme-challenge.example.com")

	// Should have both auth + public results
	expectedCount := len(DefaultResolvers) + 2
	if len(results) != expectedCount {
		t.Errorf("expected %d total results, got %d", expectedCount, len(results))
	}

	authCount := 0
	for _, r := range results {
		if r.AuthoritativeNS {
			authCount++
		}
	}
	if authCount != 2 {
		t.Errorf("expected 2 auth results, got %d", authCount)
	}
}

func TestPool_QueryAuthFirst_NoAuthResolvers(t *testing.T) {
	q := &mockQuerier{
		queryFunc: func(domain, server string) Result {
			return Result{Resolver: server, Found: true, Values: []string{"token"}, Latency: time.Millisecond}
		},
	}
	pool := NewPool(q, nil)
	// No auth resolvers set — should fall back to public resolvers so the
	// caller still gets meaningful results even when NS discovery failed.

	results := pool.QueryAuthFirst(context.Background(), "_acme-challenge.example.com")

	// Should return all public resolver results (no auth resolvers → fallback)
	if len(results) != len(DefaultResolvers) {
		t.Errorf("expected %d public results when no auth resolvers, got %d", len(DefaultResolvers), len(results))
	}
	for _, r := range results {
		if r.AuthoritativeNS {
			t.Error("expected no auth-NS-flagged results in public fallback")
		}
	}
}

func TestDefaultResolvers_Count(t *testing.T) {
	const minResolvers = 10
	if len(DefaultResolvers) < minResolvers {
		t.Errorf("expected at least %d default resolvers, got %d", minResolvers, len(DefaultResolvers))
	}
	t.Logf("default resolver count: %d", len(DefaultResolvers))
}

