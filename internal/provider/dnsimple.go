package provider

import (
	"fmt"

	"github.com/libdns/dnsimple"
)

// newDNSimple configures a DNSimple provider.
// api_access_token is required. account_id is optional (autodetected via API).
// api_url may be set to the sandbox URL for testing (https://api.sandbox.dnsimple.com).
func newDNSimple(creds map[string]string) (DNSProvider, error) {
	if creds["api_access_token"] == "" {
		return nil, fmt.Errorf("dnsimple: 'api_access_token' is required in credentials file")
	}
	return &dnsimple.Provider{
		APIAccessToken: creds["api_access_token"],
		AccountID:      creds["account_id"],
		APIURL:         creds["api_url"],
	}, nil
}

