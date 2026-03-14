package provider

import (
	"fmt"

	"github.com/libdns/namesilo"
)

func newNameSilo(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("namesilo: 'api_token' is required in credentials file")
	}
	return &namesilo.Provider{APIToken: creds["api_token"]}, nil
}

