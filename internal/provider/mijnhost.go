package provider

import (
	"fmt"

	"github.com/libdns/mijnhost"
)

// newMijnHost configures a mijn.host DNS provider.
// api_key is required (obtain from https://mijn.host/api/doc).
// base_uri is optional and defaults to https://mijn.host/api/v2/ when empty.
func newMijnHost(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("mijnhost: 'api_key' is required in credentials file")
	}
	return &mijnhost.Provider{
		ApiKey: creds["api_key"],
		// BaseUri nil → default production endpoint
	}, nil
}

