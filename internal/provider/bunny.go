package provider

import (
	"fmt"

	"github.com/libdns/bunny"
)

func newBunny(creds map[string]string) (DNSProvider, error) {
	if creds["access_key"] == "" {
		return nil, fmt.Errorf("bunny: 'access_key' is required in credentials file (Bunny.net API key)")
	}
	return &bunny.Provider{AccessKey: creds["access_key"]}, nil
}

