package provider

import (
	"fmt"

	"github.com/libdns/autodns"
)

// newAutoDNS configures an InterNetX AutoDNS provider.
// username and password are required. endpoint defaults to the AutoDNS production
// API (https://api.autodns.com/v1) when empty. context is the AutoDNS context
// number (default "4" for reseller-API context).
func newAutoDNS(creds map[string]string) (DNSProvider, error) {
	if creds["username"] == "" {
		return nil, fmt.Errorf("autodns: 'username' is required in credentials file")
	}
	if creds["password"] == "" {
		return nil, fmt.Errorf("autodns: 'password' is required in credentials file")
	}
	return &autodns.Provider{
		Username: creds["username"],
		Password: creds["password"],
		Endpoint: creds["endpoint"],
		Context:  creds["context"],
		Primary:  creds["primary"],
	}, nil
}

