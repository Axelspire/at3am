package provider

import (
	"fmt"

	"github.com/libdns/infomaniak"
)

func newInfomaniak(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("infomaniak: 'api_token' is required in credentials file")
	}
	return &infomaniak.Provider{APIToken: creds["api_token"]}, nil
}

