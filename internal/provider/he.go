package provider

import (
	"fmt"

	"github.com/libdns/he"
)

// newHurricaneElectric configures a Hurricane Electric (dns.he.net) provider.
// api_key is the DDNS key set per-record in the HE DNS console.
func newHurricaneElectric(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("he: 'api_key' is required in credentials file (set per-record in the HE DNS console)")
	}
	return &he.Provider{APIKey: creds["api_key"]}, nil
}

