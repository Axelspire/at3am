package provider

import (
	"fmt"

	"github.com/libdns/simplydotcom"
)

// newSimplyDotCom configures a Simply.com DNS provider.
// account_name and api_key are required. base_url may be overridden for staging.
func newSimplyDotCom(creds map[string]string) (DNSProvider, error) {
	if creds["account_name"] == "" {
		return nil, fmt.Errorf("simplydotcom: 'account_name' is required in credentials file")
	}
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("simplydotcom: 'api_key' is required in credentials file")
	}
	return &simplydotcom.Provider{
		AccountName: creds["account_name"],
		APIKey:      creds["api_key"],
		BaseURL:     creds["base_url"],
	}, nil
}

