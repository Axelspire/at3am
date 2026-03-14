package provider

import (
	"fmt"

	"github.com/libdns/cloudflare"
)

func newCloudflare(creds map[string]string) (DNSProvider, error) {
	token := creds["api_token"]
	if token == "" {
		return nil, fmt.Errorf("cloudflare: 'api_token' is required in credentials file")
	}
	return &cloudflare.Provider{APIToken: token}, nil
}

