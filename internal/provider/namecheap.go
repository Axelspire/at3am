package provider

import (
	"fmt"

	"github.com/libdns/namecheap"
)

func newNamecheap(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("namecheap: 'api_key' is required in credentials file")
	}
	if creds["user"] == "" {
		return nil, fmt.Errorf("namecheap: 'user' is required in credentials file")
	}
	return &namecheap.Provider{
		APIKey:      creds["api_key"],
		User:        creds["user"],
		APIEndpoint: creds["api_endpoint"], // empty → production endpoint
		ClientIP:    creds["client_ip"],    // empty → auto-discover
	}, nil
}

