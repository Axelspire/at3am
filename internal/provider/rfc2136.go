package provider

import (
	"fmt"

	"github.com/libdns/rfc2136"
)

// newRFC2136 configures a generic DNS UPDATE (RFC 2136) provider.
// server must be host:port of the authoritative name server (e.g. "ns1.example.com:53").
// key_name, key_alg, and key are the TSIG credentials; omit all three for
// unauthenticated updates (only safe on trusted networks).
func newRFC2136(creds map[string]string) (DNSProvider, error) {
	if creds["server"] == "" {
		return nil, fmt.Errorf("rfc2136: 'server' is required in credentials file (e.g. \"ns1.example.com:53\")")
	}
	return &rfc2136.Provider{
		Server:  creds["server"],
		KeyName: creds["key_name"],
		KeyAlg:  creds["key_alg"],
		Key:     creds["key"],
	}, nil
}

