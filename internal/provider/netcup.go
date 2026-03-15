package provider

import (
	"fmt"

	"github.com/libdns/netcup"
)

func newNetcup(creds map[string]string) (DNSProvider, error) {
	if creds["customer_number"] == "" {
		return nil, fmt.Errorf("netcup: 'customer_number' is required in credentials file")
	}
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("netcup: 'api_key' is required in credentials file")
	}
	if creds["api_password"] == "" {
		return nil, fmt.Errorf("netcup: 'api_password' is required in credentials file")
	}
	return &netcup.Provider{
		CustomerNumber: creds["customer_number"],
		APIKey:         creds["api_key"],
		APIPassword:    creds["api_password"],
	}, nil
}

