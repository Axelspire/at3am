package resolver

import (
	"context"
	"sync"

	logger "github.com/axelspire/at3am/internal/log"
)

// DefaultResolvers contains public anycast DNS resolvers.
var DefaultResolvers = []string{
	// Google
	"8.8.8.8", "8.8.4.4",
	// Cloudflare
	"1.1.1.1", "1.0.0.1",
	// Quad9
	"9.9.9.9", "149.112.112.112",
	// OpenDNS
	"208.67.222.222", "208.67.220.220",
	// Comodo Secure DNS
	"8.26.56.26", "8.20.247.20",
	// DNS.Watch
	"84.200.69.80", "84.200.70.40",
	// Verisign
	"64.6.64.6", "64.6.65.6",
	// CleanBrowsing
	"185.228.168.9", "185.228.169.9",
	// AdGuard
	"94.140.14.14", "94.140.15.15",
	// ControlID
	"76.76.2.0",
	// OpenDNS
	"208.67.222.222", "208.67.220.220",
	// Yandex
	"77.88.8.8", "77.88.8.1",
	// G-Core
	"95.85.95.85", "2.56.220.2",
}

// Pool manages a set of resolvers (public + authoritative).
type Pool struct {
	mu              sync.RWMutex
	publicResolvers []string
	authResolvers   []string
	querier         DNSQuerier
}

// NewPool creates a new resolver pool.
func NewPool(querier DNSQuerier, customResolvers []string) *Pool {
	public := make([]string, len(DefaultResolvers))
	copy(public, DefaultResolvers)
	public = append(public, customResolvers...)

	return &Pool{
		publicResolvers: public,
		querier:         querier,
	}
}

// DiscoverAuthNS discovers and adds authoritative nameservers for the domain.
func (p *Pool) DiscoverAuthNS(ctx context.Context, domain string) error {
	nss, err := p.querier.DiscoverAuthoritativeNS(ctx, domain)
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.authResolvers = nss
	p.mu.Unlock()
	return nil
}

// SetAuthResolvers sets the authoritative resolvers directly (useful for testing).
func (p *Pool) SetAuthResolvers(resolvers []string) {
	p.mu.Lock()
	p.authResolvers = resolvers
	p.mu.Unlock()
}

// QueryAll queries all resolvers in parallel and returns results.
func (p *Pool) QueryAll(ctx context.Context, domain string) []Result {
	p.mu.RLock()
	allPublic := make([]string, len(p.publicResolvers))
	copy(allPublic, p.publicResolvers)
	allAuth := make([]string, len(p.authResolvers))
	copy(allAuth, p.authResolvers)
	p.mu.RUnlock()

	totalCount := len(allPublic) + len(allAuth)
	results := make([]Result, 0, totalCount)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, server := range allAuth {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			r := p.querier.QueryTXT(ctx, domain, s)
			r.AuthoritativeNS = true
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(server)
	}

	for _, server := range allPublic {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			r := p.querier.QueryTXT(ctx, domain, s)
			r.AuthoritativeNS = false
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(server)
	}

	wg.Wait()
	return results
}

// QueryAuthFirst queries authoritative NS first. If ANY authoritative NS
// confirms the record (returns at least one TXT answer), public resolvers
// are queried in a second phase. If no authoritative NS finds the record,
// only the authoritative results are returned — this avoids poisoning
// public recursive resolvers with NXDOMAIN negative-cache entries.
//
// Special case: if no authoritative NS were discovered (e.g. NS discovery
// failed), the function falls back to querying public resolvers directly so
// that a meaningful result is returned rather than an empty set.
func (p *Pool) QueryAuthFirst(ctx context.Context, domain string) []Result {
	p.mu.RLock()
	allAuth := make([]string, len(p.authResolvers))
	copy(allAuth, p.authResolvers)
	allPublic := make([]string, len(p.publicResolvers))
	copy(allPublic, p.publicResolvers)
	p.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Special case: NS discovery failed — fall back to public resolvers so
	// the caller gets a meaningful result rather than an empty set.
	if len(allAuth) == 0 {
		logger.Debug("auth-first | no auth NS — falling back to %d public resolvers", len(allPublic))
		pubResults := make([]Result, 0, len(allPublic))
		for _, server := range allPublic {
			wg.Add(1)
			go func(s string) {
				defer wg.Done()
				r := p.querier.QueryTXT(ctx, domain, s)
				r.AuthoritativeNS = false
				mu.Lock()
				pubResults = append(pubResults, r)
				mu.Unlock()
			}(server)
		}
		wg.Wait()
		return pubResults
	}

	// Phase 1: query authoritative NS only.
	logger.Debug("auth-first | phase=1 querying %d auth resolvers", len(allAuth))
	authResults := make([]Result, 0, len(allAuth))
	for _, server := range allAuth {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			r := p.querier.QueryTXT(ctx, domain, s)
			r.AuthoritativeNS = true
			mu.Lock()
			authResults = append(authResults, r)
			mu.Unlock()
		}(server)
	}
	wg.Wait()

	// Check if any authoritative NS found the record.
	authConfirmed := false
	authFound := 0
	for _, r := range authResults {
		if r.Found {
			authConfirmed = true
			authFound++
		}
	}
	logger.Debug("auth-first | phase=1 done auth_found=%d/%d confirmed=%v", authFound, len(allAuth), authConfirmed)

	// Gate: if no ANS confirmed the record, return auth-only results.
	// Querying public resolvers before the source of truth confirms would
	// risk seeding their negative caches with a stale NXDOMAIN.
	if !authConfirmed {
		logger.Debug("auth-first | gate=closed — holding public resolvers until auth confirms")
		return authResults
	}

	// Phase 2: ANS confirmed — now query public resolvers.
	logger.Debug("auth-first | phase=2 gate=open querying %d public resolvers", len(allPublic))
	pubResults := make([]Result, 0, len(allPublic))
	for _, server := range allPublic {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			r := p.querier.QueryTXT(ctx, domain, s)
			r.AuthoritativeNS = false
			mu.Lock()
			pubResults = append(pubResults, r)
			mu.Unlock()
		}(server)
	}
	wg.Wait()

	return append(authResults, pubResults...)
}

// PublicCount returns the number of public resolvers.
func (p *Pool) PublicCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.publicResolvers)
}

// AuthCount returns the number of authoritative resolvers.
func (p *Pool) AuthCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.authResolvers)
}

// TotalCount returns the total number of resolvers.
func (p *Pool) TotalCount() int {
	return p.PublicCount() + p.AuthCount()
}

