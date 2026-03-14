package provider

import (
	"fmt"

	"github.com/libdns/unifi"
)

// newUnifi configures a Ubiquiti UniFi local DNS provider.
// api_key, site_id, and base_url are required.
// base_url is the local UniFi controller URL including the integration path, e.g.:
//
//	https://192.168.1.1/proxy/network/integration/v1
func newUnifi(creds map[string]string) (DNSProvider, error) {
	if creds["api_key"] == "" {
		return nil, fmt.Errorf("unifi: 'api_key' is required in credentials file")
	}
	if creds["base_url"] == "" {
		return nil, fmt.Errorf("unifi: 'base_url' is required (e.g. https://192.168.1.1/proxy/network/integration/v1)")
	}
	return &unifi.Provider{
		ApiKey:  creds["api_key"],
		SiteId:  creds["site_id"],
		BaseUrl: creds["base_url"],
	}, nil
}

