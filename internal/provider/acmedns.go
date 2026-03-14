package provider

import (
	"fmt"

	"github.com/libdns/acmedns"
)

// newACMEDNS configures an ACME-DNS provider (https://github.com/joohoi/acme-dns).
// server_url, username, password, and subdomain are required.
// These values come from a prior /register call to the ACME-DNS server.
func newACMEDNS(creds map[string]string) (DNSProvider, error) {
	if creds["server_url"] == "" {
		return nil, fmt.Errorf("acmedns: 'server_url' is required (e.g. https://auth.acme-dns.io)")
	}
	if creds["username"] == "" {
		return nil, fmt.Errorf("acmedns: 'username' is required in credentials file")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("acmedns: 'password' is required in credentials file")
	}
	if creds["subdomain"] == "" {
		return nil, fmt.Errorf("acmedns: 'subdomain' is required in credentials file")
	}
	return &acmedns.Provider{
		ServerURL: creds["server_url"],
		Username:  creds["username"],
		Password:  creds["password"],
		Subdomain: creds["subdomain"],
	}, nil
}

