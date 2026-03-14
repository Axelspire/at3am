// Package resolver implements DNS resolution with a pool of global resolvers.
package resolver

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	logger "github.com/axelspire/at3am/internal/log"
)

// Result represents the outcome of a single resolver query.
type Result struct {
	Resolver        string        `json:"resolver"`
	Found           bool          `json:"found"`
	Values          []string      `json:"values,omitempty"`
	TTL             uint32        `json:"ttl,omitempty"`
	Error           string        `json:"error,omitempty"`
	Latency         time.Duration `json:"latency_ms"`
	Authoritative   bool          `json:"authoritative"`
	AuthoritativeNS bool          `json:"authoritative_ns"`
	// DNSSECChecked is true when the query was made with the DO bit set.
	DNSSECChecked bool `json:"dnssec_checked,omitempty"`
	// DNSSECValid is true when the resolver set the AD (Authenticated Data)
	// bit, meaning it validated the full DNSSEC chain for this response.
	DNSSECValid bool `json:"dnssec_valid,omitempty"`
}

// Resolver wraps a DNS client for querying TXT records.
type Resolver struct {
	client         *dns.Client
	timeout        time.Duration
	ipv6Check      sync.Once
	ipv6Works      bool
	dnssecValidate bool
}

// SetDNSSECValidate enables or disables DNSSEC validation checking.
// When enabled, the DO bit is set on every TXT query and the AD bit
// in the response is recorded in Result.DNSSECValid.
func (r *Resolver) SetDNSSECValidate(v bool) {
	r.dnssecValidate = v
}

// DNSQuerier is an interface for DNS querying, enabling mock injection.
type DNSQuerier interface {
	QueryTXT(ctx context.Context, domain, server string) Result
	DiscoverAuthoritativeNS(ctx context.Context, domain string) ([]string, error)
}

// New creates a new Resolver with the given timeout.
func New(timeout time.Duration) *Resolver {
	return &Resolver{
		client: &dns.Client{
			Timeout: timeout,
			Net:     "udp",
		},
		timeout: timeout,
	}
}

// checkIPv6Connectivity tests if IPv6 DNS queries work in this environment.
// It attempts a simple DNS query over IPv6 to a well-known resolver.
func (r *Resolver) checkIPv6Connectivity() bool {
	r.ipv6Check.Do(func() {
		// Try a simple query to Google's IPv6 DNS server
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		client := &dns.Client{
			Net:     "udp6",
			Timeout: 2 * time.Second,
		}

		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn("google.com"), dns.TypeA)

		_, _, err := client.ExchangeContext(ctx, msg, "[2001:4860:4860::8888]:53")
		r.ipv6Works = (err == nil)
	})
	return r.ipv6Works
}

// QueryTXT queries a single resolver for TXT records of the given domain.
func (r *Resolver) QueryTXT(ctx context.Context, domain, server string) Result {
	start := time.Now()
	result := Result{
		Resolver: server,
	}

	if !strings.Contains(server, ":") {
		server = server + ":53"
	}

	msg := new(dns.Msg)
	fqdn := dns.Fqdn(domain)
	msg.SetQuestion(fqdn, dns.TypeTXT)
	msg.RecursionDesired = true

	if r.dnssecValidate {
		// DO bit: ask the resolver to return DNSSEC records and set AD on validated responses.
		msg.SetEdns0(4096, true)
	}

	resp, _, err := r.client.ExchangeContext(ctx, msg, server)
	result.Latency = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		logger.Debug("query | server=%s domain=%s latency=%s error=%v", server, domain, result.Latency.Round(time.Millisecond), err)
		return result
	}

	if resp == nil {
		result.Error = "nil response"
		logger.Debug("query | server=%s domain=%s latency=%s error=nil_response", server, domain, result.Latency.Round(time.Millisecond))
		return result
	}

	result.Authoritative = resp.Authoritative
	if r.dnssecValidate {
		result.DNSSECChecked = true
		// AD (Authenticated Data) bit: set by the resolver when it validated
		// the complete DNSSEC chain for this response.
		result.DNSSECValid = resp.AuthenticatedData
		logger.Debug("dnssec | server=%s ad=%v", server, result.DNSSECValid)
	}

	for _, rr := range resp.Answer {
		if txt, ok := rr.(*dns.TXT); ok {
			value := strings.Join(txt.Txt, "")
			result.Values = append(result.Values, value)
			result.TTL = rr.Header().Ttl
		}
	}

	result.Found = len(result.Values) > 0
	logger.Debug("query | server=%s domain=%s latency=%s found=%v values=%v ttl=%d",
		server, domain, result.Latency.Round(time.Millisecond), result.Found, result.Values, result.TTL)
	return result
}

// DiscoverAuthoritativeNS discovers authoritative nameservers for a domain.
func (r *Resolver) DiscoverAuthoritativeNS(ctx context.Context, domain string) ([]string, error) {
	// Walk up the domain to find the zone's NS records
	parts := strings.Split(strings.TrimSuffix(domain, "."), ".")
	for i := 0; i < len(parts)-1; i++ {
		zone := strings.Join(parts[i:], ".")
		nss, err := r.lookupNS(ctx, zone)
		if err == nil && len(nss) > 0 {
			return nss, nil
		}
	}
	return nil, fmt.Errorf("could not discover authoritative NS for %s", domain)
}

func (r *Resolver) lookupNS(ctx context.Context, zone string) ([]string, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(zone), dns.TypeNS)
	msg.RecursionDesired = true

	resp, _, err := r.client.ExchangeContext(ctx, msg, "8.8.8.8:53")
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response")
	}

	var servers []string
	seen := make(map[string]bool)
	for _, rr := range resp.Answer {
		if ns, ok := rr.(*dns.NS); ok {
			host := strings.TrimSuffix(ns.Ns, ".")
			ips, err := net.LookupHost(host)
			if err != nil {
				continue
			}
			// Use all IPv4 addresses for each NS hostname so that for anycast
			// providers (e.g. Cloudflare) we cover multiple PoPs. This avoids
			// being locked to one PoP that may lag in internal zone replication.
			// Include IPv6 addresses if IPv6 connectivity is available.
			ipv6Available := r.checkIPv6Connectivity()
			for _, addr := range ips {
				parsed := net.ParseIP(addr)
				if parsed == nil {
					continue // invalid IP
				}

				// Always include IPv4 addresses
				if parsed.To4() != nil {
					if !seen[addr] {
						seen[addr] = true
						servers = append(servers, addr)
					}
					continue
				}

				// Include IPv6 addresses only if connectivity check passed
				if ipv6Available && parsed.To16() != nil {
					if !seen[addr] {
						seen[addr] = true
						servers = append(servers, addr)
					}
				}
			}
		}
	}
	return servers, nil
}

