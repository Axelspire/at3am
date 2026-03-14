package provider

import (
	"fmt"

	"github.com/libdns/acmeproxy"
)

// newACMEProxy configures an ACME-Proxy provider.
// address is the URL of the ACME-Proxy server (e.g. "https://acmeproxy.example.com").
// username and password are optional when the proxy is deployed without auth.
func newACMEProxy(creds map[string]string) (DNSProvider, error) {
	if creds["address"] == "" {
		return nil, fmt.Errorf("acmeproxy: 'address' is required (e.g. https://acmeproxy.example.com)")
	}
	return &acmeproxy.Provider{
		Endpoint: creds["address"],
		Credentials: acmeproxy.Credentials{
			Username: creds["username"],
			Password: creds["password"],
		},
	}, nil
}

