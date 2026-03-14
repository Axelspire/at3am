package provider

import (
	"fmt"

	"github.com/libdns/hetzner"
)

func newHetzner(creds map[string]string) (DNSProvider, error) {
	token := creds["auth_api_token"]
	if token == "" {
		return nil, fmt.Errorf("hetzner: 'auth_api_token' is required in credentials file")
	}
	return &hetzner.Provider{AuthAPIToken: token}, nil
}

