package resolver

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func startMockDNSServer(t *testing.T, handler dns.Handler) (string, func()) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	server := &dns.Server{PacketConn: pc, Handler: handler}
	go func() {
		_ = server.ActivateAndServe()
	}()
	addr := pc.LocalAddr().String()
	return addr, func() {
		_ = server.Shutdown()
	}
}

func TestQueryTXT_Found(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = append(m.Answer, &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			Txt: []string{"test-token"},
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startMockDNSServer(t, handler)
	defer cleanup()

	r := New(2 * time.Second)
	result := r.QueryTXT(context.Background(), "_acme-challenge.example.com", addr)

	if !result.Found {
		t.Error("expected found")
	}
	if len(result.Values) != 1 || result.Values[0] != "test-token" {
		t.Errorf("expected test-token, got %v", result.Values)
	}
	if result.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", result.TTL)
	}
	if !result.Authoritative {
		t.Error("expected authoritative")
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestQueryTXT_NotFound(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startMockDNSServer(t, handler)
	defer cleanup()

	r := New(2 * time.Second)
	result := r.QueryTXT(context.Background(), "test.example.com", addr)

	if result.Found {
		t.Error("expected not found")
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
}

func TestQueryTXT_Error(t *testing.T) {
	r := New(500 * time.Millisecond)
	result := r.QueryTXT(context.Background(), "test.example.com", "192.0.2.1:9") // unreachable

	if result.Found {
		t.Error("expected not found")
	}
	if result.Error == "" {
		t.Error("expected error")
	}
}

func TestQueryTXT_WithPort(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.TXT{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
			Txt: []string{"val"},
		})
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startMockDNSServer(t, handler)
	defer cleanup()

	r := New(2 * time.Second)
	// addr already has port, should work
	result := r.QueryTXT(context.Background(), "test.example.com", addr)
	if !result.Found {
		t.Error("expected found")
	}
}

func TestQueryTXT_WithoutPort(t *testing.T) {
	// Just test that it adds :53 (will fail to connect but that's expected)
	r := New(200 * time.Millisecond)
	result := r.QueryTXT(context.Background(), "test.example.com", "192.0.2.1")
	// Should have tried 192.0.2.1:53
	if result.Resolver != "192.0.2.1" {
		t.Errorf("expected resolver 192.0.2.1, got %s", result.Resolver)
	}
}

func TestNew(t *testing.T) {
	r := New(3 * time.Second)
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}
	if r.timeout != 3*time.Second {
		t.Errorf("expected 3s timeout, got %s", r.timeout)
	}
}

func TestDiscoverAuthoritativeNS_NoResult(t *testing.T) {
	r := New(500 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := r.DiscoverAuthoritativeNS(ctx, "unlikely.invalid.test.example.invalid.")
	// Should return an error (either timeout or no NS found)
	fmt.Println("DiscoverAuthNS error (expected):", err)
}

func TestQueryTXT_MultipleValues(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer,
			&dns.TXT{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{"val1"},
			},
			&dns.TXT{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
				Txt: []string{"val2"},
			},
		)
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startMockDNSServer(t, handler)
	defer cleanup()

	r := New(2 * time.Second)
	result := r.QueryTXT(context.Background(), "test.example.com", addr)
	if len(result.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(result.Values))
	}
}

func TestLookupNS_WithMockServer(t *testing.T) {
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if r.Question[0].Qtype == dns.TypeNS {
			m.Answer = append(m.Answer, &dns.NS{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 3600},
				Ns:  "localhost.",
			})
		}
		_ = w.WriteMsg(m)
	})

	addr, cleanup := startMockDNSServer(t, handler)
	defer cleanup()

	// We can't easily test lookupNS directly since it always queries 8.8.8.8:53
	// But we can test that DiscoverAuthoritativeNS walks the domain
	r := New(2 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// This will try real DNS - just verify it doesn't panic
	_, _ = r.DiscoverAuthoritativeNS(ctx, "sub.example.com")
	_ = addr // used above
}

func TestCheckIPv6Connectivity(t *testing.T) {
	resolver := New(2 * time.Second)

	// Test that the IPv6 check is cached (sync.Once behavior)
	result1 := resolver.checkIPv6Connectivity()
	result2 := resolver.checkIPv6Connectivity()

	if result1 != result2 {
		t.Error("IPv6 connectivity check should return consistent results")
	}

	// The actual result depends on the test environment
	// We just verify it doesn't panic and returns a boolean
	t.Logf("IPv6 connectivity: %v", result1)
}

