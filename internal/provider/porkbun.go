package provider

import (
	"fmt"

	"github.com/libdns/porkbun"
)

func newPorkbun(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("porkbun: 'api_key' is required in credentials file")
	}
	if creds["api_secret_key"] == "" {
		return nil, fmt.Errorf("porkbun: 'api_secret_key' is required in credentials file")
	}
	return &porkbun.Provider{
		APIKey:       creds["api_key"],
		APISecretKey: creds["api_secret_key"],
	}, nil
}

