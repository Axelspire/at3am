package provider

import (
	"fmt"

	"github.com/libdns/digitalocean"
)

func newDigitalOcean(creds map[string]string) (DNSProvider, error) {
	token := creds["auth_token"]
	if token == "" {
		return nil, fmt.Errorf("digitalocean: 'auth_token' is required in credentials file")
	}
	return &digitalocean.Provider{APIToken: token}, nil
}

