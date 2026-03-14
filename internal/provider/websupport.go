package provider

import (
	"fmt"

	"github.com/libdns/websupport/websupport"
)

// newWebsupport configures a WebSupport.sk DNS provider.
// api_key and api_secret are required (manage at https://admin.websupport.sk/en/auth/apiKey).
// service_id is optional; when set, overrides the service lookup for a domain.
func newWebsupport(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("websupport: 'api_key' is required in credentials file")
	}
	if creds["api_secret"] == "" {
		return nil, fmt.Errorf("websupport: 'api_secret' is required in credentials file")
	}
	return &websupport.Provider{
		APIKey:    creds["api_key"],
		APISecret: creds["api_secret"],
		APIBase:   creds["api_base"],
		ServiceID: creds["service_id"],
	}, nil
}

