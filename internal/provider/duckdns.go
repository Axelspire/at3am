package provider

import (
	"fmt"

	"github.com/libdns/duckdns"
)

// newDuckDNS configures a DuckDNS provider.
// api_token is required. override_domain may be set when using CNAME delegation
// from a non-DuckDNS domain. resolver defaults to "8.8.8.8:53" when empty.
func newDuckDNS(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("duckdns: 'api_token' is required in credentials file")
	}
	return &duckdns.Provider{
		APIToken:       creds["api_token"],
		OverrideDomain: creds["override_domain"],
		Resolver:       creds["resolver"],
	}, nil
}

