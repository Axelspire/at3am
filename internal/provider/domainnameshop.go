package provider

import (
	"fmt"

	"github.com/libdns/domainnameshop"
)

// newDomainNameShop configures a Domainnameshop (formerly Domeneshop) provider.
// api_token and api_secret are required (create an API token at
// https://domainnameshop.com/user/apikeys).
func newDomainNameShop(creds map[string]string) (DNSProvider, error) {
	if creds["api_token"] == "" {
		return nil, fmt.Errorf("domainnameshop: 'api_token' is required in credentials file")
	}
	if creds["api_secret"] == "" {
		return nil, fmt.Errorf("domainnameshop: 'api_secret' is required in credentials file")
	}
	return &domainnameshop.Provider{
		APIToken:  creds["api_token"],
		APISecret: creds["api_secret"],
	}, nil
}

