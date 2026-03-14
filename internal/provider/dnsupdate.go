package provider

import (
	"fmt"

	"github.com/libdns/dnsupdate"
)

// newDNSUpdate configures a DNS UPDATE (RFC 2136 with extensions) provider.
// addr is the host:port of the authoritative nameserver.
// tsig, if set, must be in the format "algorithm:name:base64secret", e.g.:
//
//	"hmac-sha256.:mykey.:base64secret=="
//
// Leave tsig empty for unauthenticated updates (LAN / split-horizon only).
// Note: this provider differs from rfc2136 in that it is maintained by the
// libdns core team and may support additional DNS UPDATE extensions.
func newDNSUpdate(creds map[string]string) (DNSProvider, error) {
	if creds["addr"] == "" {
		return nil, fmt.Errorf("dnsupdate: 'addr' is required in credentials file (e.g. \"ns1.example.com:53\")")
	}
	return &dnsupdate.Provider{
		Addr: creds["addr"],
		TSIG: creds["tsig"],
	}, nil
}

