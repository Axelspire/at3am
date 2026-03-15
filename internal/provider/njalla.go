package provider

import (
	"fmt"

	"github.com/libdns/njalla"
)

// newNjalla configures a Njalla DNS provider.
// api_token is required (generate at https://njal.la/settings/api/).
func newNjalla(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("njalla: 'api_token' is required in credentials file")
	}
	return &njalla.Provider{APIToken: creds["api_token"]}, nil
}

