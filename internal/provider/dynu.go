package provider

import (
	"fmt"

	"github.com/libdns/dynu"
)

// newDynu configures a Dynu DNS provider.
// api_token is required. own_domain may be set to override the Dynu subdomain.
func newDynu(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("dynu: 'api_token' is required in credentials file")
	}
	return &dynu.Provider{
		APIToken:  creds["api_token"],
		OwnDomain: creds["own_domain"],
	}, nil
}

