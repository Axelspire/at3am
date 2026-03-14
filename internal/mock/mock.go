// Package mock provides mock DNS resolution for testing at3am without real DNS.
package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/axelspire/at3am/internal/resolver"
)

// Scenario defines a mock DNS scenario.
type Scenario struct {
	Name        string
	Description string
	// QueryFunc returns a mock result for a given domain and server.
	QueryFunc func(domain, server string, callCount int) resolver.Result
	// AuthNS is a list of mock authoritative nameservers.
	AuthNS []string
}

// Querier implements resolver.DNSQuerier with mock responses.
type Querier struct {
	mu        sync.Mutex
	scenario  Scenario
	callCount map[string]int
}

// NewQuerier creates a mock querier with the given scenario.
func NewQuerier(scenario Scenario) *Querier {
	return &Querier{
		scenario:  scenario,
		callCount: make(map[string]int),
	}
}

// QueryTXT returns a mock TXT record result.
func (q *Querier) QueryTXT(_ context.Context, domain, server string) resolver.Result {
	q.mu.Lock()
	key := server + ":" + domain
	q.callCount[key]++
	count := q.callCount[key]
	q.mu.Unlock()

	return q.scenario.QueryFunc(domain, server, count)
}

// DiscoverAuthoritativeNS returns mock authoritative NS.
func (q *Querier) DiscoverAuthoritativeNS(_ context.Context, _ string) ([]string, error) {
	if len(q.scenario.AuthNS) == 0 {
		return nil, fmt.Errorf("mock: no authoritative NS configured")
	}
	return q.scenario.AuthNS, nil
}

// PredefinedScenarios returns all available mock scenarios.
func PredefinedScenarios() map[string]Scenario {
	return map[string]Scenario{
		"instant": {
			Name:        "instant",
			Description: "Record is immediately available everywhere",
			AuthNS:      []string{"10.0.0.1", "10.0.0.2"},
			QueryFunc: func(domain, server string, _ int) resolver.Result {
				return resolver.Result{
					Resolver: server, Found: true,
					Values: []string{"mock-validation-token"}, TTL: 300,
					Latency: 5 * time.Millisecond,
				}
			},
		},
		"slow_propagation": {
			Name:        "slow_propagation",
			Description: "Auth NS have it immediately, public resolvers pick it up after a few polls",
			AuthNS:      []string{"10.0.0.1", "10.0.0.2"},
			QueryFunc: func(domain, server string, callCount int) resolver.Result {
				// Auth resolvers always have it
				if server == "10.0.0.1" || server == "10.0.0.2" {
					return resolver.Result{
						Resolver: server, Found: true,
						Values: []string{"mock-validation-token"}, TTL: 60,
						Latency: 10 * time.Millisecond,
					}
				}
				// Public resolvers: found after 3 calls
				if callCount >= 3 {
					return resolver.Result{
						Resolver: server, Found: true,
						Values: []string{"mock-validation-token"}, TTL: 300,
						Latency: 30 * time.Millisecond,
					}
				}
				return resolver.Result{
					Resolver: server, Found: false,
					Latency: 20 * time.Millisecond,
				}
			},
		},
		"timeout": {
			Name:        "timeout",
			Description: "Record never appears (simulates a missing record or wrong domain)",
			AuthNS:      []string{"10.0.0.1"},
			QueryFunc: func(domain, server string, _ int) resolver.Result {
				return resolver.Result{
					Resolver: server, Found: false,
					Latency: 15 * time.Millisecond,
				}
			},
		},
		"flaky": {
			Name:        "flaky",
			Description: "Some resolvers intermittently fail",
			AuthNS:      []string{"10.0.0.1", "10.0.0.2"},
			QueryFunc: func(domain, server string, callCount int) resolver.Result {
				// Every other call for certain resolvers fails
				if (server == "8.8.8.8" || server == "1.1.1.1") && callCount%2 == 1 {
					return resolver.Result{
						Resolver: server, Error: "i/o timeout",
						Latency: 2 * time.Second,
					}
				}
				return resolver.Result{
					Resolver: server, Found: true,
					Values: []string{"mock-validation-token"}, TTL: 300,
					Latency: 10 * time.Millisecond,
				}
			},
		},
		"partial": {
			Name:        "partial",
			Description: "Some resolvers see the record, others never do",
			AuthNS:      []string{"10.0.0.1", "10.0.0.2"},
			QueryFunc: func(domain, server string, _ int) resolver.Result {
				// Auth and first few public see it
				switch server {
				case "10.0.0.1", "10.0.0.2", "8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1",
					"9.9.9.9", "149.112.112.112", "208.67.222.222", "208.67.220.220":
					return resolver.Result{
						Resolver: server, Found: true,
						Values: []string{"mock-validation-token"}, TTL: 300,
						Latency: 10 * time.Millisecond,
					}
				default:
					return resolver.Result{Resolver: server, Found: false, Latency: 10 * time.Millisecond}
				}
			},
		},
	}
}

