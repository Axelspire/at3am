package provider

import (
	"fmt"

	"github.com/libdns/gcore"
)

// newGCore configures a G-Core Labs DNS provider.
// api_key is the permanent API key from https://accounts.gcorelabs.com/profile/api-tokens
func newGCore(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("gcore: 'api_key' is required in credentials file")
	}
	return &gcore.Provider{APIKey: creds["api_key"]}, nil
}

